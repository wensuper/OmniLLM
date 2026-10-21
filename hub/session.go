package hub

import (
	"context"
	"crypto/subtle"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	omnillm "github.com/wensuper/OmniLLM"
	"github.com/wensuper/OmniLLM/internal/e2e"
	"github.com/wensuper/OmniLLM/internal/proto"
	"github.com/wensuper/OmniLLM/internal/provider"
	"github.com/wensuper/OmniLLM/internal/relaylink"
	"github.com/wensuper/OmniLLM/internal/relayproto"
)

// mailboxConn is one device's virtual connection over the shared relay
// link. It implements e2e.MsgConn.
type mailboxConn struct {
	id     relayproto.ID
	gen    uint64 // the relay's attachment generation
	link   *relaylink.HubLink
	in     chan []byte
	ctx    context.Context
	cancel context.CancelCauseFunc
	once   sync.Once
}

func (c *mailboxConn) ReadMessage(ctx context.Context) ([]byte, error) {
	select {
	case b := <-c.in:
		return b, nil
	case <-c.ctx.Done():
		return nil, io.EOF
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *mailboxConn) WriteMessage(ctx context.Context, b []byte) error {
	if c.ctx.Err() != nil {
		return io.EOF
	}
	return c.link.Send(ctx, c.id, b)
}

func (c *mailboxConn) Close() error { c.close(); return nil }

func (c *mailboxConn) close() { c.closeCause(nil) }

// closeCause ends the connection and says why, for whoever is still
// working on it (a pairing whose owner is being asked).
func (c *mailboxConn) closeCause(err error) { c.once.Do(func() { c.cancel(err) }) }

// errLinkDown ends every session when this Mac's own link to the relay
// drops: the device did not leave, the Mac lost remote access.
var errLinkDown = errors.New("hub: the relay link dropped")

func gen(b []byte) uint64 {
	if len(b) != 8 {
		return 0
	}
	return binary.BigEndian.Uint64(b)
}

// dispatch routes relay events to per-mailbox sessions.
func (h *Hub) dispatch(link *relaylink.HubLink) {
	for ev := range link.Events() {
		switch ev.Type {
		case relayproto.Open:
			h.openSession(link, ev.Mailbox, gen(ev.Payload))
		case relayproto.Data:
			h.mu.Lock()
			c := h.sessions[ev.Mailbox]
			h.mu.Unlock()
			if c == nil {
				continue
			}
			select {
			case c.in <- ev.Payload:
			default:
				// A device that floods its mailbox is cut off, once.
				h.mu.Lock()
				if h.sessions[ev.Mailbox] == c {
					delete(h.sessions, ev.Mailbox)
				}
				h.mu.Unlock()
				c.close()
				go func(id relayproto.ID) {
					ctx, cancel := context.WithTimeout(h.ctx, 5*time.Second)
					defer cancel()
					_ = link.Kick(ctx, id)
				}(ev.Mailbox)
			}
		case relayproto.Closed:
			// Ignore a Closed for an older attachment that arrives after a
			// newer Open (a quick reconnect).
			h.mu.Lock()
			c := h.sessions[ev.Mailbox]
			if c != nil && c.gen == gen(ev.Payload) {
				delete(h.sessions, ev.Mailbox)
			} else {
				c = nil
			}
			h.mu.Unlock()
			if c != nil {
				c.close()
			}
		case relayproto.LinkDown:
			// The relay dropped every device with the link.
			h.mu.Lock()
			old := h.sessions
			h.sessions = map[relayproto.ID]*mailboxConn{}
			h.mu.Unlock()
			for _, c := range old {
				c.closeCause(errLinkDown)
			}
		case relayproto.ClaimFail:
			h.opts.Logf("hub: the relay refused a mailbox claim")
		}
	}
}

func (h *Hub) openSession(link *relaylink.HubLink, id relayproto.ID, g uint64) {
	ctx, cancel := context.WithCancelCause(h.ctx)
	c := &mailboxConn{id: id, gen: g, link: link, in: make(chan []byte, 64), ctx: ctx, cancel: cancel}
	h.mu.Lock()
	if old := h.sessions[id]; old != nil {
		old.close()
	}
	h.sessions[id] = c
	p := h.pairings[id]
	h.mu.Unlock()
	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		defer func() {
			c.close()
			h.mu.Lock()
			if h.sessions[id] == c {
				delete(h.sessions, id)
			}
			h.mu.Unlock()
		}()
		if p != nil {
			h.servePairing(ctx, c, p)
			return
		}
		h.serveDevice(ctx, c)
	}()
}

