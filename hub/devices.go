package hub

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	omnillm "github.com/wensuper/OmniLLM"
	"github.com/wensuper/OmniLLM/internal/e2e"
	"github.com/wensuper/OmniLLM/internal/proto"
	"github.com/wensuper/OmniLLM/internal/relayproto"
	"github.com/wensuper/OmniLLM/store"
)

// Device is a paired device. Only its public key is kept; the hub never has
// the device's private key or its mailbox secret (only a hash of it).
type Device struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	PublicKey  []byte    `json:"public_key"`
	Mailbox    string    `json:"mailbox"`     // hex
	SecretHash []byte    `json:"secret_hash"` // relayproto.SecretHash
	Models     []string  `json:"models,omitempty"`
	Budget     int64     `json:"budget_micro_usd"`
	Created    time.Time `json:"created"`
	LastSeen   time.Time `json:"last_seen,omitempty"`
	Expires    time.Time `json:"expires,omitempty"`
	// BackgroundOnly devices (servers, scripts) are always treated as
	// Background, whatever they claim; a device a person uses may say a
	// person is waiting.
	BackgroundOnly bool `json:"background_only,omitempty"`
	// Paused by the owner: its requests are refused (it can still list
	// its models, so its screen can say why) until turned on again.
	Paused bool `json:"paused,omitempty"`
	// Online: the device has a session open right now (Devices fills it
	// in; it is never stored).
	Online bool `json:"-"`
}

func (d Device) mailbox() (relayproto.ID, [32]byte, error) {
	var sh [32]byte
	id, err := relayproto.ParseID(d.Mailbox)
	if err != nil || len(d.SecretHash) != 32 {
		return id, sh, errors.New("hub: bad device record")
	}
	copy(sh[:], d.SecretHash)
	return id, sh, nil
}

func (d Device) principal() principal {
	p := principal{key: "device/" + d.ID, models: d.Models, budget: d.Budget, forceBackground: d.BackgroundOnly}
	if d.Paused {
		p.paused = omnillm.MsgDevicePaused
	}
	return p
}

// touch records when a device was last seen, under its own key, so it can
// never re-create a removed device or overwrite the owner's settings.
func (h *Hub) touch(ctx context.Context, id string) {
	h.deviceMu.Lock()
	defer h.deviceMu.Unlock()
	if _, err := h.getDevice(ctx, id); err != nil {
		return // removed meanwhile: leave nothing behind
	}
	_ = store.PutJSON(ctx, h.opts.Store, "devicelast/"+id, h.opts.Now().UTC())
}

func (h *Hub) listDevices(ctx context.Context) ([]Device, error) {
	keys, err := h.opts.Store.List(ctx, "device/")
	if err != nil {
		return nil, err
	}
	var out []Device
	for _, k := range keys {
		var d Device
		if store.GetJSON(ctx, h.opts.Store, k, &d) == nil {
			_ = store.GetJSON(ctx, h.opts.Store, "devicelast/"+d.ID, &d.LastSeen)
			out = append(out, d)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Created.Before(out[j].Created) })
	return out, nil
}

// Devices lists paired devices, each with whether it is online now.
// Devices are named uniquely; two that share a name from before that rule
// (or from two hubs' records merged) are told apart once, here, the later
// one becoming "iPhone (2)", as Apps does.
func (h *Hub) Devices(ctx context.Context) ([]Device, error) {
	h.deviceMu.Lock()
	devs, err := h.listDevices(ctx)
	names := make([]string, len(devs))
	for i, d := range devs {
		names[i] = d.Name
	}
	for i, d := range devs {
		n, ok := renumber(names, i)
		if !ok {
			continue
		}
		// Rename the stored record only: the last-seen time stays under
		// its own key.
		if rec, gerr := h.getDevice(ctx, d.ID); gerr == nil {
			rec.Name = n
			if store.PutJSON(ctx, h.opts.Store, "device/"+d.ID, rec) == nil {
				devs[i].Name, names[i] = n, n
			}
		}
	}
	h.deviceMu.Unlock()
	h.mu.Lock()
	for i := range devs {
		devs[i].Online = findSession(h.devSessions, devs[i].ID) != nil
	}
	h.mu.Unlock()
	return devs, err
}

func (h *Hub) deviceByMailbox(ctx context.Context, id relayproto.ID) (Device, bool) {
	devs, _ := h.listDevices(ctx)
	for _, d := range devs {
		if d.Mailbox == id.String() {
			return d, true
		}
	}
	return Device{}, false
}

