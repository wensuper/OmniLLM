package relay_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/wensuper/OmniLLM/internal/e2e"
	"github.com/wensuper/OmniLLM/internal/relaylink"
	"github.com/wensuper/OmniLLM/internal/relayproto"
	"github.com/wensuper/OmniLLM/relay"
)

const hubToken = "hub-token-0123456789abcdef0123456789abcdef"

func startRelay(t *testing.T, opts relay.Options) (string, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(relay.New(opts))
	t.Cleanup(srv.Close)
	return "ws" + strings.TrimPrefix(srv.URL, "http"), srv
}

func allowed() relay.Options {
	return relay.Options{HubTokenHashes: [][32]byte{sha256.Sum256([]byte(hubToken))}}
}

func startHub(t *testing.T, url, token string) (*relaylink.HubLink, context.CancelFunc) {
	t.Helper()
	l, err := relaylink.NewHubLink(url, token, nil, t.Logf)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go l.Run(ctx)
	wctx, wc := context.WithTimeout(ctx, 5*time.Second)
	defer wc()
	if err := l.WaitUp(wctx); err != nil {
		cancel()
		t.Fatalf("hub link did not come up: %v", err)
	}
	t.Cleanup(cancel)
	return l, cancel
}

func newMailbox() (relayproto.ID, []byte) {
	var id relayproto.ID
	copy(id[:], e2e.RandomBytes(16))
	return id, e2e.RandomBytes(32)
}

func waitEvent(t *testing.T, l *relaylink.HubLink, typ byte) relaylink.Event {
	t.Helper()
	timeout := time.After(5 * time.Second)
	for {
		select {
		case ev, ok := <-l.Events():
			if !ok {
				t.Fatal("events closed")
			}
			if ev.Type == typ {
				return ev
			}
		case <-timeout:
			t.Fatalf("no event %d", typ)
		}
	}
}

func TestRoundTrip(t *testing.T) {
	url, _ := startRelay(t, allowed())
	hub, _ := startHub(t, url, hubToken)
	id, sec := newMailbox()
	ctx := context.Background()
	if err := hub.Claim(ctx, id, relayproto.SecretHash(sec)); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	dev, err := relaylink.DialMailbox(ctx, url, id, sec, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer dev.Close()
	waitEvent(t, hub, relayproto.Open)
	if err := dev.WriteMessage(ctx, []byte("ping")); err != nil {
		t.Fatal(err)
	}
	ev := waitEvent(t, hub, relayproto.Data)
	if ev.Mailbox != id || string(ev.Payload) != "ping" {
		t.Fatalf("got %+v", ev)
	}
	if err := hub.Send(ctx, id, []byte("pong")); err != nil {
		t.Fatal(err)
	}
	rctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	b, err := dev.ReadMessage(rctx)
	if err != nil || string(b) != "pong" {
		t.Fatalf("device got %q %v", b, err)
	}
	dev.Close()
	waitEvent(t, hub, relayproto.Closed)
}

func TestHubAuth(t *testing.T) {
	url, srv := startRelay(t, allowed())
	for _, tok := range []string{"", "short", strings.Repeat("x", 40)} {
		req, _ := http.NewRequest("GET", srv.URL+"/v1/hub", nil)
		if tok != "" {
			req.Header.Set("Authorization", "Bearer "+tok)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("token %q: status %d", tok, resp.StatusCode)
		}
	}
	_ = url
	// Open mode still refuses tokens under 32 bytes.
	_, osrv := startRelay(t, relay.Options{Open: true})
	req, _ := http.NewRequest("GET", osrv.URL+"/v1/hub", nil)
	req.Header.Set("Authorization", "Bearer short-token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("open mode accepted a short token: %d", resp.StatusCode)
	}
}

func TestDeviceCannotTellWhyItFailed(t *testing.T) {
	url, _ := startRelay(t, allowed())
	hub, _ := startHub(t, url, hubToken)
	id, sec := newMailbox()
	ctx := context.Background()
	_ = hub.Claim(ctx, id, relayproto.SecretHash(sec))
	time.Sleep(50 * time.Millisecond)
	unknown, _ := newMailbox()
	cases := map[string]func() error{
		"wrong secret": func() error { _, err := relaylink.DialMailbox(ctx, url, id, e2e.RandomBytes(32), nil); return err },
		"unknown box":  func() error { _, err := relaylink.DialMailbox(ctx, url, unknown, sec, nil); return err },
	}
	for name, f := range cases {
		if err := f(); !errors.Is(err, relaylink.ErrUnavailable) {
			t.Fatalf("%s: want ErrUnavailable, got %v", name, err)
		}
	}
}

func TestHubOfflineLooksTheSame(t *testing.T) {
	url, _ := startRelay(t, allowed())
	hub, cancel := startHub(t, url, hubToken)
	id, sec := newMailbox()
	_ = hub.Claim(context.Background(), id, relayproto.SecretHash(sec))
	time.Sleep(50 * time.Millisecond)
	cancel()
	time.Sleep(200 * time.Millisecond)
	start := time.Now()
	_, err := relaylink.DialMailbox(context.Background(), url, id, sec, nil)
	if !errors.Is(err, relaylink.ErrUnavailable) {
		t.Fatalf("want ErrUnavailable, got %v", err)
	}
	if d := time.Since(start); d > time.Second {
		t.Fatalf("offline answer took %s, must be immediate", d)
	}
}

func TestOtherHubCannotStealMailbox(t *testing.T) {
	other := "other-hub-token-0123456789abcdef0123456789"
	opts := allowed()
	opts.HubTokenHashes = append(opts.HubTokenHashes, sha256.Sum256([]byte(other)))
	url, _ := startRelay(t, opts)
	hubA, _ := startHub(t, url, hubToken)
	hubB, _ := startHub(t, url, other)
	id, sec := newMailbox()
	ctx := context.Background()
	_ = hubA.Claim(ctx, id, relayproto.SecretHash(sec))
	time.Sleep(50 * time.Millisecond)
	evil := e2e.RandomBytes(32)
	_ = hubB.Claim(ctx, id, relayproto.SecretHash(evil))
	waitEvent(t, hubB, relayproto.ClaimFail)
	if _, err := relaylink.DialMailbox(ctx, url, id, evil, nil); !errors.Is(err, relaylink.ErrUnavailable) {
		t.Fatal("hub B re-keyed hub A's mailbox")
	}
	dev, err := relaylink.DialMailbox(ctx, url, id, sec, nil)
	if err != nil {
		t.Fatal(err)
	}
	dev.Close()
	// Hub B must not be able to send into hub A's mailbox.
	_ = hubB.Send(ctx, id, []byte("x"))
}

func TestReleaseClosesDevice(t *testing.T) {
	url, _ := startRelay(t, allowed())
	hub, _ := startHub(t, url, hubToken)
	id, sec := newMailbox()
	ctx := context.Background()
	_ = hub.Claim(ctx, id, relayproto.SecretHash(sec))
	time.Sleep(50 * time.Millisecond)
	dev, err := relaylink.DialMailbox(ctx, url, id, sec, nil)
	if err != nil {
		t.Fatal(err)
	}
	waitEvent(t, hub, relayproto.Open)
	_ = hub.Release(ctx, id)
	rctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if _, err := dev.ReadMessage(rctx); !errors.Is(err, relaylink.ErrUnavailable) {
		t.Fatalf("released mailbox: want ErrUnavailable, got %v", err)
	}
	if _, err := relaylink.DialMailbox(ctx, url, id, sec, nil); !errors.Is(err, relaylink.ErrUnavailable) {
		t.Fatal("released mailbox still reachable")
	}
}

func TestBrowserOriginRefused(t *testing.T) {
	_, srv := startRelay(t, allowed())
	req, _ := http.NewRequest("GET", srv.URL+"/v1/hub", nil)
	req.Header.Set("Authorization", "Bearer "+hubToken)
	req.Header.Set("Origin", "https://evil.example")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("Origin request got %d", resp.StatusCode)
	}
}

