// Package device is the device role: a program on another machine that
// uses the hub's models through the relay, end-to-end encrypted
// (principle 2). It never talks to a model provider and never holds a
// provider credential (principle 1).
package device

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	omnillm "github.com/wensuper/OmniLLM"
	"github.com/wensuper/OmniLLM/internal/e2e"
	"github.com/wensuper/OmniLLM/internal/proto"
	"github.com/wensuper/OmniLLM/internal/relaylink"
	"github.com/wensuper/OmniLLM/internal/relayproto"
	"github.com/wensuper/OmniLLM/secret"
)

// Tool is something the host lets the hub's conversations call.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Schema      json.RawMessage `json:"schema,omitempty"`
	// ReadOnly tools only read; by default only these may be called.
	ReadOnly bool `json:"read_only"`
}

// ToolHost runs tools for the hub (for NEXA: its MCP tool dispatcher, so
// all of NEXA's own rules and confirmations apply).
type ToolHost interface {
	Tools() []Tool
	Call(ctx context.Context, name string, args json.RawMessage) (json.RawMessage, error)
}

// Options is what the host decides.
type Options struct {
	Secrets   secret.Store      // required
	Transport http.RoundTripper // to the relay; nil: default
	Logf      func(format string, args ...any)
	Tools     ToolHost
	// AllowedTools names the tools the hub may call. Nil: only ReadOnly
	// tools. A name must still be offered by Tools to run.
	AllowedTools []string
	// AckTimeout is how long to wait for the hub to acknowledge a request
	// before reporting it offline (default 3s).
	AckTimeout time.Duration
}

const (
	sKey     = "device/key"
	sHubKey  = "device/hub_key"
	sRelay   = "device/relay"
	sMailbox = "device/mailbox"
	sSecret  = "device/mailbox_secret"
	sID      = "device/id"
	// maxTools bounds tool calls running at once on this device.
	maxTools = 4
)

// Info describes the pairing.
type Info struct {
	DeviceID string
	Name     string
}

func pairFailed() error {
	return &omnillm.Error{Code: omnillm.CodePairingFailed, Msg: omnillm.MsgPairingFailed}
}

// Pair joins a hub using the pairing text shown on the hub. show is called
// with the six-digit code; the owner checks it matches the hub's screen and
// confirms there. Credentials are saved only after the hub confirms.
func Pair(ctx context.Context, opts Options, code, name string, show func(sas string)) (Info, error) {
	if opts.Secrets == nil {
		return Info{}, errors.New("device: Secrets is required")
	}
	pc, err := proto.ParsePairingCode(code, time.Now())
	if err != nil {
		return Info{}, err
	}
	if _, err := relaylink.CheckURL(pc.Relay); err != nil {
		return Info{}, err
	}
	mid, err := relayproto.ParseID(pc.Mailbox)
	if err != nil {
		return Info{}, err
	}
	key, err := e2e.GenerateKey()
	if err != nil {
		return Info{}, err
	}
	conn, err := relaylink.DialMailbox(ctx, pc.Relay, mid, pc.Secret, opts.Transport)
	if err != nil {
		return Info{}, &omnillm.Error{Code: omnillm.CodePairingFailed, Msg: omnillm.MsgPairingExpired}
	}
	defer conn.Close()
	sess, err := e2e.Dial(ctx, conn, mid[:], key, pc.HubKey, pc.PSK)
	if err != nil {
		return Info{}, pairFailed()
	}
	if show != nil {
		show(e2e.SAS(sess.ChannelBinding()))
	}
	hello, _ := proto.Marshal(proto.Envelope{T: proto.TPairHello, Name: name})
	if err := sess.Send(ctx, hello); err != nil {
		return Info{}, pairFailed()
	}
	wctx, cancel := context.WithDeadline(ctx, time.Unix(pc.Expires, 0).Add(30*time.Second))
	defer cancel()
	b, err := sess.Recv(wctx)
	if err != nil {
		return Info{}, pairFailed()
	}
	env, err := proto.Unmarshal(b)
	if err != nil {
		return Info{}, pairFailed()
	}
	if env.T == proto.TPairDenied {
		return Info{}, &omnillm.Error{Code: omnillm.CodePairingDenied, Msg: omnillm.MsgPairingDenied}
	}
	if env.T != proto.TPairOK || env.DeviceID == "" || len(env.Secret) != 32 {
		return Info{}, pairFailed()
	}
	// The hub replaced the mailbox secret from the pairing code; the old
	// one no longer opens the mailbox. sID is written last: it marks the
	// pairing complete.
	saved := []string{}
	for _, kv := range []struct {
		n string
		v []byte
	}{{sKey, key.Marshal()}, {sHubKey, pc.HubKey}, {sRelay, []byte(pc.Relay)}, {sMailbox, []byte(pc.Mailbox)},
		{sSecret, env.Secret}, {sID, []byte(env.DeviceID)}} {
		if err := opts.Secrets.Set(ctx, kv.n, kv.v); err != nil {
			for _, n := range saved {
				_ = opts.Secrets.Delete(ctx, n)
			}
			return Info{}, err
		}
		saved = append(saved, kv.n)
	}
	return Info{DeviceID: env.DeviceID, Name: env.Name}, nil
}