func (h *Hub) getDevice(ctx context.Context, id string) (Device, error) {
	var d Device
	if id == "" || strings.ContainsAny(id, "/ ") {
		return d, omnillm.ErrNotFound
	}
	if err := store.GetJSON(ctx, h.opts.Store, "device/"+id, &d); err != nil {
		return d, &omnillm.Error{Code: omnillm.CodeNotFound, Msg: omnillm.MsgNotFound}
	}
	return d, nil
}

// SetDevice changes a device's models, budget or expiry.
func (h *Hub) SetDevice(ctx context.Context, id string, models []string, budget int64, expires time.Time) error {
	h.deviceMu.Lock()
	defer h.deviceMu.Unlock()
	d, err := h.getDevice(ctx, id)
	if err != nil {
		return err
	}
	if budget <= 0 {
		return &omnillm.Error{Code: omnillm.CodeBadRequest, Msg: "bad_model", Detail: "budget must be positive (no unlimited budgets)"}
	}
	d.Models, d.Budget, d.Expires = models, budget, expires
	return store.PutJSON(ctx, h.opts.Store, "device/"+id, d)
}

// NoLimit as a device's Budget: no daily limit of its own; the hub's daily
// total (Options.GlobalBudget) still applies.
const NoLimit int64 = -1

// UpdateDevice renames a device, changes its daily budget (micro-USD, or
// NoLimit), pauses or resumes it, and/or sets the models it may use (an
// empty list: all, including models added later). A nil field is left as
// it is. A name another device already has gets a number, as at pairing.
func (h *Hub) UpdateDevice(ctx context.Context, id string, name *string, budget *int64, paused *bool, models *[]string) (Device, error) {
	h.deviceMu.Lock()
	defer h.deviceMu.Unlock()
	d, err := h.getDevice(ctx, id)
	if err != nil {
		return d, err
	}
	if budget != nil {
		if *budget != NoLimit && *budget <= 0 {
			return d, &omnillm.Error{Code: omnillm.CodeBadRequest, Msg: omnillm.MsgBadBudget}
		}
		d.Budget = *budget
	}
	if name != nil {
		n := cleanName(*name)
		devs, err := h.listDevices(ctx)
		if err != nil {
			return d, err
		}
		d.Name = uniqueName(n, deviceNames(devs, d.ID))
	}
	if paused != nil {
		d.Paused = *paused
	}
	if models != nil {
		ms, err := h.checkModels(ctx, *models, false, d.Models)
		if err != nil {
			return d, err
		}
		d.Models = ms
	}
	if err := store.PutJSON(ctx, h.opts.Store, "device/"+id, d); err != nil {
		return d, err
	}
	if d.Paused {
		// Off means off now: a reply in progress stops too, and the device
		// is told it was paused.
		h.stopDeviceChats(id, &omnillm.Error{Code: omnillm.CodeDevicePaused, Msg: omnillm.MsgDevicePaused})
	}
	return d, nil
}

// Revoke removes a device at once: its key is deleted, its mailbox released
// and any open session closed.
func (h *Hub) Revoke(ctx context.Context, id string) error {
	d, err := h.revokeRecord(ctx, id)
	if err != nil {
		return err
	}
	h.dropDevice(ctx, d)
	return nil
}

// revokeRecord replaces the device's record with its tombstone. Split from
// dropDevice so a device removing itself hears "removed" before its
// session is closed.
func (h *Hub) revokeRecord(ctx context.Context, id string) (Device, error) {
	h.deviceMu.Lock()
	defer h.deviceMu.Unlock()
	d, err := h.getDevice(ctx, id)
	if err != nil {
		return d, err
	}
	// The tombstone first: the device is never without both records.
	if err = h.putTombstone(ctx, d); err == nil {
		err = h.opts.Store.Delete(ctx, "device/"+id)
	}
	_ = h.opts.Store.Delete(ctx, "devicelast/"+id)
	return d, err
}

// dropDevice ends a removed device's open session and removes its kept
// conversations.
func (h *Hub) dropDevice(ctx context.Context, d Device) {
	h.chats.Forget("device/" + d.ID)
	// The mailbox stays claimed (revoked.go): when the device comes back
	// it is told it was removed. Its open session ends now.
	mid, _, err := d.mailbox()
	if err == nil {
		h.mu.Lock()
		s := h.sessions[mid]
		delete(h.sessions, mid)
		link := h.link
		h.mu.Unlock()
		if s != nil {
			s.close()
		}
		// Have the relay drop the device's connection too: its next
		// request then reconnects at once and is told it was removed,
		// instead of waiting for an answer on a connection nobody serves.
		if link != nil {
			_ = link.Kick(ctx, mid)
		}
	}
	h.opts.Logf("hub: device %s removed", d.ID)
}

// ---------- pairing ----------

