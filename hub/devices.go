package hub

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
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
	return principal{key: "device/" + d.ID, models: d.Models, budget: d.Budget, forceBackground: d.BackgroundOnly}
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

// Devices lists paired devices.
func (h *Hub) Devices(ctx context.Context) ([]Device, error) { return h.listDevices(ctx) }

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

// Revoke removes a device at once: its key is deleted, its mailbox released
// and any open session closed.
func (h *Hub) Revoke(ctx context.Context, id string) error {
	h.deviceMu.Lock()
	d, err := h.getDevice(ctx, id)
	if err != nil {
		h.deviceMu.Unlock()
		return err
	}
	err = h.opts.Store.Delete(ctx, "device/"+id)
	_ = h.opts.Store.Delete(ctx, "devicelast/"+id)
	h.deviceMu.Unlock()
	if err != nil {
		return err
	}
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
		if link != nil {
			_ = link.Release(ctx, mid)
		}
	}
	h.opts.Logf("hub: device %s removed", d.ID)
	return nil
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
	// returns their answer. Required.
	Confirm func(ctx context.Context, r PairRequest) (bool, error)
}

// Pairing is a pending pairing.
type Pairing struct {
	// Code is the pairing text for the device (also what a QR code holds).
	// It is a secret: show it only on the hub's own screen.
	Code    string
	Expires time.Time
	done    chan pairResult
}

type pairResult struct {
	dev Device
	err error
}

// Wait blocks until the device pairs, the owner refuses, or it expires.
func (p *Pairing) Wait(ctx context.Context) (Device, error) {
	select {
	case r := <-p.done:
		return r.dev, r.err
	case <-ctx.Done():
		return Device{}, ctx.Err()
	}
}

type pairing struct {
	id       relayproto.ID
	psk      []byte
	sh       [32]byte
	opts     PairOptions
	expires  time.Time
	attempts int
	inFlight bool
	done     chan pairResult
	finished bool
}

// Pair starts a pairing. The hub must be connected to its relay.
func (h *Hub) Pair(ctx context.Context, opts PairOptions) (*Pairing, error) {
	if opts.Confirm == nil {
		return nil, errors.New("hub: PairOptions.Confirm is required")
	}
	h.mu.Lock()
	link := h.link
	h.mu.Unlock()
	if link == nil || !link.Up() {
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
		expires: h.opts.Now().Add(opts.TTL), done: make(chan pairResult, 1)}
	h.mu.Lock()
	if len(h.pairings) >= 8 {
		h.mu.Unlock()
		return nil, &omnillm.Error{Code: omnillm.CodeBusy, Msg: omnillm.MsgBusyQueue}
	}
	h.pairings[id] = p
	h.mu.Unlock()
	if err := link.Claim(ctx, id, p.sh); err != nil {
		h.endPairing(p, &omnillm.Error{Code: omnillm.CodeHubOffline, Msg: omnillm.MsgRelayUnavailable})
		return nil, err
	}
	text := proto.PairingCode{Relay: h.opts.RelayURL, Mailbox: id.String(), Secret: secretB,
		HubKey: h.key.Public, PSK: append([]byte{}, p.psk...), Expires: p.expires.Unix()}.Encode()
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
		}
	}()
	return &Pairing{Code: text, Expires: p.expires, done: p.done}, nil
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
	code := e2e.SAS(sess.ChannelBinding())
	cctx, cancel := context.WithDeadline(ctx, p.expires)
	ok, err := p.opts.Confirm(cctx, PairRequest{DeviceName: name, Code: code})
	cancel()
	if err != nil || !ok {
		b, _ := proto.Marshal(proto.Envelope{T: proto.TPairDenied})
		_ = sess.Send(ctx, b)
		h.endPairing(p, &omnillm.Error{Code: omnillm.CodePairingDenied, Msg: omnillm.MsgPairingDenied})
		return omnillm.ErrPairingDenied
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
	// Re-check under the lock: the same device key may not be paired twice.
	devs, err = h.listDevices(ctx)
	for _, other := range devs {
		if subtle.ConstantTimeCompare(other.PublicKey, d.PublicKey) == 1 {
			err = omnillm.ErrPairingFailed
		}
	}
	if err == nil {
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

// ---------- app tokens (for programs that cannot embed the library) ----------

// App is a program allowed to use the hub's local API with a token.
type App struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	TokenHash []byte    `json:"token_hash"`
	Models    []string  `json:"models,omitempty"`
	Budget    int64     `json:"budget_micro_usd"`
	Created   time.Time `json:"created"`
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
	a := App{ID: newDeviceID(), Name: cleanName(name), TokenHash: tokenHash(tok), Models: models, Budget: budget, Created: h.opts.Now().UTC()}
	if err := store.PutJSON(ctx, h.opts.Store, "app/"+a.ID, a); err != nil {
		return App{}, "", err
	}
	return a, tok, nil
}

// Apps lists app tokens (hashes only).
func (h *Hub) Apps(ctx context.Context) ([]App, error) {
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
	return out, nil
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
	apps, err := h.Apps(ctx)
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
	return principal{key: "app/" + a.ID, models: a.Models, budget: a.Budget, forceBackground: true}
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
	var out []omnillm.ModelInfo
	for _, m := range all {
		if len(a.Models) == 0 || contains(a.Models, m.ID) {
			out = append(out, m)
		}
	}
	return out, nil
}