// Paired reports whether this device has a complete saved pairing.
func Paired(ctx context.Context, s secret.Store) bool {
	_, err := s.Get(ctx, sID)
	return err == nil
}

// Forget deletes this device's pairing (sID first, so a partial delete
// never looks paired).
func Forget(ctx context.Context, s secret.Store) error {
	for _, n := range []string{sID, sKey, sHubKey, sRelay, sMailbox, sSecret} {
		if err := s.Delete(ctx, n); err != nil {
			return err
		}
	}
	return nil
}

// Device is an opened, paired device.
type Device struct {
	opts    Options
	key     e2e.KeyPair
	hubKey  []byte
	relay   string
	mailbox relayproto.ID
	secret  []byte
	id      string

	mu     sync.Mutex
	cur    *conn
	closed bool
	dialMu sync.Mutex
	tools  chan struct{}
}

// Open loads the saved pairing. Connecting happens on first use.
func Open(ctx context.Context, opts Options) (*Device, error) {
	if opts.Secrets == nil {
		return nil, errors.New("device: Secrets is required")
	}
	if opts.Logf == nil {
		opts.Logf = func(string, ...any) {}
	}
	if opts.AckTimeout <= 0 {
		opts.AckTimeout = 3 * time.Second
	}
	var err error
	get := func(n string) []byte {
		if err != nil {
			return nil
		}
		var b []byte
		b, err = opts.Secrets.Get(ctx, n)
		return b
	}
	d := &Device{opts: opts, tools: make(chan struct{}, maxTools)}
	d.id = string(get(sID))
	kb := get(sKey)
	d.hubKey = get(sHubKey)
	d.relay = string(get(sRelay))
	mb := string(get(sMailbox))
	d.secret = get(sSecret)
	if err != nil {
		return nil, errors.New("device: not paired")
	}
	if d.key, err = e2e.UnmarshalKey(kb); err != nil {
		return nil, err
	}
	if d.mailbox, err = relayproto.ParseID(mb); err != nil {
		return nil, err
	}
	return d, nil
}

// ID is this device's ID on the hub.
func (d *Device) ID() string { return d.id }

// Close drops the connection.
func (d *Device) Close() error {
	d.mu.Lock()
	d.closed = true
	c := d.cur
	d.cur = nil
	d.mu.Unlock()
	if c != nil {
		c.close()
	}
	return nil
}

func offline() error { return &omnillm.Error{Code: omnillm.CodeHubOffline, Msg: omnillm.MsgHubOffline} }

// waiter receives the messages for one request. Its channel is never
// closed (no send can race a close); dead says the request is over.
type waiter struct {
	ch   chan proto.Envelope
	dead chan struct{}
	once sync.Once
}

func (w *waiter) kill() { w.once.Do(func() { close(w.dead) }) }

// conn is one live session with the hub.
type conn struct {
	d       *Device
	sess    *e2e.Session
	ctx     context.Context
	cancel  context.CancelFunc
	mu      sync.Mutex
	wait    map[uint64]*waiter
	nextID  atomic.Uint64
	retired atomic.Bool
}

func (d *Device) connect(ctx context.Context) (*conn, error) {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return nil, errors.New("device: closed")
	}
	if c := d.cur; c != nil && c.ctx.Err() == nil && !c.sess.Expired() {
		d.mu.Unlock()
		return c, nil
	}
	d.mu.Unlock()

	// One dial at a time; Close and live requests do not wait for it.
	d.dialMu.Lock()
	defer d.dialMu.Unlock()
	d.mu.Lock()
	if c := d.cur; c != nil && c.ctx.Err() == nil && !c.sess.Expired() {
		d.mu.Unlock()
		return c, nil
	}
	d.mu.Unlock()

	mc, err := relaylink.DialMailbox(ctx, d.relay, d.mailbox, d.secret, d.opts.Transport)
	if err != nil {
		return nil, offline()
	}
	sess, err := e2e.Dial(ctx, mc, d.mailbox[:], d.key, d.hubKey, nil)
	if err != nil {
		mc.Close()
		return nil, offline()
	}
	cctx, cancel := context.WithCancel(context.Background())
	c := &conn{d: d, sess: sess, ctx: cctx, cancel: cancel, wait: map[uint64]*waiter{}}
	go c.readLoop()

	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		c.close()
		return nil, errors.New("device: closed")
	}
	old := d.cur
	d.cur = c
	d.mu.Unlock()
	if old != nil {
		// Let replies already streaming on the old session finish; it
		// closes itself when its last request ends.
		old.retire()
	}
	return c, nil
}

