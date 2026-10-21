package hub_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	omnillm "github.com/wensuper/OmniLLM"
	"github.com/wensuper/OmniLLM/device"
	"github.com/wensuper/OmniLLM/hub"
	"github.com/wensuper/OmniLLM/secret"
	"github.com/wensuper/OmniLLM/store"
)

// Pausing a device stops a reply it is being given, not only the next
// one; the device hears that it was paused, and the upstream call ends.
func TestPauseStopsAReplyInProgress(t *testing.T) {
	cancelled := make(chan struct{}, 1)
	e := setup(t, nil)
	e.fake.setHook(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		if r.URL.Path == "/models" {
			next(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"first\"}}]}\n\n")
		w.(http.Flusher).Flush()
		<-r.Context().Done()
		cancelled <- struct{}{}
	})
	d, _ := e.pairOK()
	ctx := context.Background()
	s, err := d.Complete(ctx, omnillm.Request{Model: "fake/m1", Messages: []omnillm.Message{{Role: "user", Content: "hello"}}, Caller: omnillm.Interactive})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if c, err := s.Recv(); err != nil || c.Text != "first" {
		t.Fatalf("%q %v", c.Text, err)
	}
	devs, _ := e.hub.Devices(ctx)
	if _, err := e.hub.UpdateDevice(ctx, devs[0].ID, nil, nil, ptr(true), nil); err != nil {
		t.Fatal(err)
	}
	got := make(chan error, 1)
	go func() {
		for {
			if _, err := s.Recv(); err != nil {
				got <- err
				return
			}
		}
	}()
	select {
	case err := <-got:
		if !errors.Is(err, omnillm.ErrDevicePaused) || omnillm.AsError(err).Msg != omnillm.MsgDevicePaused {
			t.Fatalf("the open reply ended with %v, not device_paused", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("pausing did not stop the reply in progress")
	}
	select {
	case <-cancelled:
	case <-time.After(5 * time.Second):
		t.Fatal("pausing did not end the upstream call")
	}
}

// The Mac losing its relay link while the owner is asked ends the pairing
// with "remote access is unavailable", not "the device left".
func TestPairingEndedByLinkDownSaysRelay(t *testing.T) {
	// The hub's own connections, so the test can cut its link to the relay.
	var mu sync.Mutex
	var conns []net.Conn
	tr := &http.Transport{DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
		c, err := (&net.Dialer{}).DialContext(ctx, network, addr)
		if err == nil {
			mu.Lock()
			conns = append(conns, c)
			mu.Unlock()
		}
		return c, err
	}}
	e := setup(t, func(o *hub.Options) { o.Transport = tr })
	inQuestion := make(chan struct{})
	p, err := e.hub.Pair(context.Background(), hub.PairOptions{Confirm: func(ctx context.Context, _ hub.PairRequest) (bool, error) {
		close(inQuestion)
		<-ctx.Done()
		return false, ctx.Err()
	}})
	if err != nil {
		t.Fatal(err)
	}
	dctx, dcancel := context.WithCancel(context.Background())
	defer dcancel()
	go func() { _, _ = device.Pair(dctx, device.Options{Secrets: secret.NewMemory()}, p.Code, "phone", nil) }()
	select {
	case <-inQuestion:
	case <-time.After(10 * time.Second):
		t.Fatal("owner was never asked")
	}
	mu.Lock()
	for _, c := range conns { // the Mac's link to the relay drops
		_ = c.Close()
	}
	mu.Unlock()
	wctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = p.Wait(wctx)
	if errors.Is(err, hub.ErrPairingDeviceLeft) || omnillm.AsError(err).Msg != omnillm.MsgRelayUnavailable {
		t.Fatalf("want relay unavailable, got %v", err)
	}
}

// Two apps with one name from before names were unique are told apart
// when the hub lists them; a new app never takes a used name.
func TestDuplicateAppNamesAreToldApart(t *testing.T) {
	ctx := context.Background()
	st := store.NewMemory()
	base := time.Now().Add(-time.Hour).UTC()
	for i, id := range []string{"aaaa", "bbbb", "cccc"} {
		a := hub.App{ID: id, Name: "my-script", TokenHash: []byte{byte(i)}, Budget: 1, Created: base.Add(time.Duration(i) * time.Minute)}
		if err := store.PutJSON(ctx, st, "app/"+id, a); err != nil {
			t.Fatal(err)
		}
	}
	h, err := hub.New(ctx, hub.Options{Secrets: secret.NewMemory(), Store: st})
	if err != nil {
		t.Fatal(err)
	}
	apps, err := h.Apps(ctx)
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]string{}
	for _, a := range apps {
		names[a.ID] = a.Name
	}
	if names["aaaa"] != "my-script" || names["bbbb"] != "my-script 2" || names["cccc"] != "my-script 3" {
		t.Fatalf("names %v", names)
	}
	var kept hub.App
	if err := store.GetJSON(ctx, st, "app/cccc", &kept); err != nil || kept.Name != "my-script 3" {
		t.Fatalf("the new name was not kept: %+v %v", kept, err)
	}
	a, _, err := h.AddApp(ctx, "My-Script", nil, 0)
	if err != nil || a.Name != "My-Script 4" {
		t.Fatalf("new app %q %v", a.Name, err)
	}
}

// An app over its own limit, or using a model it may not, hears about
// the app, not about "this device".
func TestAppsGetTheirOwnSentences(t *testing.T) {
	e := setup(t, nil)
	ctx := context.Background()
	a, tok, err := e.hub.AddApp(ctx, "script", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.hub.UpdateApp(ctx, a.ID, nil, nil, nil, ptr([]string{"fake/m2"})); err != nil {
		t.Fatal(err)
	}
	req := omnillm.Request{Model: "fake/m1", Messages: []omnillm.Message{{Role: "user", Content: "hello"}}}
	if _, err := e.hub.CompleteAsApp(ctx, tok, req); omnillm.AsError(err).Msg != omnillm.MsgModelNotAllowedApp {
		t.Fatalf("model not allowed: %v", err)
	}
	// One call is spent, then the limit is set below it: used up.
	if _, err := e.hub.UpdateApp(ctx, a.ID, nil, nil, nil, ptr([]string{})); err != nil {
		t.Fatal(err)
	}
	s, err := e.hub.CompleteAsApp(ctx, tok, req)
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := s.Recv(); err != nil {
			break
		}
	}
	s.Close()
	if _, err := e.hub.UpdateApp(ctx, a.ID, nil, ptr(int64(1)), nil, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := e.hub.CompleteAsApp(ctx, tok, req); omnillm.AsError(err).Msg != omnillm.MsgBudgetApp {
		t.Fatalf("over budget: %v", err)
	}
	// A device keeps the device sentence.
	d, _ := e.pairOK()
	devs, _ := e.hub.Devices(ctx)
	if _, err := e.hub.UpdateDevice(ctx, devs[0].ID, nil, nil, nil, ptr([]string{"fake/m2"})); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ask(ctx, d, "fake/m1", "hello", omnillm.Interactive); omnillm.AsError(err).Msg != omnillm.MsgModelNotAllowed {
		t.Fatalf("device: %v", err)
	}
}