// PairRequest is what the owner confirms on the hub.
type PairRequest struct {
	DeviceName string
	Code       string // six digits, also shown on the device
}

// PairOptions configures one pairing.
type PairOptions struct {
	Name   string   // suggested name; the device may send its own
	Models []string // empty: all
	Budget int64    // micro-USD/day; 0: default
	TTL    time.Duration
	// BackgroundOnly for servers and scripts: their calls never count as a
	// person waiting.
	BackgroundOnly bool
	// Confirm is called once the device has proved it holds the pairing
	// code. The host shows the code and device name to the owner and
	// returns their answer. Required, unless ConfirmName is set.
	Confirm func(ctx context.Context, r PairRequest) (bool, error)
	// ConfirmName is Confirm for a host that also lets the owner name the
	// device while confirming; "" keeps the name it asked with. Used
	// instead of Confirm when set.
	ConfirmName func(ctx context.Context, r PairRequest) (ok bool, name string, err error)
}

// ErrPairingDeviceLeft ends a pairing whose device side went away (it
// cancelled or lost its connection) while the owner was being asked: not
// a refusal, and the host can say so.
var ErrPairingDeviceLeft = &omnillm.Error{Code: omnillm.CodePairingFailed, Msg: omnillm.MsgPairingFailed, Detail: "the device left"}

// Pairing is a pending pairing.
type Pairing struct {
	// Code is the pairing text for the device (also what a QR code holds).
	// It is a secret: show it only on the hub's own screen.
	Code string
	// Link is the pairing link for phones and tablets: scanned with the
	// camera it opens the web client, which pairs from it. Equally secret.
	Link    string
	Expires time.Time
	done    chan pairResult
	cancel  func()
}

type pairResult struct {
	dev Device
	err error
}

// Wait blocks until the device pairs, the owner refuses, or it expires.
// If ctx ends first the pairing is cancelled: its code stops working and
// its slot is freed.
func (p *Pairing) Wait(ctx context.Context) (Device, error) {
	select {
	case r := <-p.done:
		return r.dev, r.err
	case <-ctx.Done():
		p.Cancel()
		return Device{}, ctx.Err()
	}
}

// Cancel ends a pending pairing; the code stops working. It does nothing
// once the pairing has finished.
func (p *Pairing) Cancel() {
	if p.cancel != nil {
		p.cancel()
	}
}

type pairing struct {
	id        relayproto.ID
	psk       []byte
	sh        [32]byte
	opts      PairOptions
	expires   time.Time
	attempts  int
	inFlight  bool
	done      chan pairResult
	cancelled chan struct{}
	finished  bool
}

// Pair starts a pairing. The hub must be connected to its relay.
func (h *Hub) Pair(ctx context.Context, opts PairOptions) (*Pairing, error) {
	if opts.Confirm == nil && opts.ConfirmName == nil {
		return nil, errors.New("hub: PairOptions.Confirm is required")
	}
	h.mu.Lock()
	link := h.link
	h.mu.Unlock()
	if link == nil {
		return nil, &omnillm.Error{Code: omnillm.CodeHubOffline, Msg: omnillm.MsgRelayUnavailable}
	}
	// Just after a relay restart the link may still be reconnecting; give
	// it a moment instead of failing the owner's command.
	wctx, wcancel := context.WithTimeout(ctx, 15*time.Second)
	err := link.WaitUp(wctx)
	wcancel()
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, &omnillm.Error{Code: omnillm.CodeHubOffline, Msg: omnillm.MsgRelayUnavailable}
	}
	if opts.TTL <= 0 || opts.TTL > 10*time.Minute {
		opts.TTL = 5 * time.Minute
	}
	if opts.Budget <= 0 {
		opts.Budget = h.opts.DefaultBudget
	}
	var id relayproto.ID
	copy(id[:], e2e.RandomBytes(relayproto.IDLen))
	secretB := e2e.RandomBytes(32)
	p := &pairing{id: id, psk: e2e.RandomBytes(32), sh: relayproto.SecretHash(secretB), opts: opts,
		expires: h.opts.Now().Add(opts.TTL), done: make(chan pairResult, 1), cancelled: make(chan struct{})}
	h.mu.Lock()
	if len(h.pairings) >= 8 {
		h.mu.Unlock()
		return nil, &omnillm.Error{Code: omnillm.CodeBusy, Msg: omnillm.MsgBusyQueue}
	}
	h.pairings[id] = p
	h.mu.Unlock()
	if err := link.Claim(ctx, id, p.sh); err != nil {
		h.endPairing(p, &omnillm.Error{Code: omnillm.CodeHubOffline, Msg: omnillm.MsgRelayUnavailable})
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, &omnillm.Error{Code: omnillm.CodeHubOffline, Msg: omnillm.MsgRelayUnavailable, Detail: err.Error()}
	}
	pc := proto.PairingCode{Relay: h.opts.RelayURL, Mailbox: id.String(), Secret: secretB,
		HubKey: h.key.Public, PSK: append([]byte{}, p.psk...), Expires: p.expires.Unix()}
	text := pc.Encode()
	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		t := time.NewTimer(time.Until(p.expires))
		defer t.Stop()
		select {
		case <-t.C:
			h.endPairing(p, &omnillm.Error{Code: omnillm.CodePairingFailed, Msg: omnillm.MsgPairingExpired})
		case <-h.ctx.Done():
			h.endPairing(p, h.ctx.Err())
		case <-p.cancelled:
			h.endPairing(p, &omnillm.Error{Code: omnillm.CodePairingFailed, Msg: omnillm.MsgPairingExpired})
		}
	}()
	var once sync.Once
	return &Pairing{Code: text, Link: pc.Link(h.opts.WebAppURL), Expires: p.expires, done: p.done,
		cancel: func() {
			once.Do(func() {
				close(p.cancelled)
				// Free the slot now, not when the timer goroutine runs.
				h.endPairing(p, &omnillm.Error{Code: omnillm.CodePairingFailed, Msg: omnillm.MsgPairingExpired})
			})
		}}, nil
}

