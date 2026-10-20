// Package relaylink is the client side of the relay: the hub's single
// multiplexed link, and a device's connection to its mailbox. Both only
// carry end-to-end ciphertext produced by package e2e.
//
// In the browser (GOOS=js) only the device side is built, on the browser's
// own WebSocket, so the web client carries neither net/http nor a
// WebSocket library (docs/design.md §13.3).
package relaylink

import (
	"errors"
	"net/netip"
	"net/url"
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
		ip, err := netip.ParseAddr(host)
		if !(host == "localhost" || (err == nil && ip.IsLoopback())) {
			return nil, errors.New("relaylink: ws:// is only allowed on loopback; use wss://")
		}
	default:
		return nil, errors.New("relaylink: relay URL must be wss://")
	}
	return u, nil
}

// ErrUnavailable means the mailbox is unknown, the secret is wrong or the
// hub is offline; the relay deliberately does not say which.
var ErrUnavailable = errors.New("relaylink: hub unavailable")