func (h *Hub) servePairing(ctx context.Context, c *mailboxConn, p *pairing) {
	// One handshake at a time per pairing, and at most three in total: a
	// second, simultaneous handshake (someone who saw the code) must never
	// race the real device to the owner's confirmation.
	h.mu.Lock()
	if p.finished || p.inFlight {
		h.mu.Unlock()
		return
	}
	p.attempts++
	if p.attempts > 3 {
		h.mu.Unlock()
		h.endPairing(p, &omnillm.Error{Code: omnillm.CodePairingFailed, Msg: omnillm.MsgPairingFailed})
		return
	}
	p.inFlight = true
	psk := append([]byte{}, p.psk...)
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		p.inFlight = false
		h.mu.Unlock()
	}()
	sess, err := e2e.Accept(ctx, c, c.id[:], h.key, psk, nil)
	clear(psk)
	if err != nil {
		return
	}
	defer sess.Close()
	hctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	b, err := sess.Recv(hctx)
	cancel()
	if err != nil {
		// A wrong pairing secret fails right here: the first message does
		// not decrypt.
		return
	}
	env, err := proto.Unmarshal(b)
	if err != nil || env.T != proto.TPairHello {
		return
	}
	_ = h.finishPairing(ctx, p, sess, env)
}

// maxChatsPerDevice bounds concurrent requests from one device.
const maxChatsPerDevice = 4

func (h *Hub) serveDevice(ctx context.Context, c *mailboxConn) {
	d, ok := h.deviceByMailbox(ctx, c.id)
	if !ok {
		if t, ok := h.tombstoneByMailbox(ctx, c.id); ok {
			h.serveRemoved(ctx, c, t)
		}
		return
	}
	sess, err := e2e.Accept(ctx, c, c.id[:], h.key, nil, func(peer []byte) error {
		if subtle.ConstantTimeCompare(peer, d.PublicKey) != 1 {
			return errors.New("hub: unknown device key")
		}
		return nil
	})
	if err != nil {
		return
	}
	defer sess.Close()
	ds := &devSession{h: h, sess: sess, devID: d.ID, cancels: map[uint64]context.CancelCauseFunc{}, tools: map[uint64]chan proto.Envelope{}}
	defer ds.cancelAll()
	registered := false
	defer func() {
		if registered {
			h.mu.Lock()
			h.devSessions = removeSession(h.devSessions, d.ID, ds)
			h.mu.Unlock()
		}
	}()
	for {
		b, err := sess.Recv(ctx)
		if err != nil {
			return
		}
		env, err := proto.Unmarshal(b)
		if err != nil {
			return
		}
		// The device record is re-read for every message: a revoke, an
		// expiry or a change of models or budget applies at once.
		cur, err := h.getDevice(ctx, d.ID)
		if err != nil || (!cur.Expires.IsZero() && h.opts.Now().After(cur.Expires)) {
			h.sendErr(ctx, sess, env.ID, &omnillm.Error{Code: omnillm.CodeInvalidToken, Msg: omnillm.MsgDeviceRevoked})
			return
		}
		if !registered {
			// Only a device that sent a message it could only have
			// encrypted with its keys is reachable for tool calls (a
			// replayed handshake never gets here).
			registered = true
			h.mu.Lock()
			h.devSessions = appendSession(h.devSessions, d.ID, ds)
			h.mu.Unlock()
			h.touch(ctx, d.ID)
		}
		ds.handle(ctx, env, cur)
	}
}

type devSession struct {
	h       *Hub
	sess    *e2e.Session
	devID   string
	mu      sync.Mutex
	closed  bool
	cancels map[uint64]context.CancelCauseFunc
	tools   map[uint64]chan proto.Envelope
	active  atomic.Int32
	nextID  atomic.Uint64
}

func (ds *devSession) send(ctx context.Context, e proto.Envelope) error {
	b, err := proto.Marshal(e)
	if err != nil {
		return err
	}
	return ds.sess.Send(ctx, b)
}

func (h *Hub) sendErr(ctx context.Context, sess *e2e.Session, id uint64, err error) {
	b, _ := proto.Marshal(proto.Envelope{T: proto.TChatError, ID: id, Err: proto.ToWire(err)})
	_ = sess.Send(ctx, b)
}

func (ds *devSession) cancelAll() {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.closed = true
	for _, c := range ds.cancels {
		c(nil)
	}
	for _, ch := range ds.tools {
		close(ch)
	}
	ds.tools = map[uint64]chan proto.Envelope{}
}