func (c *conn) close() {
	c.cancel()
	c.sess.Close()
	c.mu.Lock()
	for _, w := range c.wait {
		w.kill()
	}
	c.mu.Unlock()
}

func (c *conn) retire() {
	c.retired.Store(true)
	c.mu.Lock()
	idle := len(c.wait) == 0
	c.mu.Unlock()
	if idle {
		c.close()
	}
}

func (c *conn) readLoop() {
	defer c.close()
	for {
		b, err := c.sess.Recv(c.ctx)
		if err != nil {
			return
		}
		env, err := proto.Unmarshal(b)
		if err != nil {
			return
		}
		switch env.T {
		case proto.TToolCall:
			c.startTool(env)
			continue
		case proto.TPong:
			continue
		}
		c.mu.Lock()
		w := c.wait[env.ID]
		c.mu.Unlock()
		if w == nil {
			continue
		}
		select {
		case w.ch <- env:
		case <-w.dead:
		default:
			// This request's reader is not keeping up: fail it alone
			// rather than stall every other request on the session.
			w.kill()
			c.unregister(env.ID)
			_ = c.send(c.ctx, proto.Envelope{T: proto.TChatCancel, ID: env.ID})
		}
	}
}

func (c *conn) register() (uint64, *waiter) {
	id := c.nextID.Add(1)
	w := &waiter{ch: make(chan proto.Envelope, 256), dead: make(chan struct{})}
	c.mu.Lock()
	c.wait[id] = w
	c.mu.Unlock()
	return id, w
}

func (c *conn) unregister(id uint64) {
	c.mu.Lock()
	delete(c.wait, id)
	idle := len(c.wait) == 0
	c.mu.Unlock()
	if idle && c.retired.Load() {
		c.close()
	}
}

func (c *conn) send(ctx context.Context, e proto.Envelope) error {
	b, err := proto.Marshal(e)
	if err != nil {
		return err
	}
	return c.sess.Send(ctx, b)
}

func (c *conn) toolAllowed(name string) bool {
	host := c.d.opts.Tools
	if host == nil {
		return false
	}
	for _, t := range host.Tools() {
		if t.Name != name {
			continue
		}
		if c.d.opts.AllowedTools == nil {
			return t.ReadOnly
		}
		return slices.Contains(c.d.opts.AllowedTools, name)
	}
	return false
}

func toolErr(id uint64, err error) proto.Envelope {
	return proto.Envelope{T: proto.TToolResult, ID: id, Err: proto.ToWire(err)}
}

func (c *conn) startTool(env proto.Envelope) {
	select {
	case c.d.tools <- struct{}{}:
	default:
		_ = c.send(c.ctx, toolErr(env.ID, &omnillm.Error{Code: omnillm.CodeBusy, Msg: omnillm.MsgBusyQueue, RetryAfter: 5 * time.Second}))
		return
	}
	go func() {
		defer func() { <-c.d.tools }()
		c.runTool(env)
	}()
}

func (c *conn) runTool(env proto.Envelope) {
	ctx, cancel := context.WithTimeout(c.ctx, 2*time.Minute)
	defer cancel()
	if !c.toolAllowed(env.Tool) {
		_ = c.send(ctx, toolErr(env.ID, &omnillm.Error{Code: omnillm.CodeToolNotAllowed, Msg: omnillm.MsgToolNotAllowed}))
		return
	}
	var res json.RawMessage
	var err error
	func() {
		// A tool that panics must not take the host down.
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("tool panicked")
			}
		}()
		res, err = c.d.opts.Tools.Call(ctx, env.Tool, env.Args)
	}()
	if err != nil {
		_ = c.send(ctx, toolErr(env.ID, err))
		return
	}
	if !json.Valid(res) {
		res, _ = json.Marshal(string(res))
	}
	_ = c.send(ctx, proto.Envelope{T: proto.TToolResult, ID: env.ID, Result: res})
}