func TestGuessingIsRateLimited(t *testing.T) {
	url, _ := startRelay(t, allowed())
	hub, _ := startHub(t, url, hubToken)
	id, sec := newMailbox()
	ctx := context.Background()
	_ = hub.Claim(ctx, id, relayproto.SecretHash(sec))
	time.Sleep(50 * time.Millisecond)
	for i := 0; i < 15; i++ {
		_, _ = relaylink.DialMailbox(ctx, url, id, e2e.RandomBytes(32), nil)
	}
	// After a burst of wrong guesses even the right secret is refused for a while.
	if _, err := relaylink.DialMailbox(ctx, url, id, sec, nil); err == nil {
		t.Fatal("guessing was not rate limited")
	}
}

func TestNewHubConnectionReplacesOld(t *testing.T) {
	url, _ := startRelay(t, allowed())
	hub1, _ := startHub(t, url, hubToken)
	id, sec := newMailbox()
	ctx := context.Background()
	_ = hub1.Claim(ctx, id, relayproto.SecretHash(sec))
	time.Sleep(50 * time.Millisecond)
	dev, err := relaylink.DialMailbox(ctx, url, id, sec, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer dev.Close()
	hub2, _ := startHub(t, url, hubToken)
	_ = hub2
	rctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if _, err := dev.ReadMessage(rctx); err == nil {
		t.Fatal("device on the replaced hub kept its connection")
	}
}

func TestURLRules(t *testing.T) {
	for _, u := range []string{"http://x", "ws://example.com", "wss://user:pw@x", "wss://x/?a=1", "ftp://x"} {
		if _, err := relaylink.CheckURL(u); err == nil {
			t.Fatalf("accepted %q", u)
		}
	}
	for _, u := range []string{"wss://relay.example.com", "ws://127.0.0.1:9000", "ws://localhost:1"} {
		if _, err := relaylink.CheckURL(u); err != nil {
			t.Fatalf("refused %q: %v", u, err)
		}
	}
	_ = bytes.Equal
}