func (ds *devSession) handle(ctx context.Context, env proto.Envelope, d Device) {
	switch env.T {
	case proto.TPing:
		_ = ds.send(ctx, proto.Envelope{T: proto.TPong})
	case proto.TModelsList:
		ms, err := ds.h.modelsFor(ctx, d)
		if err != nil {
			ds.h.sendErr(ctx, ds.sess, env.ID, err)
			return
		}
		_ = ds.send(ctx, proto.Envelope{T: proto.TModels, ID: env.ID, Models: ms})
	case proto.TChatStart:
		ds.startChat(ctx, env, d)
	case proto.TDeviceLeave:
		// The person removed this device on the device itself ("remove
		// this device"): the same as the owner removing it on the hub, so
		// it doesn't linger in the device list. The answer goes out before
		// the session is closed.
		dev, err := ds.h.revokeRecord(ctx, d.ID)
		if err != nil {
			ds.h.sendErr(ctx, ds.sess, env.ID, err)
			return
		}
		_ = ds.send(ctx, proto.Envelope{T: proto.TDeviceLeft, ID: env.ID})
		ds.h.dropDevice(ctx, dev)
	case proto.TChatCancel:
		ds.mu.Lock()
		if c := ds.cancels[env.ID]; c != nil {
			c(nil)
		}
		ds.mu.Unlock()
	case proto.TToolResult:
		ds.mu.Lock()
		ch := ds.tools[env.ID]
		delete(ds.tools, env.ID)
		ds.mu.Unlock()
		if ch != nil {
			ch <- env
			close(ch)
		}
	default:
		// Unknown types are ignored (additive protocol).
	}
}

// deviceModels is what a device may use. Everything is Remote from the
// device's point of view: its data leaves the device for the hub.
func (h *Hub) deviceModels(ctx context.Context, d Device) ([]omnillm.ModelInfo, error) {
	all, err := h.Models(ctx)
	if err != nil {
		return nil, err
	}
	var out []omnillm.ModelInfo
	for _, m := range all {
		if len(d.Models) > 0 && !contains(d.Models, m.ID) {
			continue
		}
		if m.HumanOnly && d.BackgroundOnly {
			continue
		}
		m.OnHost = m.Locality == omnillm.Local
		m.Locality = omnillm.Remote
		out = append(out, m)
	}
	return out, nil
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

func (ds *devSession) startChat(ctx context.Context, env proto.Envelope, d Device) {
	id := env.ID
	// Acknowledge on receipt so the device can tell "hub gone" from "model
	// thinking" within seconds.
	_ = ds.send(ctx, proto.Envelope{T: proto.TChatAck, ID: id})
	if env.Req == nil || id == 0 {
		ds.h.sendErr(ctx, ds.sess, id, &omnillm.Error{Code: omnillm.CodeBadRequest, Msg: "bad_messages"})
		return
	}
	if ds.active.Add(1) > maxChatsPerDevice {
		ds.active.Add(-1)
		ds.h.sendErr(ctx, ds.sess, id, &omnillm.Error{Code: omnillm.CodeBusy, Msg: omnillm.MsgBusyQueue, RetryAfter: 5 * time.Second})
		return
	}
	cctx, cancel := context.WithCancelCause(ctx)
	ds.mu.Lock()
	if _, dup := ds.cancels[id]; dup || ds.closed {
		ds.mu.Unlock()
		cancel(nil)
		ds.active.Add(-1)
		return
	}
	ds.cancels[id] = cancel
	ds.mu.Unlock()
	req := *env.Req
	p := d.principal()
	ds.h.wg.Add(1)
	go func() {
		defer ds.h.wg.Done()
		defer func() {
			cancel(nil)
			ds.mu.Lock()
			delete(ds.cancels, id)
			ds.mu.Unlock()
			ds.active.Add(-1)
		}()
		s, err := ds.h.complete(cctx, p, req)
		if err != nil {
			ds.h.sendErr(ctx, ds.sess, id, stopCause(cctx, err))
			return
		}
		defer s.Close()
		ds.relayStream(ctx, cctx, id, s)
	}()
}

// relayStream sends deltas coalesced about every 100 ms (fewer, fuller
// frames: less mobile data, less timing signal for the relay).
func (ds *devSession) relayStream(ctx, cctx context.Context, id uint64, s omnillm.Stream) {
	type item struct {
		text, model string
		err         error
	}
	ch := make(chan item, 64)
	go func() {
		defer close(ch)
		for {
			c, err := s.Recv()
			select {
			case ch <- item{c.Text, c.Model, err}:
			case <-cctx.Done():
				return
			}
			if err != nil {
				return
			}
		}
	}()
	var buf strings.Builder
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	flush := func() bool {
		if buf.Len() == 0 {
			return true
		}
		err := ds.send(ctx, proto.Envelope{T: proto.TChatDelta, ID: id, Text: buf.String()})
		buf.Reset()
		return err == nil
	}
	// stopped: the hub ended the chat (the owner paused this device); the
	// device hears why, after what was already said. A cancel the device
	// asked for itself needs no answer.
	stopped := func() {
		if err := stopCause(cctx, nil); err != nil && flush() {
			ds.h.sendErr(ctx, ds.sess, id, err)
		}
	}
	for {
		select {
		case it, ok := <-ch:
			if !ok {
				stopped()
				return
			}
			if it.model != "" {
				// Which model (and version) is answering, ahead of its words.
				if !flush() || ds.send(ctx, proto.Envelope{T: proto.TChatDelta, ID: id, Model: provider.PrettyModel(it.model)}) != nil {
					return
				}
			}
			buf.WriteString(it.text)
			if it.err != nil {
				if !flush() {
					return
				}
				if errors.Is(it.err, io.EOF) {
					u := s.Usage()
					_ = ds.send(ctx, proto.Envelope{T: proto.TChatEnd, ID: id, Usage: &u, Finish: s.FinishReason()})
				} else {
					ds.h.sendErr(ctx, ds.sess, id, stopCause(cctx, it.err))
				}
				return
			}
			if buf.Len() >= 8<<10 && !flush() {
				return
			}
		case <-tick.C:
			if !flush() {
				return
			}
		case <-cctx.Done():
			stopped()
			return
		}
	}
}

// stopCause is why the hub stopped a chat, when it did (the owner paused
// the device), else err.
func stopCause(cctx context.Context, err error) error {
	var oe *omnillm.Error
	if c := context.Cause(cctx); c != nil && errors.As(c, &oe) && oe.Code == omnillm.CodeDevicePaused {
		return c
	}
	return err
}

// stopChats ends every reply this session is giving, telling the device
// why (err).
func (ds *devSession) stopChats(err error) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	for _, c := range ds.cancels {
		c(err)
	}
}