// wireErr turns a chat.error into an error, never a nil pointer.
func wireErr(env proto.Envelope) error {
	if env.Err == nil {
		return &omnillm.Error{Code: omnillm.CodeUpstream, Msg: omnillm.MsgUpstream}
	}
	return env.Err.Error()
}

// next waits for the next message for one request.
func (c *conn) next(ctx context.Context, w *waiter, timeout <-chan time.Time) (proto.Envelope, error) {
	select {
	case env := <-w.ch:
		return env, nil
	case <-w.dead:
		return proto.Envelope{}, offline()
	case <-c.ctx.Done():
		return proto.Envelope{}, offline()
	case <-timeout:
		return proto.Envelope{}, offline()
	case <-ctx.Done():
		return proto.Envelope{}, ctx.Err()
	}
}

// Models lists what this device may use. Every model is Remote from here.
func (d *Device) Models(ctx context.Context) ([]omnillm.ModelInfo, error) {
	c, err := d.connect(ctx)
	if err != nil {
		return nil, err
	}
	id, w := c.register()
	defer c.unregister(id)
	if err := c.send(ctx, proto.Envelope{T: proto.TModelsList, ID: id}); err != nil {
		return nil, offline()
	}
	t := time.NewTimer(d.opts.AckTimeout + 5*time.Second)
	defer t.Stop()
	env, err := c.next(ctx, w, t.C)
	if err != nil {
		return nil, err
	}
	if env.T == proto.TChatError {
		return nil, wireErr(env)
	}
	for i := range env.Models {
		env.Models[i].Locality = omnillm.Remote
	}
	return env.Models, nil
}

// Complete sends a request to the hub and streams the reply.
func (d *Device) Complete(ctx context.Context, req omnillm.Request) (omnillm.Stream, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	c, err := d.connect(ctx)
	if err != nil {
		return nil, err
	}
	id, w := c.register()
	r := req
	if err := c.send(ctx, proto.Envelope{T: proto.TChatStart, ID: id, Req: &r}); err != nil {
		c.unregister(id)
		c.close()
		return nil, offline()
	}
	// Wait for the hub's acknowledgement: a hub that vanished without the
	// relay noticing is reported within AckTimeout, not after a long hang.
	t := time.NewTimer(d.opts.AckTimeout)
	defer t.Stop()
	env, err := c.next(ctx, w, t.C)
	if err != nil {
		c.unregister(id)
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			_ = c.send(context.Background(), proto.Envelope{T: proto.TChatCancel, ID: id})
		} else {
			c.close()
		}
		return nil, err
	}
	s := &stream{ctx: ctx, c: c, id: id, w: w}
	switch env.T {
	case proto.TChatAck:
	case proto.TChatError:
		c.unregister(id)
		return nil, wireErr(env)
	default:
		s.pending = &env
	}
	return s, nil
}

type stream struct {
	ctx     context.Context
	c       *conn
	id      uint64
	w       *waiter
	pending *proto.Envelope

	mu     sync.Mutex
	usage  omnillm.Usage
	finish string
	done   atomic.Bool
	once   sync.Once
}

func (s *stream) end() {
	s.done.Store(true)
	s.c.unregister(s.id)
}

func (s *stream) Recv() (omnillm.Chunk, error) {
	for {
		if s.done.Load() {
			return omnillm.Chunk{}, io.EOF
		}
		var env proto.Envelope
		if s.pending != nil {
			env, s.pending = *s.pending, nil
		} else {
			e, err := s.c.next(s.ctx, s.w, nil)
			if err != nil {
				if s.ctx.Err() != nil {
					s.Close()
				} else {
					s.end()
				}
				return omnillm.Chunk{}, err
			}
			env = e
		}
		switch env.T {
		case proto.TChatDelta:
			if env.Text != "" {
				return omnillm.Chunk{Text: env.Text}, nil
			}
		case proto.TChatEnd:
			s.mu.Lock()
			if env.Usage != nil {
				s.usage = *env.Usage
			}
			s.finish = env.Finish
			s.mu.Unlock()
			s.end()
			return omnillm.Chunk{}, io.EOF
		case proto.TChatError:
			s.end()
			return omnillm.Chunk{}, wireErr(env)
		}
	}
}

func (s *stream) Usage() omnillm.Usage {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.usage
}

func (s *stream) FinishReason() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.finish
}

// Close ends the request; a Recv blocked in another goroutine wakes up.
func (s *stream) Close() error {
	s.once.Do(func() {
		if !s.done.Load() {
			_ = s.c.send(context.Background(), proto.Envelope{T: proto.TChatCancel, ID: s.id})
		}
		s.w.kill()
		s.end()
	})
	return nil
}
