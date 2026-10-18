package hub_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	omnillm "github.com/wensuper/OmniLLM"
	"github.com/wensuper/OmniLLM/device"
	"github.com/wensuper/OmniLLM/hub"
	"github.com/wensuper/OmniLLM/internal/e2e"
	"github.com/wensuper/OmniLLM/internal/egress"
	"github.com/wensuper/OmniLLM/internal/proto"
	"github.com/wensuper/OmniLLM/internal/relaylink"
	"github.com/wensuper/OmniLLM/internal/relayproto"
	"github.com/wensuper/OmniLLM/secret"
	"github.com/wensuper/OmniLLM/store"
)

// rawSession opens a real e2e session with the device's saved keys, so the
// test can send anything a compromised device might.
func rawSession(t *testing.T, e *env, sec *secret.Memory) *e2e.Session {
	t.Helper()
	ctx := context.Background()
	get := func(n string) []byte {
		b, err := sec.Get(ctx, n)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	key, _ := e2e.UnmarshalKey(get("device/key"))
	mid, _ := relayproto.ParseID(string(get("device/mailbox")))
	conn, err := relaylink.DialMailbox(ctx, e.relayWS, mid, get("device/mailbox_secret"), nil)
	if err != nil {
		t.Fatal(err)
	}
	s, err := e2e.Dial(ctx, conn, mid[:], key, get("device/hub_key"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestMisbehavingDeviceCannotHurtTheHub(t *testing.T) {
	e := setup(t, nil)
	_, evilSec := e.pairOK()
	good, _ := e.pairOK()
	ctx := context.Background()
	s := rawSession(t, e, evilSec)
	junk := [][]byte{
		[]byte("not json"),
		[]byte(`{"t":""}`),
		[]byte(`{"t":"chat.start","id":1}`),
		[]byte(`{"t":"chat.start","id":0,"req":{"model":"fake/m1","messages":[{"role":"user","content":"x"}]}}`),
		[]byte(`{"t":"tool.result","id":99,"result":{}}`),
		[]byte(`{"t":"chat.cancel","id":12345}`),
		[]byte(`{"t":"pair.hello","name":"again"}`),
		[]byte(`{"t":"unknown.type.from.the.future"}`),
		[]byte(`{"t":"chat.start","id":2,"req":{"model":"../../etc/passwd","messages":[{"role":"user","content":"x"}]}}`),
		[]byte(`{"t":"chat.start","id":3,"req":{"model":"fake/m1","messages":[{"role":"system","content":"x"}],"caller":9}}`),
	}
	for _, j := range junk {
		_ = s.Send(ctx, j)
	}
	// Flood far past the per-mailbox buffer.
	for i := 0; i < 500; i++ {
		_ = s.Send(ctx, []byte(`{"t":"ping"}`))
	}
	// The other device is unaffected.
	if text, _, err := ask(ctx, good, "fake/m1", "hello", omnillm.Interactive); err != nil || text != sentinelReply {
		t.Fatalf("good device affected: %q %v", text, err)
	}
	if e.fake.calls.Load() > 1 {
		t.Fatalf("junk reached the provider: %d calls", e.fake.calls.Load())
	}
}

func TestDeviceConcurrencyCap(t *testing.T) {
	e := setup(t, nil)
	_, sec := e.pairOK()
	ctx := context.Background()
	gate := make(chan struct{})
	e.fake.gate.Store(&gate) // hold upstream replies
	defer close(gate)
	s := rawSession(t, e, sec)
	for i := 1; i <= 8; i++ {
		b, _ := proto.Marshal(proto.Envelope{T: proto.TChatStart, ID: uint64(i), Req: &omnillm.Request{Model: "fake/m1",
			Messages: []omnillm.Message{{Role: "user", Content: "hello"}}, Caller: omnillm.Interactive}})
		_ = s.Send(ctx, b)
	}
	busy := 0
	rctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	for busy < 4 {
		b, err := s.Recv(rctx)
		if err != nil {
			t.Fatalf("expected 4 busy replies (cap 4 per device), got %d: %v", busy, err)
		}
		env, _ := proto.Unmarshal(b)
		if env.T == proto.TChatError && env.Err != nil && env.Err.Code == omnillm.CodeBusy {
			busy++
		}
	}
}

func TestManyDevicesConcurrently(t *testing.T) {
	var peak, cur atomic.Int32
	e := setup(t, func(o *hub.Options) {
		o.Limits = func(string) egress.Limits {
			return egress.Limits{MaxConcurrent: 2, RequestsPerMinute: 6000, MaxQueue: 64}
		}
	})
	// Count upstream concurrency.
	e.fake.setHook(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		if r.URL.Path == "/models" {
			next(w, r)
			return
		}
		n := cur.Add(1)
		for {
			p := peak.Load()
			if n <= p || peak.CompareAndSwap(p, n) {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
		next(w, r)
		cur.Add(-1)
	})
	var devs []*device.Device
	for i := 0; i < 3; i++ {
		d, _ := e.pairOK()
		devs = append(devs, d)
	}
	var wg sync.WaitGroup
	var ok, busy, other atomic.Int32
	for _, d := range devs {
		for j := 0; j < 8; j++ {
			wg.Add(1)
			go func(d *device.Device) {
				defer wg.Done()
				_, _, err := ask(context.Background(), d, "fake/m1", "hello", omnillm.Interactive)
				switch {
				case err == nil:
					ok.Add(1)
				case errors.Is(err, omnillm.ErrBusy):
					busy.Add(1)
				default:
					other.Add(1)
					t.Log(err)
				}
			}(d)
		}
	}
	wg.Wait()
	if peak.Load() > 2 {
		t.Fatalf("upstream concurrency %d > 2", peak.Load())
	}
	if other.Load() != 0 || ok.Load() == 0 {
		t.Fatalf("ok=%d busy=%d other=%d", ok.Load(), busy.Load(), other.Load())
	}
}

func TestHubRestartKeepsDevices(t *testing.T) {
	e := setup(t, nil)
	d, _ := e.pairOK()
	ctx := context.Background()
	if _, _, err := ask(ctx, d, "fake/m1", "hello", omnillm.Interactive); err != nil {
		t.Fatal(err)
	}
	sctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	_ = e.hub.Stop(sctx)
	cancel()
	h2, err := hub.New(ctx, hub.Options{Secrets: e.hubSec, Store: e.hubSt, RelayURL: e.relayWS, RelayToken: relayToken})
	if err != nil {
		t.Fatal(err)
	}
	if err := h2.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer h2.Stop(context.Background())
	deadline := time.Now().Add(5 * time.Second)
	for !h2.RelayUp() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(100 * time.Millisecond)
	if text, _, err := ask(ctx, d, "fake/m1", "hello", omnillm.Interactive); err != nil || text != sentinelReply {
		t.Fatalf("after restart: %q %v", text, err)
	}
}

func TestPairingExpires(t *testing.T) {
	e := setup(t, nil)
	ctx := context.Background()
	asked := atomic.Bool{}
	p, err := e.hub.Pair(ctx, hub.PairOptions{TTL: time.Second, Confirm: func(context.Context, hub.PairRequest) (bool, error) {
		asked.Store(true)
		return true, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Wait(ctx); err == nil {
		t.Fatal("pairing should expire")
	}
	if _, err := device.Pair(ctx, device.Options{Secrets: secret.NewMemory()}, p.Code, "late", nil); err == nil || asked.Load() {
		t.Fatalf("expired code worked: %v asked=%v", err, asked.Load())
	}
}

func TestCancelMidStreamReleasesUpstream(t *testing.T) {
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
	ctx, cancel := context.WithCancel(context.Background())
	s, err := d.Complete(ctx, omnillm.Request{Model: "fake/m1", Messages: []omnillm.Message{{Role: "user", Content: "hello"}}, Caller: omnillm.Interactive})
	if err != nil {
		t.Fatal(err)
	}
	if c, err := s.Recv(); err != nil || c.Text != "first" {
		t.Fatalf("%q %v", c.Text, err)
	}
	cancel()
	s.Close()
	select {
	case <-cancelled:
	case <-time.After(5 * time.Second):
		t.Fatal("cancelling on the device did not cancel the upstream request")
	}
	// The slot is free again: a new request goes through.
	e.fake.setHook(nil)
	if _, _, err := ask(context.Background(), d, "fake/m1", "hello", omnillm.Interactive); err != nil {
		t.Fatalf("slot not released: %v", err)
	}
}

func TestTamperingRelayGetsNothingThrough(t *testing.T) {
	e := setup(t, nil)
	d, _ := e.pairOK()
	ctx := context.Background()
	before := e.fake.calls.Load()
	// From now on the "relay" flips a byte in everything it receives.
	e.tap.mu.Lock()
	e.tap.buf.Reset()
	e.tap.mu.Unlock()
	tamper.Store(true)
	defer tamper.Store(false)
	_, _, err := ask(ctx, d, "fake/m1", sentinelPrompt, omnillm.Interactive)
	if err == nil {
		t.Fatal("a tampered request was answered")
	}
	if e.fake.calls.Load() != before {
		t.Fatal("a tampered request reached the provider")
	}
}

func TestStoreSurvivesAndNoKeyInIt(t *testing.T) {
	// Exercise the file store path the CLI uses.
	dir := t.TempDir()
	st, err := store.OpenFile(dir + "/state.json")
	if err != nil {
		t.Fatal(err)
	}
	sec, _ := secret.NewFile(dir+"/secrets.enc", []byte(strings.Repeat("k", 32)))
	ctx := context.Background()
	h, err := hub.New(ctx, hub.Options{Secrets: sec, Store: st})
	if err != nil {
		t.Fatal(err)
	}
	fake := newFake(t)
	if _, err := h.AddProvider(ctx, providerCfg(fake.URL), sentinelKey); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ask(ctx, h, "fake/m1", "hello", omnillm.Background); err != nil {
		t.Fatal(err)
	}
	raw := readFile(t, dir+"/state.json") + readFile(t, dir+"/secrets.enc")
	if strings.Contains(raw, sentinelKey) || strings.Contains(raw, sentinelReply) {
		t.Fatal("key or content on disk in clear")
	}
	_ = httptest.NewRecorder
}
