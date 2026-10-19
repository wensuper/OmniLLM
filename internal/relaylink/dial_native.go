//go:build !js

package relaylink

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/coder/websocket"
)

// dialDevice opens a mailbox from a program: the secret is a header.
func dialDevice(ctx context.Context, u, auth string, rt http.RoundTripper) (*websocket.Conn, error) {
	c, resp, err := websocket.Dial(ctx, u, dialOpts(rt, auth))
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	if err != nil {
		if resp != nil && (resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusTooManyRequests) {
			return nil, ErrUnavailable
		}
		return nil, fmt.Errorf("relaylink: %w", err)
	}
	return c, nil
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