// endPairing finishes a pending pairing once. On failure the mailbox is
// released, so the pairing code is dead.
func (h *Hub) endPairing(p *pairing, err error) {
	h.mu.Lock()
	if p.finished {
		h.mu.Unlock()
		return
	}
	p.finished = true
	delete(h.pairings, p.id)
	link := h.link
	clear(p.psk)
	h.mu.Unlock()
	if err != nil && link != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = link.Release(ctx, p.id)
		cancel()
	}
	p.done <- pairResult{err: err}
}

// uniqueName returns name, or "name (2)" with the lowest number from 2 up
// that none of names already is (case does not count). A number already
// used the older way ("iPhone 2") is skipped too, so two devices never
// read as the same one; names given that way stay as they are.
func uniqueName(name string, names []string) string {
	taken := make(map[string]bool, len(names))
	for _, n := range names {
		taken[strings.ToLower(n)] = true
	}
	if !taken[strings.ToLower(name)] {
		return name
	}
	for i := 2; ; i++ {
		n := fmt.Sprintf("%s (%d)", name, i)
		if !taken[strings.ToLower(n)] && !taken[strings.ToLower(fmt.Sprintf("%s %d", name, i))] {
			return n
		}
	}
}

// renumber says what names[i] becomes when an earlier name is the same
// (case does not count): the lowest "name (N)" that no other name is,
// later ones included, so a name the owner chose ("iPhone (2)") is never
// taken by a clash. ok is false when names[i] stays as it is.
func renumber(names []string, i int) (string, bool) {
	clash := false
	for _, n := range names[:i] {
		if strings.EqualFold(n, names[i]) {
			clash = true
			break
		}
	}
	if !clash {
		return names[i], false
	}
	others := make([]string, 0, len(names)-1)
	others = append(others, names[:i]...)
	others = append(others, names[i+1:]...)
	return uniqueName(names[i], others), true
}

// deviceNames lists the devices' names, leaving out the one with id skip.
func deviceNames(devs []Device, skip string) []string {
	var out []string
	for _, d := range devs {
		if d.ID != skip {
			out = append(out, d.Name)
		}
	}
	return out
}

// appNames lists the apps' names, leaving out the one with id skip.
func appNames(apps []App, skip string) []string {
	var out []string
	for _, a := range apps {
		if a.ID != skip {
			out = append(out, a.Name)
		}
	}
	return out
}

// cleanName keeps a device name printable and short.
func cleanName(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsPrint(r) && !unicode.Is(unicode.Bidi_Control, r) {
			b.WriteRune(r)
		}
		if b.Len() >= 64 {
			break
		}
	}
	out := strings.TrimSpace(b.String())
	if out == "" {
		out = "device"
	}
	return out
}

func newDeviceID() string {
	return strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(e2e.RandomBytes(10)))
}

