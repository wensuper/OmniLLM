// Package relaylink is the client side of the relay: the hub's single
// multiplexed link, and a device's connection to its mailbox. Both only
// carry end-to-end ciphertext produced by package e2e.
package relaylink

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"math/rand/v2"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/wensuper/OmniLLM/internal/relayproto"
)

// CheckURL accepts wss:// anywhere and ws:// only on loopback (tests and a
// relay on the same machine). The relay is untrusted, but TLS still keeps
// the link's metadata from the network in between.
func CheckURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, errors.New("relaylink: bad relay URL")
	}
	switch u.Scheme {
	case "wss":
	case "ws":
		host := u.Hostname()
		ip := net.ParseIP(host)
		if !(host == "localhost" || (ip != nil && ip.IsLoopback())) {
			return nil, errors.New("relaylink: ws:// is only allowed on loopback; use wss://")
		}
	default:
		return nil, errors.New("relaylink: relay URL must be wss://")
	}
	return u, nil
}

func dialOpts(rt http.RoundTripper, auth string) *websocket.DialOptions {
	h := http.Header{}
	h.Set("Authorization", "Bearer "+auth)
	c := &http.Client{Transport: rt, Timeout: 0}
	if rt == nil {
		c.Transport = http.DefaultTransport
	}
	c.CheckRedirect = func(*http.Request, []*http.Request) error { return errors.New("relaylink: redirects refused") }
	return &websocket.DialOptions{HTTPClient: c, HTTPHeader: h, CompressionMode: websocket.CompressionDisabled}
}

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
		start := time.Now()
		err := l.runOnce(ctx)
		if ctx.Err() != nil {
			return
		}
		if time.Since(start) > time.Minute {
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

func (l *HubLink) runOnce(ctx context.Context) error {
	dctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	c, resp, err := websocket.Dial(dctx, l.url, dialOpts(l.rt, l.token))
	cancel()
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	if err != nil {
		return err
	}
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
			return err
		}
	}
	for {
		typ, b, err := c.Read(ctx)
		if err != nil {
			return err
		}
		if typ != websocket.MessageBinary {
			return errors.New("relaylink: unexpected text frame")
		}
		t, id, payload, err := relayproto.Decode(b)
		if err != nil {
			return err
		}
		switch t {
		case relayproto.Open, relayproto.Data, relayproto.Closed, relayproto.ClaimFail:
		default:
			continue
		}
		select {
		case l.events <- Event{Type: t, Mailbox: id, Payload: payload}:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (l *HubLink) write(ctx context.Context, c *websocket.Conn, b []byte) error {
	wctx, cancel := context.WithTimeout(ctx, 15*time.Second)
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

// ErrUnavailable means the mailbox is unknown, the secret is wrong or the
// hub is offline; the relay deliberately does not say which.
var ErrUnavailable = errors.New("relaylink: hub unavailable")

// DialMailbox connects a device to its mailbox.
func DialMailbox(ctx context.Context, relayURL string, id relayproto.ID, secret []byte, rt http.RoundTripper) (*DeviceConn, error) {
	u, err := CheckURL(relayURL)
	if err != nil {
		return nil, err
	}
	u.Path = "/v1/mailbox/" + id.String()
	dctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	c, resp, err := websocket.Dial(dctx, u.String(), dialOpts(rt, base64.RawURLEncoding.EncodeToString(secret)))
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	if err != nil {
		if resp != nil && (resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusTooManyRequests) {
			return nil, ErrUnavailable
		}
		return nil, fmt.Errorf("relaylink: %w", err)
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

func (d *DeviceConn) WriteMessage(ctx context.Context, b []byte) error {
	wctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	return d.c.Write(wctx, websocket.MessageBinary, b)
}

func (d *DeviceConn) Close() error { return d.c.Close(websocket.StatusNormalClosure, "bye") }
