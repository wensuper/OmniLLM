//go:build !js

package relaylink

import (
	"context"
	"encoding/base64"
	"errors"
	"math/rand/v2"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/wensuper/OmniLLM/internal/relayproto"
)

// Event is something the relay told the hub.
type Event struct {
	Type    byte // relayproto.Open, Data, Closed, ClaimFail
	Mailbox relayproto.ID
	Payload []byte
}

// HubLink keeps one connection to the relay, reconnecting with backoff and
// re-claiming every mailbox after each reconnect.
type HubLink struct {
	url   string
	token string
	rt    http.RoundTripper

	mu     sync.Mutex
	conn   *websocket.Conn
	claims map[relayproto.ID][32]byte
	up     bool
	upCh   chan struct{}

	events chan Event
	logf   func(string, ...any)

	// first: until a connection succeeds, the hub says it is a new
	// install (relay.FirstHeader); then firstDone runs once.
	first     bool
	firstDone func()
}

// FirstConnection marks this hub as a new install: its next successful
// connection says so (only for the relay's daily count), then done runs.
func (l *HubLink) FirstConnection(done func()) {
	l.mu.Lock()
	l.first, l.firstDone = true, done
	l.mu.Unlock()
}

// NewHubLink prepares a link; Run connects.
func NewHubLink(relayURL, token string, rt http.RoundTripper, logf func(string, ...any)) (*HubLink, error) {
	u, err := CheckURL(relayURL)
	if err != nil {
		return nil, err
	}
	if len(token) < 32 {
		return nil, errors.New("relaylink: hub token too short")
	}
	if logf == nil {
		logf = func(string, ...any) {}
	}
	u.Path = "/v1/hub"
	return &HubLink{url: u.String(), token: token, rt: rt, claims: map[relayproto.ID][32]byte{},
		events: make(chan Event, 256), upCh: make(chan struct{}), logf: logf}, nil
}

// Events delivers relay events. It is closed when Run returns.
func (l *HubLink) Events() <-chan Event { return l.events }

// Up reports whether the link is connected now.
func (l *HubLink) Up() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.up
}