// finishPairing runs on the hub after the device's handshake with the
// pairing secret succeeded and its pair.hello arrived.
func (h *Hub) finishPairing(ctx context.Context, p *pairing, sess *e2e.Session, hello proto.Envelope) error {
	name := cleanName(hello.Name)
	if p.opts.Name != "" {
		name = cleanName(p.opts.Name)
	}
	// base is the name before any number: it is numbered again under the
	// lock right before the device is stored, since another device may
	// have been paired with the same name while the owner was asked.
	base := name
	// Two phones both called "iPhone" can't be told apart in the device
	// list (and the wrong one gets removed): the second is "iPhone (2)".
	if devs, err := h.listDevices(ctx); err == nil {
		name = uniqueName(name, deviceNames(devs, ""))
	}
	code := e2e.SAS(sess.ChannelBinding())
	cctx, cancel := context.WithDeadline(ctx, p.expires)
	go func() {
		select {
		case <-p.cancelled: // the owner's command went away mid-question
			cancel()
		case <-cctx.Done():
		}
	}()
	var (
		ok    bool
		given string
		err   error
	)
	if p.opts.ConfirmName != nil {
		ok, given, err = p.opts.ConfirmName(cctx, PairRequest{DeviceName: name, Code: code})
	} else {
		ok, err = p.opts.Confirm(cctx, PairRequest{DeviceName: name, Code: code})
	}
	expired := cctx.Err() == context.DeadlineExceeded
	cancel()
	if (err != nil || !ok) && errors.Is(context.Cause(ctx), errLinkDown) {
		// This Mac lost its link to the relay while the owner was asked:
		// the device did not leave; remote access is down.
		h.endPairing(p, &omnillm.Error{Code: omnillm.CodeHubOffline, Msg: omnillm.MsgRelayUnavailable})
		return omnillm.ErrPairingFailed
	}
	if (err != nil || !ok) && ctx.Err() != nil && !expired {
		// The device side ended while the owner was looking (it cancelled,
		// or its connection dropped): not the owner's refusal.
		h.endPairing(p, ErrPairingDeviceLeft)
		return ErrPairingDeviceLeft
	}
	if err != nil && expired {
		b, _ := proto.Marshal(proto.Envelope{T: proto.TPairDenied})
		_ = sess.Send(ctx, b)
		h.endPairing(p, &omnillm.Error{Code: omnillm.CodePairingFailed, Msg: omnillm.MsgPairingExpired})
		return omnillm.ErrPairingFailed
	}
	if err != nil || !ok {
		b, _ := proto.Marshal(proto.Envelope{T: proto.TPairDenied})
		_ = sess.Send(ctx, b)
		h.endPairing(p, &omnillm.Error{Code: omnillm.CodePairingDenied, Msg: omnillm.MsgPairingDenied})
		return omnillm.ErrPairingDenied
	}
	if g := cleanName(given); strings.TrimSpace(given) != "" && g != name {
		// The owner's own name for it (not just the one proposed); still
		// never a second "iPhone".
		base = g
	}
	// The same device key may not be paired twice.
	devs, _ := h.listDevices(ctx)
	for _, d := range devs {
		if subtle.ConstantTimeCompare(d.PublicKey, sess.PeerStatic()) == 1 {
			h.endPairing(p, &omnillm.Error{Code: omnillm.CodePairingFailed, Msg: omnillm.MsgPairingFailed})
			return omnillm.ErrPairingFailed
		}
	}
	// Claim the pairing exactly once: it may have expired or been cancelled
	// while the owner was looking at the code.
	h.mu.Lock()
	if p.finished || h.opts.Now().After(p.expires) {
		h.mu.Unlock()
		h.endPairing(p, &omnillm.Error{Code: omnillm.CodePairingFailed, Msg: omnillm.MsgPairingExpired})
		return omnillm.ErrPairingFailed
	}
	p.finished = true
	delete(h.pairings, p.id)
	clear(p.psk)
	h.mu.Unlock()
	// Replace the mailbox secret that was in the pairing code: whoever saw
	// the code can no longer even reach the mailbox. The new secret travels
	// only inside the encrypted session.
	newSecret := e2e.RandomBytes(32)
	newHash := relayproto.SecretHash(newSecret)
	h.mu.Lock()
	link := h.link
	h.mu.Unlock()
	d := Device{ID: newDeviceID(), Name: name, PublicKey: sess.PeerStatic(), Mailbox: p.id.String(),
		SecretHash: newHash[:], Models: p.opts.Models, Budget: p.opts.Budget, Created: h.opts.Now().UTC(),
		BackgroundOnly: p.opts.BackgroundOnly}
	h.deviceMu.Lock()
	// Re-check under the lock: the same device key may not be paired twice,
	// and the name is numbered against the devices there are now.
	devs, err = h.listDevices(ctx)
	for _, other := range devs {
		if subtle.ConstantTimeCompare(other.PublicKey, d.PublicKey) == 1 {
			err = omnillm.ErrPairingFailed
		}
	}
	if err == nil {
		d.Name = uniqueName(base, deviceNames(devs, ""))
		name = d.Name
		err = store.PutJSON(ctx, h.opts.Store, "device/"+d.ID, d)
	}
	h.deviceMu.Unlock()
	if err == nil && link != nil {
		err = link.Claim(ctx, p.id, newHash)
	}
	if err != nil {
		h.deviceMu.Lock()
		_ = h.opts.Store.Delete(ctx, "device/"+d.ID)
		h.deviceMu.Unlock()
		if link != nil {
			_ = link.Release(ctx, p.id)
		}
		p.done <- pairResult{err: err}
		return err
	}
	b, _ := proto.Marshal(proto.Envelope{T: proto.TPairOK, DeviceID: d.ID, Name: name, Secret: newSecret})
	if err := sess.Send(ctx, b); err != nil {
		// The device saved nothing and the code's secret is already
		// replaced: undo, so no ghost device is left behind.
		h.deviceMu.Lock()
		_ = h.opts.Store.Delete(ctx, "device/"+d.ID)
		h.deviceMu.Unlock()
		if link != nil {
			_ = link.Release(ctx, p.id)
		}
		p.done <- pairResult{err: &omnillm.Error{Code: omnillm.CodePairingFailed, Msg: omnillm.MsgPairingFailed}}
		return err
	}
	p.done <- pairResult{dev: d}
	h.opts.Logf("hub: device %s paired", d.ID)
	return nil
}