// stopDeviceChats ends the replies a device is being given right now (it
// was paused): pausing stops a reply in progress, not only the next one.
func (h *Hub) stopDeviceChats(id string, err error) {
	h.mu.Lock()
	var ss []*devSession
	for _, e := range h.devSessions {
		if e.deviceID == id {
			ss = append(ss, e.s)
		}
	}
	h.mu.Unlock()
	for _, s := range ss {
		s.stopChats(err)
	}
}

// DefaultToolTimeout bounds CallTool when ctx has no deadline.
const DefaultToolTimeout = 2 * time.Minute

// CallTool asks the device running the named tool host to run a tool. The
// device decides with its own allowlist; this is the hub-side primitive for
// the AI-app role (design §8.3).
func (h *Hub) CallTool(ctx context.Context, deviceID, tool string, args json.RawMessage) (json.RawMessage, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultToolTimeout)
		defer cancel()
	}
	h.mu.Lock()
	ds := findSession(h.devSessions, deviceID)
	h.mu.Unlock()
	if ds == nil {
		return nil, &omnillm.Error{Code: omnillm.CodeHubOffline, Msg: omnillm.MsgHubOffline}
	}
	if len(args) > 1<<20 || (len(args) > 0 && !json.Valid(args)) || len(tool) == 0 || len(tool) > 128 {
		return nil, &omnillm.Error{Code: omnillm.CodeBadRequest, Msg: "bad_messages"}
	}
	id := ds.nextID.Add(1) | 1<<62
	ch := make(chan proto.Envelope, 1)
	ds.mu.Lock()
	if ds.closed {
		ds.mu.Unlock()
		return nil, &omnillm.Error{Code: omnillm.CodeHubOffline, Msg: omnillm.MsgHubOffline}
	}
	ds.tools[id] = ch
	ds.mu.Unlock()
	defer func() {
		ds.mu.Lock()
		delete(ds.tools, id)
		ds.mu.Unlock()
	}()
	if err := ds.send(ctx, proto.Envelope{T: proto.TToolCall, ID: id, Tool: tool, Args: args}); err != nil {
		return nil, &omnillm.Error{Code: omnillm.CodeHubOffline, Msg: omnillm.MsgHubOffline}
	}
	select {
	case env, ok := <-ch:
		if !ok {
			return nil, &omnillm.Error{Code: omnillm.CodeHubOffline, Msg: omnillm.MsgHubOffline}
		}
		if env.Err != nil {
			return nil, env.Err.Error()
		}
		return env.Result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

type sessionEntry struct {
	deviceID string
	s        *devSession
}

func appendSession(xs []sessionEntry, id string, s *devSession) []sessionEntry {
	return append(xs, sessionEntry{id, s})
}

func removeSession(xs []sessionEntry, id string, s *devSession) []sessionEntry {
	for i, x := range xs {
		if x.deviceID == id && x.s == s {
			return append(xs[:i], xs[i+1:]...)
		}
	}
	return xs
}

func findSession(xs []sessionEntry, id string) *devSession {
	for i := len(xs) - 1; i >= 0; i-- {
		if xs[i].deviceID == id {
			return xs[i].s
		}
	}
	return nil
}
