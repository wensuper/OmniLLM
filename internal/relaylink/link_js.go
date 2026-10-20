//go:build js

package relaylink

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"syscall/js"
	"time"

	"github.com/wensuper/OmniLLM/internal/relayproto"
)

// Must match relay.WebProtocol and relay.WebAuthPrefix (the relay package
// is not built for the browser).
const (
	webProtocol   = "omnillm.v1"
	webAuthPrefix = "omnillm.auth."
)

// maxRead is the largest frame a device accepts, as on native builds.
const maxRead = 65535 + 64

// DeviceConn is a device's connection to its mailbox, on the browser's
// WebSocket. It implements e2e.MsgConn.
type DeviceConn struct {
	ws    js.Value
	funcs []js.Func

	mu     sync.Mutex
	queue  [][]byte      // binary frames not read yet
	ready  bool          // the relay's "ready" frame has arrived
	err    error         // set once the connection is closed
	wake   chan struct{} // signalled on every change above
	opened chan struct{} // closed on the open event
}

// DialMailbox connects a device to its mailbox. A browser cannot set
// headers or read the HTTP status, so the secret travels as a subprotocol
// and the relay answers with a "ready" frame or the CloseUnavailable code.
// The browser chooses the network path; rt is ignored.
func DialMailbox(ctx context.Context, relayURL string, id relayproto.ID, secret []byte, _ any) (*DeviceConn, error) {
	u, err := CheckURL(relayURL)
	if err != nil {
		return nil, err
	}
	u.Path = "/v1/mailbox/" + id.String()
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	d, err := open(u.String(), webAuthPrefix+base64.RawURLEncoding.EncodeToString(secret))
	if err != nil {
		return nil, err
	}
	select {
	case <-d.opened:
	case <-d.wake: // closed before opening
	case <-ctx.Done():
	}
	if d.ws.Get("protocol").String() != webProtocol {
		if err := d.failed(ctx); err != nil {
			return nil, err
		}
		d.abort()
		return nil, errors.New("relaylink: this relay does not accept browsers")
	}
	for {
		d.mu.Lock()
		ready, cerr := d.ready, d.err
		d.mu.Unlock()
		if ready {
			return d, nil
		}
		if cerr != nil {
			return nil, cerr
		}
		select {
		case <-d.wake:
		case <-ctx.Done():
			d.abort()
			return nil, fmt.Errorf("relaylink: %w", ctx.Err())
		}
	}
}

// failed returns why a connection that never opened failed, or nil if it
// is still open.
func (d *DeviceConn) failed(ctx context.Context) error {
	d.mu.Lock()
	err := d.err
	d.mu.Unlock()
	if err != nil {
		return err
	}
	if ctx.Err() != nil {
		d.abort()
		return fmt.Errorf("relaylink: %w", ctx.Err())
	}
	return nil
}

func open(u, auth string) (d *DeviceConn, err error) {
	defer func() {
		// The constructor throws on a malformed URL or subprotocol.
		if r := recover(); r != nil {
			d, err = nil, fmt.Errorf("relaylink: %v", r)
		}
	}()
	protos := js.Global().Get("Array").New(webProtocol, auth)
	ws := js.Global().Get("WebSocket").New(u, protos)
	ws.Set("binaryType", "arraybuffer")
	d = &DeviceConn{ws: ws, wake: make(chan struct{}, 1), opened: make(chan struct{})}
	d.on("open", func(js.Value) { close(d.opened) })
	d.on("message", d.onMessage)
	d.on("close", func(ev js.Value) {
		code := ev.Get("code").Int()
		if code == relayproto.CloseUnavailable || code == relayproto.CloseKicked {
			d.closeWith(ErrUnavailable)
		} else {
			d.closeWith(fmt.Errorf("relaylink: connection closed (%d)", code))
		}
		// No more events come after close; let go of the callbacks once
		// this one has returned.
		go d.release()
	})
	return d, nil
}

func (d *DeviceConn) on(event string, f func(js.Value)) {
	fn := js.FuncOf(func(_ js.Value, args []js.Value) any {
		f(args[0])
		return nil
	})
	d.funcs = append(d.funcs, fn)
	d.ws.Set("on"+event, fn)
}

func (d *DeviceConn) release() {
	for _, ev := range []string{"open", "message", "close"} {
		d.ws.Set("on"+ev, js.Null())
	}
	for _, fn := range d.funcs {
		fn.Release()
	}
}

// onMessage runs on the browser's event loop and must not block.
func (d *DeviceConn) onMessage(ev js.Value) {
	data := ev.Get("data")
	d.mu.Lock()
	ready := d.ready
	d.mu.Unlock()
	if data.Type() == js.TypeString {
		if !ready && data.String() == "ready" {
			d.mu.Lock()
			d.ready = true
			d.mu.Unlock()
			d.signal()
			return
		}
		d.fail(errors.New("relaylink: unexpected text frame"))
		return
	}
	if !ready {
		d.fail(errors.New("relaylink: unexpected first frame"))
		return
	}
	n := data.Get("byteLength").Int()
	if n > maxRead {
		d.fail(errors.New("relaylink: frame too large"))
		return
	}
	b := make([]byte, n)
	js.CopyBytesToGo(b, js.Global().Get("Uint8Array").New(data))
	d.mu.Lock()
	if d.err == nil {
		d.queue = append(d.queue, b)
	}
	d.mu.Unlock()
	d.signal()
}

func (d *DeviceConn) signal() {
	select {
	case d.wake <- struct{}{}:
	default:
	}
}

// closeWith records why the connection ended (the first reason wins).
func (d *DeviceConn) closeWith(err error) {
	d.mu.Lock()
	if d.err == nil {
		d.err = err
	}
	d.mu.Unlock()
	d.signal()
}

// fail ends the connection for a protocol error.
func (d *DeviceConn) fail(err error) {
	d.closeWith(err)
	// A browser may only send 1000 or 3000–4999; 4002 marks this side's
	// protocol error.
	d.ws.Call("close", 4002, "protocol error")
}

// abort ends a connection the caller gives up on.
func (d *DeviceConn) abort() {
	d.closeWith(errors.New("relaylink: connection closed"))
	d.ws.Call("close")
}

// ReadMessage returns the next binary frame. Like the WebSocket library
// on native builds, a cancelled ctx closes the connection.
func (d *DeviceConn) ReadMessage(ctx context.Context) ([]byte, error) {
	for {
		d.mu.Lock()
		if len(d.queue) > 0 {
			b := d.queue[0]
			d.queue[0] = nil
			d.queue = d.queue[1:]
			d.mu.Unlock()
			return b, nil
		}
		err := d.err
		d.mu.Unlock()
		if err != nil {
			return nil, err
		}
		select {
		case <-d.wake:
		case <-ctx.Done():
			d.abort()
			return nil, ctx.Err()
		}
	}
}

// WriteMessage sends one binary frame. The browser queues it; nothing here
// blocks, so the caller's cancellation does not matter.
func (d *DeviceConn) WriteMessage(_ context.Context, b []byte) (err error) {
	d.mu.Lock()
	cerr := d.err
	d.mu.Unlock()
	if cerr != nil {
		return cerr
	}
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("relaylink: %v", r)
		}
	}()
	a := js.Global().Get("Uint8Array").New(len(b))
	js.CopyBytesToJS(a, b)
	d.ws.Call("send", a)
	return nil
}

func (d *DeviceConn) Close() error {
	d.closeWith(errors.New("relaylink: connection closed"))
	d.ws.Call("close", 1000, "bye")
	return nil
}