// checkModels is a device's or app's model list as the owner set it: every
// id must exist, each once; nil (all models) for an empty list. Apps make
// background calls only, so chat-only models are not newly theirs. had is
// the list before: what is on it stays without the checks (whether a model
// can be used is decided again at every call).
func (h *Hub) checkModels(ctx context.Context, ids []string, forApp bool, had []string) ([]string, error) {
	var out []string
	for _, id := range ids {
		if contains(had, id) {
			// Already on its list: kept as it is, even when it can't be
			// used now (its source removed, or chat-only for an app), so
			// saving other changes never drops what the owner didn't
			// switch off.
			if !contains(out, id) {
				out = append(out, id)
			}
			continue
		}
		m, err := h.getModel(ctx, id)
		if err != nil {
			return nil, &omnillm.Error{Code: omnillm.CodeBadRequest, Msg: omnillm.MsgUnknownModel}
		}
		if forApp && (m.HumanOnly || h.subscription(m.Provider)) {
			return nil, &omnillm.Error{Code: omnillm.CodeModelNotAllowed, Msg: omnillm.MsgHumanOnly}
		}
		if !contains(out, id) {
			out = append(out, id)
		}
	}
	return out, nil
}

// modelsFor is the model list a device is answered with: refused while it
// is paused (its screen keeps saying so, and sees when it is on again),
// without the models the owner keeps off devices, with the owner's names.
func (h *Hub) modelsFor(ctx context.Context, d Device) ([]omnillm.ModelInfo, error) {
	if d.Paused {
		return nil, &omnillm.Error{Code: omnillm.CodeDevicePaused, Msg: omnillm.MsgDevicePaused}
	}
	ms, err := h.deviceModels(ctx, d)
	if err != nil {
		return nil, err
	}
	h.freshLogins()
	ms = h.forDevices(ctx, ms)
	if len(ms) >= 2 {
		// Something to choose between: Auto first.
		ms = append([]omnillm.ModelInfo{autoInfo()}, ms...)
	}
	return ms, nil
}

// ---------- the owner's choices per model ----------

// modelPref is the owner's choice for one model: their own name for it,
// and whether devices are offered it at all. It is kept apart from the
// model record, so re-adding a model source or a newly reported version
// never undoes it.
type modelPref struct {
	Name   string `json:"name,omitempty"`
	Hidden bool   `json:"hidden,omitempty"`
	NoAuto bool   `json:"no_auto,omitempty"` // Auto never chooses it
}

func (h *Hub) modelPrefs(ctx context.Context) map[string]modelPref {
	keys, _ := h.opts.Store.List(ctx, "modelpref/")
	out := make(map[string]modelPref, len(keys))
	for _, k := range keys {
		var mp modelPref
		if store.GetJSON(ctx, h.opts.Store, k, &mp) == nil {
			out[strings.TrimPrefix(k, "modelpref/")] = mp
		}
	}
	return out
}