// WaitUp blocks until connected or ctx ends.
func (l *HubLink) WaitUp(ctx context.Context) error {
	for {
		l.mu.Lock()
		up, ch := l.up, l.upCh
		l.mu.Unlock()
		if up {
			return nil
		}
		select {
		case <-ch:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Run connects and reconnects until ctx ends.
func (l *HubLink) Run(ctx context.Context) {
	defer close(l.events)
	backoff := time.Second
	for ctx.Err() == nil {
		upFor, err := l.runOnce(ctx)
		if ctx.Err() != nil {
			return
		}
		// A link that worked for a while starts over from the shortest
		// wait; one that drops right after connecting keeps backing off.
		if upFor > 10*time.Second {
			backoff = time.Second
		}
		l.logf("relay link down: %v; retrying in %s", err, backoff)
		// #nosec G404 -- reconnect jitter only; nothing secret depends on it.
		jitter := time.Duration(rand.Int64N(int64(backoff/2) + 1))
		select {
		case <-time.After(backoff + jitter):
		case <-ctx.Done():
			return
		}
		// Capped low: after a relay outage the hub must be back within
		// seconds, not minutes; jitter still spreads reconnecting hubs.
		if backoff < maxBackoff {
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}
}

func (l *HubLink) runOnce(ctx context.Context) (upFor time.Duration, err error) {
	dctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	opts := dialOpts(l.rt, l.token)
	l.mu.Lock()
	first, done := l.first, l.firstDone
	l.mu.Unlock()
	if first {
		opts.HTTPHeader.Set("X-OmniLLM-First", "1")
	}
	c, resp, err := websocket.Dial(dctx, l.url, opts)
	cancel()
	if err == nil && first {
		l.mu.Lock()
		l.first, l.firstDone = false, nil
		l.mu.Unlock()
		if done != nil {
			done()
		}
	}
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	if err != nil {
		return 0, err
	}
	since := time.Now()
	c.SetReadLimit(relayproto.MaxMessage)
	// Go up and take the snapshot in one step: a Claim made after this
	// point is sent directly, one made before is in the snapshot. Sending a
	// claim twice is harmless.
	l.mu.Lock()
	l.conn = c
	l.up = true
	close(l.upCh)
	claims := make(map[relayproto.ID][32]byte, len(l.claims))
	for k, v := range l.claims {
		claims[k] = v
	}
	l.mu.Unlock()
	defer func() {
		upFor = time.Since(since)
		l.mu.Lock()
		l.up = false
		l.upCh = make(chan struct{})
		if l.conn == c {
			l.conn = nil
		}
		l.mu.Unlock()
		c.CloseNow()
		// Every device session through this link is gone.
		select {
		case l.events <- Event{Type: relayproto.LinkDown}:
		case <-ctx.Done():
		}
	}()
	for id, h := range claims {
		if err := l.write(ctx, c, relayproto.Encode(relayproto.Claim, id, h[:])); err != nil {
			return 0, err
		}
	}
	for {
		typ, b, err := c.Read(ctx)
		if err != nil {
			return 0, err
		}
		if typ != websocket.MessageBinary {
			return 0, errors.New("relaylink: unexpected text frame")
		}
		t, id, payload, err := relayproto.Decode(b)
		if err != nil {
			return 0, err
		}
		switch t {
		case relayproto.Open, relayproto.Data, relayproto.Closed, relayproto.ClaimFail:
		default:
			continue
		}
		select {
		case l.events <- Event{Type: t, Mailbox: id, Payload: payload}:
		case <-ctx.Done():
			return 0, ctx.Err()
		}
	}
}

// write sends one frame. The WebSocket library closes the whole connection
// when a write's context ends mid-write, so a write never takes the
// caller's cancellation (one device giving up must not drop the link for
// every device); it has its own time limit instead.
func (l *HubLink) write(_ context.Context, c *websocket.Conn, b []byte) error {
	return writeFrame(c, b)
}

func writeFrame(c *websocket.Conn, b []byte) error {
	wctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return c.Write(wctx, websocket.MessageBinary, b)
}

// maxBackoff caps the wait between reconnect attempts.
const maxBackoff = 10 * time.Second

// ErrDown is returned when the link is not connected.
var ErrDown = errors.New("relaylink: not connected to the relay")

func (l *HubLink) send(ctx context.Context, b []byte) error {
	l.mu.Lock()
	c := l.conn
	up := l.up
	l.mu.Unlock()
	if c == nil || !up {
		return ErrDown
	}
	return l.write(ctx, c, b)
}

// Claim registers a mailbox (kept across reconnects).
func (l *HubLink) Claim(ctx context.Context, id relayproto.ID, secretHash [32]byte) error {
	l.mu.Lock()
	l.claims[id] = secretHash
	l.mu.Unlock()
	err := l.send(ctx, relayproto.Encode(relayproto.Claim, id, secretHash[:]))
	if errors.Is(err, ErrDown) {
		return nil // claimed on the next connect
	}
	return err
}

// Release forgets a mailbox and closes any device on it.
func (l *HubLink) Release(ctx context.Context, id relayproto.ID) error {
	l.mu.Lock()
	delete(l.claims, id)
	l.mu.Unlock()
	err := l.send(ctx, relayproto.Encode(relayproto.Release, id, nil))
	if errors.Is(err, ErrDown) {
		return nil
	}
	return err
}

// Kick closes the device connection on a mailbox.
func (l *HubLink) Kick(ctx context.Context, id relayproto.ID) error {
	return l.send(ctx, relayproto.Encode(relayproto.Kick, id, nil))
}

// Send forwards one end-to-end message to the device on id.
func (l *HubLink) Send(ctx context.Context, id relayproto.ID, msg []byte) error {
	if len(msg) > 65535 {
		return errors.New("relaylink: message too large")
	}
	return l.send(ctx, relayproto.Encode(relayproto.Data, id, msg))
}

// DeviceConn is a device's connection to its mailbox. It implements
// e2e.MsgConn.
type DeviceConn struct {
	c *websocket.Conn
}

// DialMailbox connects a device to its mailbox.
func DialMailbox(ctx context.Context, relayURL string, id relayproto.ID, secret []byte, rt http.RoundTripper) (*DeviceConn, error) {
	u, err := CheckURL(relayURL)
	if err != nil {
		return nil, err
	}
	u.Path = "/v1/mailbox/" + id.String()
	dctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	c, err := dialDevice(dctx, u.String(), base64.RawURLEncoding.EncodeToString(secret), rt)
	if err != nil {
		return nil, err
	}
	c.SetReadLimit(65535 + 64)
	return &DeviceConn{c: c}, nil
}

func (d *DeviceConn) ReadMessage(ctx context.Context) ([]byte, error) {
	typ, b, err := d.c.Read(ctx)
	if err != nil {
		var ce websocket.CloseError
		if errors.As(err, &ce) && (ce.Code == relayproto.CloseUnavailable || ce.Code == relayproto.CloseKicked) {
			return nil, ErrUnavailable
		}
		return nil, err
	}
	if typ != websocket.MessageBinary {
		return nil, errors.New("relaylink: unexpected text frame")
	}
	return b, nil
}

// WriteMessage sends one frame; like the hub's writes it ignores the
// caller's cancellation (a stopped reply must not close the connection the
// device's other requests use).
func (d *DeviceConn) WriteMessage(_ context.Context, b []byte) error {
	return writeFrame(d.c, b)
}

func (d *DeviceConn) Close() error { return d.c.Close(websocket.StatusNormalClosure, "bye") }
