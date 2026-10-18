package hub_test

// Regression tests for the round-3 review findings.

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	omnillm "github.com/wensuper/OmniLLM"
	"github.com/wensuper/OmniLLM/device"
	"github.com/wensuper/OmniLLM/hub"
	"github.com/wensuper/OmniLLM/internal/egress"
	"github.com/wensuper/OmniLLM/internal/proto"
	"github.com/wensuper/OmniLLM/internal/relaylink"
	"github.com/wensuper/OmniLLM/internal/relayproto"
	"github.com/wensuper/OmniLLM/secret"
)

func TestPairingCodeSecretIsDeadAfterPairing(t *testing.T) {
	e := setup(t, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	codes := make(chan string, 1)
	p, err := e.hub.Pair(ctx, hub.PairOptions{Confirm: func(ctx context.Context, r hub.PairRequest) (bool, error) {
		return r.Code == <-codes, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	sec := secret.NewMemory()
	if _, err := device.Pair(ctx, device.Options{Secrets: sec}, p.Code, "phone", func(s string) { codes <- s }); err != nil {
		t.Fatal(err)
	}
	pc, _ := proto.ParsePairingCode(p.Code, time.Now())
	mid, _ := relayproto.ParseID(pc.Mailbox)
	time.Sleep(100 * time.Millisecond)
	if _, err := relaylink.DialMailbox(ctx, e.relayWS, mid, pc.Secret, nil); !errors.Is(err, relaylink.ErrUnavailable) {
		t.Fatalf("the secret from the pairing code still opens the mailbox: %v", err)
	}
	d, _ := device.Open(ctx, device.Options{Secrets: sec})
	defer d.Close()
	if _, _, err := ask(ctx, d, "fake/m1", "hello", omnillm.Interactive); err != nil {
		t.Fatalf("the paired device must use its new secret: %v", err)
	}
}

func TestSimultaneousPairingsNeverBothSucceed(t *testing.T) {
	e := setup(t, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var asking atomic.Int32
	var overlap atomic.Bool
	p, err := e.hub.Pair(ctx, hub.PairOptions{Confirm: func(ctx context.Context, r hub.PairRequest) (bool, error) {
		if asking.Add(1) > 1 {
			overlap.Store(true)
		}
		time.Sleep(200 * time.Millisecond)
		asking.Add(-1)
		return true, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	var ok atomic.Int32
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := device.Pair(ctx, device.Options{Secrets: secret.NewMemory()}, p.Code, "x", nil); err == nil {
				ok.Add(1)
			}
		}()
	}
	wg.Wait()
	if overlap.Load() {
		t.Fatal("the owner was asked about two devices at once")
	}
	devs, _ := e.hub.Devices(ctx)
	if ok.Load() > 1 || len(devs) > 1 {
		t.Fatalf("%d pairings succeeded, %d devices", ok.Load(), len(devs))
	}
}

func TestLiveSessionSeesNewPermissions(t *testing.T) {
	e := setup(t, nil)
	d, _ := e.pairOK()
	ctx := context.Background()
	if _, _, err := ask(ctx, d, "fake/m2", "hello", omnillm.Interactive); err != nil {
		t.Fatal(err)
	}
	devs, _ := e.hub.Devices(ctx)
	if err := e.hub.SetDevice(ctx, devs[0].ID, []string{"fake/m1"}, 5_000_000, time.Time{}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ask(ctx, d, "fake/m2", "hello", omnillm.Interactive); !errors.Is(err, omnillm.ErrModelNotAllowed) {
		t.Fatalf("the live session kept its old model list: %v", err)
	}
	_ = e.hub.SetDevice(ctx, devs[0].ID, nil, 5_000_000, time.Now().Add(-time.Minute))
	if _, _, err := ask(ctx, d, "fake/m1", "hello", omnillm.Interactive); err == nil {
		t.Fatal("an expired device kept working on its live session")
	}
}

func TestBackgroundOnlyDeviceCannotClaimAPerson(t *testing.T) {
	e := setup(t, nil)
	ctx := context.Background()
	_ = e.hub.SetModel(ctx, "fake/m2", hub.ModelUpdate{HumanOnly: ptr(true)})
	sec, _, err := e.pair(func(a, b string) bool { return a == b }, hub.PairOptions{BackgroundOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	d, _ := device.Open(ctx, device.Options{Secrets: sec})
	defer d.Close()
	if _, _, err := ask(ctx, d, "fake/m2", "hello", omnillm.Interactive); !errors.Is(err, omnillm.ErrModelNotAllowed) {
		t.Fatalf("a background-only device reached a human-only model by claiming Interactive: %v", err)
	}
	ms, _ := d.Models(ctx)
	for _, m := range ms {
		if m.ID == "fake/m2" {
			t.Fatal("human-only model listed for a background-only device")
		}
	}
}

func TestSetModelKeepsHumanOnly(t *testing.T) {
	e := setup(t, nil)
	ctx := context.Background()
	_ = e.hub.SetModel(ctx, "fake/m2", hub.ModelUpdate{HumanOnly: ptr(true)})
	in, out := 3.0, 15.0
	if err := e.hub.SetModel(ctx, "fake/m2", hub.ModelUpdate{PriceIn: &in, PriceOut: &out}); err != nil {
		t.Fatal(err)
	}
	// Re-adding the provider (a new key) must not reset it either.
	if _, err := e.hub.AddProvider(ctx, providerCfg(e.fake.URL), sentinelKey); err != nil {
		t.Fatal(err)
	}
	ms, _ := e.hub.Models(ctx)
	for _, m := range ms {
		if m.ID == "fake/m2" && !m.HumanOnly {
			t.Fatal("HumanOnly was silently cleared")
		}
	}
}

func TestCancelsDoNotOpenTheBreaker(t *testing.T) {
	// A high request rate isolates the breaker from the rate bucket.
	e := setup(t, func(o *hub.Options) {
		o.Limits = func(string) egress.Limits { return egress.Limits{RequestsPerMinute: 6000} }
	})
	e.fake.setHook(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		if r.URL.Path == "/models" {
			next(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	})
	for i := 0; i < 8; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		s, err := e.hub.Complete(ctx, omnillm.Request{Model: "fake/m1", Caller: omnillm.Interactive,
			Messages: []omnillm.Message{{Role: "user", Content: "hello"}}})
		if err == nil {
			_, _ = s.Recv()
			s.Close()
		}
		cancel()
	}
	e.fake.setHook(nil)
	if _, _, err := ask(context.Background(), e.hub, "fake/m1", "hello", omnillm.Interactive); err != nil {
		t.Fatalf("cancelled calls tripped the breaker: %v (%s)", err, omnillm.AsError(err).Msg)
	}
}

func TestRelayRestartRecovers(t *testing.T) {
	e := setup(t, nil)
	d, _ := e.pairOK()
	ctx := context.Background()
	if _, _, err := ask(ctx, d, "fake/m1", "hello", omnillm.Interactive); err != nil {
		t.Fatal(err)
	}
	// Drop every connection the relay has (as a relay restart would).
	e.relay.CloseClientConnections()
	deadline := time.Now().Add(20 * time.Second)
	var err error
	for time.Now().Before(deadline) {
		if _, _, err = ask(ctx, d, "fake/m1", "hello", omnillm.Interactive); err == nil {
			break
		}
		time.Sleep(300 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("no recovery after the relay dropped everything: %v", err)
	}
	devs, _ := e.hub.Devices(ctx)
	tctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	start := time.Now()
	_, _ = e.hub.CallTool(tctx, devs[0].ID, "read_inbox", nil)
	if time.Since(start) > 3500*time.Millisecond {
		t.Fatal("CallTool hung past its deadline")
	}
}

func TestCloseDuringStreamsDoesNotPanic(t *testing.T) {
	e := setup(t, nil)
	_, sec := e.pairOK()
	ctx := context.Background()
	for round := 0; round < 5; round++ {
		d, _ := device.Open(ctx, device.Options{Secrets: sec})
		var wg sync.WaitGroup
		for i := 0; i < 4; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _, _ = ask(ctx, d, "fake/m1", "hello", omnillm.Interactive)
			}()
		}
		time.Sleep(time.Duration(round*15) * time.Millisecond)
		d.Close()
		wg.Wait()
	}
}

func TestRevokedDeviceStaysRevokedAfterRestart(t *testing.T) {
	e := setup(t, nil)
	d, _ := e.pairOK()
	ctx := context.Background()
	_, _, _ = ask(ctx, d, "fake/m1", "hello", omnillm.Interactive)
	devs, _ := e.hub.Devices(ctx)
	_ = e.hub.Revoke(ctx, devs[0].ID)
	// The device keeps trying.
	for i := 0; i < 3; i++ {
		_, _, _ = ask(ctx, d, "fake/m1", "hello", omnillm.Interactive)
	}
	if devs, _ := e.hub.Devices(ctx); len(devs) != 0 {
		t.Fatal("revoked device came back")
	}
	if keys, _ := e.hubSt.List(ctx, "device"); len(keys) != 0 {
		t.Fatalf("records left behind: %v", keys)
	}
}