// SetModelPref renames a model for people ("" goes back to the name its
// source gives it) and/or shows or hides it on devices. A nil field is
// left as it is. A hidden model is not offered to devices, and a device
// that still asks for it is told it is gone; the host and apps keep it.
func (h *Hub) SetModelPref(ctx context.Context, id string, name *string, shown *bool) error {
	if _, err := h.getModel(ctx, id); err != nil {
		return err
	}
	h.deviceMu.Lock()
	defer h.deviceMu.Unlock()
	var mp modelPref
	_ = store.GetJSON(ctx, h.opts.Store, "modelpref/"+id, &mp)
	if name != nil {
		mp.Name = ""
		if strings.TrimSpace(*name) != "" {
			mp.Name = cleanName(*name)
		}
	}
	if shown != nil {
		mp.Hidden = !*shown
	}
	if mp == (modelPref{}) {
		return h.opts.Store.Delete(ctx, "modelpref/"+id)
	}
	return store.PutJSON(ctx, h.opts.Store, "modelpref/"+id, mp)
}

// ModelView is a model as the owner sees it: with the owner's name for
// it, and whether it is kept off devices.
type ModelView struct {
	omnillm.ModelInfo
	Hidden bool `json:"hidden,omitempty"`
	// NoAuto: kept out of Auto's choices.
	NoAuto bool `json:"no_auto,omitempty"`
	// SourceOff: its model source is switched off, so no one gets it now.
	SourceOff bool `json:"source_off,omitempty"`
}

// OwnerModels lists every model for the owner's own screens.
func (h *Hub) OwnerModels(ctx context.Context) ([]ModelView, error) {
	all, err := h.Models(ctx)
	if err != nil {
		return nil, err
	}
	prefs := h.modelPrefs(ctx)
	off := h.sourcesOff(ctx)
	out := make([]ModelView, 0, len(all))
	for _, m := range all {
		mp := prefs[m.ID]
		if mp.Name != "" {
			m.DisplayName, m.Renamed = mp.Name, true
		}
		out = append(out, ModelView{ModelInfo: m, Hidden: mp.Hidden, NoAuto: mp.NoAuto, SourceOff: off[m.Provider]})
	}
	return out, nil
}

// forDevices is a model list as devices get it: the owner's names, and
// without the models kept off devices or whose source is switched off.
func (h *Hub) forDevices(ctx context.Context, ms []omnillm.ModelInfo) []omnillm.ModelInfo {
	prefs := h.modelPrefs(ctx)
	off := h.sourcesOff(ctx)
	out := ms[:0:0]
	for _, m := range ms {
		mp := prefs[m.ID]
		if mp.Hidden || off[m.Provider] {
			continue
		}
		if mp.Name != "" {
			m.DisplayName, m.Renamed = mp.Name, true
		}
		out = append(out, m)
	}
	return out
}

// ---------- app tokens (for programs that cannot embed the library) ----------

// App is a program allowed to use the hub's local API with a token.
type App struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	TokenHash []byte    `json:"token_hash"`
	Models    []string  `json:"models,omitempty"`
	Budget    int64     `json:"budget_micro_usd"`
	Created   time.Time `json:"created"`
	// Paused by the owner, as for a device.
	Paused bool `json:"paused,omitempty"`
}

const appTokenPrefix = "omni_app_"

func tokenHash(t string) []byte {
	s := sha256.Sum256([]byte("omnillm/app-token/1\x00" + t))
	return s[:]
}

// AddApp creates an app token. The token is returned once and only its
// hash is stored. App calls are always Background.
func (h *Hub) AddApp(ctx context.Context, name string, models []string, budget int64) (App, string, error) {
	if budget <= 0 {
		budget = h.opts.DefaultBudget
	}
	tok := appTokenPrefix + hex.EncodeToString(e2e.RandomBytes(32))
	h.deviceMu.Lock()
	defer h.deviceMu.Unlock()
	apps, _ := h.listApps(ctx)
	a := App{ID: newDeviceID(), Name: uniqueName(cleanName(name), appNames(apps, "")), TokenHash: tokenHash(tok), Models: models, Budget: budget, Created: h.opts.Now().UTC()}
	if err := store.PutJSON(ctx, h.opts.Store, "app/"+a.ID, a); err != nil {
		return App{}, "", err
	}
	return a, tok, nil
}

// Apps lists app tokens (hashes only). Apps are named uniquely, as
// devices are; two apps that share a name from before that rule (or from
// two hubs' records merged) are told apart once, here, the later one
// becoming "my-script (2)".
func (h *Hub) Apps(ctx context.Context) ([]App, error) {
	h.deviceMu.Lock()
	defer h.deviceMu.Unlock()
	apps, err := h.listApps(ctx)
	if err != nil {
		return nil, err
	}
	names := make([]string, len(apps))
	for i, a := range apps {
		names[i] = a.Name
	}
	for i, a := range apps {
		if n, ok := renumber(names, i); ok {
			a.Name = n
			if store.PutJSON(ctx, h.opts.Store, "app/"+a.ID, a) == nil {
				apps[i], names[i] = a, n
			}
		}
	}
	return apps, nil
}

