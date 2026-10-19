//go:build js

package relaylink

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/coder/websocket"
)

// Must match relay.WebProtocol and relay.WebAuthPrefix (the relay package
// is not built for the browser).
const (
	webProtocol   = "omnillm.v1"
	webAuthPrefix = "omnillm.auth."
)

// dialDevice opens a mailbox from a browser. A browser cannot set headers
// or read the HTTP status, so the secret travels as a subprotocol and the
// relay answers with a "ready" frame or the CloseUnavailable code.
func dialDevice(ctx context.Context, u, auth string, _ http.RoundTripper) (*websocket.Conn, error) {
	c, _, err := websocket.Dial(ctx, u, &websocket.DialOptions{Subprotocols: []string{webProtocol, webAuthPrefix + auth}})
	if err != nil {
		return nil, fmt.Errorf("relaylink: %w", err)
	}
	if c.Subprotocol() != webProtocol {
		c.CloseNow()
		return nil, errors.New("relaylink: this relay does not accept browsers")
	}
	typ, b, err := c.Read(ctx)
	if err != nil {
		if websocket.CloseStatus(err) == 4404 {
			return nil, ErrUnavailable
		}
		return nil, fmt.Errorf("relaylink: %w", err)
	}
	if typ != websocket.MessageText || string(b) != "ready" {
		c.CloseNow()
		return nil, errors.New("relaylink: unexpected first frame")
	}
	return c, nil
}

// dialOpts: a hub link never runs in a browser.
func dialOpts(http.RoundTripper, string) *websocket.DialOptions { return &websocket.DialOptions{} }