// listApps reads the app records, oldest first.
func (h *Hub) listApps(ctx context.Context) ([]App, error) {
	keys, err := h.opts.Store.List(ctx, "app/")
	if err != nil {
		return nil, err
	}
	var out []App
	for _, k := range keys {
		var a App
		if store.GetJSON(ctx, h.opts.Store, k, &a) == nil {
			out = append(out, a)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Created.Before(out[j].Created) })
	return out, nil
}

// UpdateApp renames an app, changes its daily budget (micro-USD, or
// NoLimit), pauses or resumes it and/or sets its models, as UpdateDevice
// does for devices. Apps never get chat-only models.
func (h *Hub) UpdateApp(ctx context.Context, id string, name *string, budget *int64, paused *bool, models *[]string) (App, error) {
	var a App
	if id == "" || strings.ContainsAny(id, "/ ") {
		return a, omnillm.ErrNotFound
	}
	h.deviceMu.Lock()
	defer h.deviceMu.Unlock()
	if err := store.GetJSON(ctx, h.opts.Store, "app/"+id, &a); err != nil {
		return a, &omnillm.Error{Code: omnillm.CodeNotFound, Msg: omnillm.MsgNotFound}
	}
	if budget != nil {
		if *budget != NoLimit && *budget <= 0 {
			return a, &omnillm.Error{Code: omnillm.CodeBadRequest, Msg: omnillm.MsgBadBudget}
		}
		a.Budget = *budget
	}
	if name != nil {
		apps, _ := h.listApps(ctx)
		a.Name = uniqueName(cleanName(*name), appNames(apps, a.ID))
	}
	if paused != nil {
		a.Paused = *paused
	}
	if models != nil {
		ms, err := h.checkModels(ctx, *models, true, a.Models)
		if err != nil {
			return a, err
		}
		a.Models = ms
	}
	return a, store.PutJSON(ctx, h.opts.Store, "app/"+id, a)
}

// RevokeApp deletes an app token.
func (h *Hub) RevokeApp(ctx context.Context, id string) error {
	if id == "" || strings.ContainsAny(id, "/ ") {
		return omnillm.ErrNotFound
	}
	return h.opts.Store.Delete(ctx, "app/"+id)
}

func (h *Hub) appByToken(ctx context.Context, tok string) (App, error) {
	if !strings.HasPrefix(tok, appTokenPrefix) || len(tok) > 200 {
		return App{}, &omnillm.Error{Code: omnillm.CodeInvalidToken, Msg: omnillm.MsgInvalidToken}
	}
	want := tokenHash(tok)
	apps, err := h.listApps(ctx)
	if err != nil {
		return App{}, err
	}
	found := App{}
	match := 0
	for _, a := range apps {
		if subtle.ConstantTimeCompare(a.TokenHash, want) == 1 {
			found = a
			match = 1
		}
	}
	if match == 0 {
		return App{}, &omnillm.Error{Code: omnillm.CodeInvalidToken, Msg: omnillm.MsgInvalidToken}
	}
	return found, nil
}

func (a App) principal() principal {
	p := principal{key: "app/" + a.ID, models: a.Models, budget: a.Budget, forceBackground: true}
	if a.Paused {
		p.paused = omnillm.MsgAppPaused
	}
	return p
}

// CompleteAsApp runs a request for an app token (the local API). App calls
// are always Background, whatever the request says.
func (h *Hub) CompleteAsApp(ctx context.Context, token string, req omnillm.Request) (omnillm.Stream, error) {
	a, err := h.appByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	return h.complete(ctx, a.principal(), req)
}

// ModelsForApp lists what an app token may use, with locality as seen from
// this machine.
func (h *Hub) ModelsForApp(ctx context.Context, token string) ([]omnillm.ModelInfo, error) {
	a, err := h.appByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	all, err := h.Models(ctx)
	if err != nil {
		return nil, err
	}
	off := h.sourcesOff(ctx)
	var out []omnillm.ModelInfo
	for _, m := range all {
		// App tokens are Background: models kept for people are not theirs.
		if m.HumanOnly || off[m.Provider] {
			continue
		}
		if len(a.Models) == 0 || contains(a.Models, m.ID) {
			out = append(out, m)
		}
	}
	// The owner's names; hiding a model is about devices, not apps.
	prefs := h.modelPrefs(ctx)
	for i := range out {
		if n := prefs[out[i].ID].Name; n != "" {
			out[i].DisplayName, out[i].Renamed = n, true
		}
	}
	return out, nil
}
