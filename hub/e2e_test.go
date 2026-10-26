package hub_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	omnillm "github.com/wensuper/OmniLLM"
	"github.com/wensuper/OmniLLM/device"
	"github.com/wensuper/OmniLLM/hub"
	"github.com/wensuper/OmniLLM/internal/provider"
	"github.com/wensuper/OmniLLM/relay"
	"github.com/wensuper/OmniLLM/secret"
	"github.com/wensuper/OmniLLM/store"
)

const (
	relayToken     = "relay-hub-token-0123456789abcdef0123456789"
	sentinelKey    = "sk-SENTINELKEY-9f8e7d6c5b4a39281706f5e4d3c2b1a0"
	sentinelPrompt = "SENTINEL-PROMPT-7c1d0b5e-the-relay-must-never-see-this"
	sentinelReply  = "SENTINEL-REPLY-41a2f9-the-relay-must-never-see-this"
)

func init() { provider.AllowLoopbackForTests() }

// tapListener records every byte the relay receives or sends.
type tapListener struct {
	net.Listener
	mu  sync.Mutex
	buf bytes.Buffer
}

func (l *tapListener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return &tapConn{Conn: c, l: l}, nil
}

type tapConn struct {
	net.Conn
	l *tapListener
}

// tamper makes the tapped relay flip a byte in everything it receives.
var tamper atomic.Bool

func (c *tapConn) Read(b []byte) (int, error) {
	n, err := c.Conn.Read(b)
	if n > 8 && tamper.Load() {
		b[n-1] ^= 0x55
	}
	c.l.mu.Lock()
	c.l.buf.Write(b[:n])
	c.l.mu.Unlock()
	return n, err
}

func (c *tapConn) Write(b []byte) (int, error) {
	c.l.mu.Lock()
	c.l.buf.Write(b)
	c.l.mu.Unlock()
	return c.Conn.Write(b)
}

func (l *tapListener) seen() []byte {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]byte{}, l.buf.Bytes()...)
}

// fakeProvider is an OpenAI-compatible upstream. Tests change its
// behaviour only through atomics (hook, gate, status), never by replacing
// fields the server goroutines read.
type fakeProvider struct {
	*httptest.Server
	calls    atomic.Int32
	lastAuth atomic.Value
	status   atomic.Int32
	gate     atomic.Pointer[chan struct{}]
	hook     atomic.Pointer[func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc)]
}

func (f *fakeProvider) setHook(h func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc)) {
	if h == nil {
		f.hook.Store(nil)
		return
	}
	f.hook.Store(&h)
}

func newFake(t *testing.T) *fakeProvider {
	f := &fakeProvider{}
	base := func(w http.ResponseWriter, r *http.Request) {
		f.lastAuth.Store(r.Header.Get("Authorization"))
		if r.Header.Get("Authorization") != "Bearer "+sentinelKey {
			w.WriteHeader(401)
			return
		}
		if r.URL.Path == "/models" {
			fmt.Fprint(w, `{"data":[{"id":"m1"},{"id":"m2"}]}`)
			return
		}
		f.calls.Add(1)
		if s := f.status.Load(); s != 0 {
			w.Header().Set("Retry-After", "45")
			w.WriteHeader(int(s))
			fmt.Fprintf(w, `{"error":"your key %s"}`, sentinelKey)
			return
		}
		if g := f.gate.Load(); g != nil {
			select {
			case <-*g:
			case <-r.Context().Done():
				return
			}
		}
		b, _ := io.ReadAll(r.Body)
		if !bytes.Contains(b, []byte(sentinelPrompt)) && !bytes.Contains(b, []byte("hello")) {
			w.WriteHeader(400)
			return
		}
		fl := w.(http.Flusher)
		for _, part := range []string{sentinelReply[:10], sentinelReply[10:]} {
			fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":%q}}]}\n\n", part)
			fl.Flush()
		}
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":100,\"completion_tokens\":20}}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h := f.hook.Load(); h != nil {
			(*h)(w, r, base)
			return
		}
		base(w, r)
	}))
	t.Cleanup(f.Close)
	return f
}

type env struct {
	t       *testing.T
	tap     *tapListener
	relay   *httptest.Server
	relayWS string
	fake    *fakeProvider
	hub     *hub.Hub
	hubSec  *secret.Memory
	hubSt   *store.Memory
}

func setup(t *testing.T, mod func(*hub.Options)) *env {
	t.Helper()
	// Subscription programs keep conversations under these; never the
	// real ones.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("OMNILLM_HOME", filepath.Join(home, "omnillm"))
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	tap := &tapListener{Listener: ln}
	rs := httptest.NewUnstartedServer(relay.New(relay.Options{HubTokenHashes: [][32]byte{sha256.Sum256([]byte(relayToken))}}))
	rs.Listener = tap
	rs.Start()
	t.Cleanup(rs.Close)
	e := &env{t: t, tap: tap, relay: rs, relayWS: "ws" + strings.TrimPrefix(rs.URL, "http"), fake: newFake(t),
		hubSec: secret.NewMemory(), hubSt: store.NewMemory()}
	opts := hub.Options{Secrets: e.hubSec, Store: e.hubSt, RelayURL: e.relayWS, RelayToken: relayToken, Logf: t.Logf}
	if mod != nil {
		mod(&opts)
	}
	ctx := context.Background()
	h, err := hub.New(ctx, opts)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.AddProvider(ctx, provider.Config{Name: "fake", Kind: provider.OpenAI, BaseURL: e.fake.URL, NeedsKey: true}, sentinelKey); err != nil {
		t.Fatal(err)
	}
	if err := h.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = h.Stop(ctx)
	})
	deadline := time.Now().Add(5 * time.Second)
	for !h.RelayUp() {
		if time.Now().After(deadline) {
			t.Fatal("hub never reached the relay")
		}
		time.Sleep(10 * time.Millisecond)
	}
	e.hub = h
	return e
}

// pair runs a full pairing; confirm decides on the hub side.
func (e *env) pair(confirm func(code, devCode string) bool, opts hub.PairOptions) (*secret.Memory, device.Info, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	devCode := make(chan string, 1)
	opts.Confirm = func(ctx context.Context, r hub.PairRequest) (bool, error) {
		select {
		case c := <-devCode:
			return confirm(r.Code, c), nil
		case <-ctx.Done():
			return false, ctx.Err()
		}
	}
	p, err := e.hub.Pair(ctx, opts)
	if err != nil {
		return nil, device.Info{}, err
	}
	sec := secret.NewMemory()
	info, err := device.Pair(ctx, device.Options{Secrets: sec}, p.Code, "Test Phone", func(s string) { devCode <- s })
	return sec, info, err
}

func (e *env) pairOK() (*device.Device, *secret.Memory) {
	e.t.Helper()
	sec, _, err := e.pair(func(a, b string) bool { return a == b }, hub.PairOptions{})
	if err != nil {
		e.t.Fatalf("pairing: %v", err)
	}
	d, err := device.Open(context.Background(), device.Options{Secrets: sec})
	if err != nil {
		e.t.Fatal(err)
	}
	e.t.Cleanup(func() { d.Close() })
	return d, sec
}

func ask(ctx context.Context, c interface {
	Complete(context.Context, omnillm.Request) (omnillm.Stream, error)
}, model, text string, caller omnillm.Caller) (string, omnillm.Usage, error) {
	s, err := c.Complete(ctx, omnillm.Request{Model: model, Messages: []omnillm.Message{{Role: "user", Content: text}}, Caller: caller})
	if err != nil {
		return "", omnillm.Usage{}, err
	}
	defer s.Close()
	var b strings.Builder
	for {
		ch, err := s.Recv()
		if errors.Is(err, io.EOF) {
			return b.String(), s.Usage(), nil
		}
		if err != nil {
			return b.String(), omnillm.Usage{}, err
		}
		b.WriteString(ch.Text)
	}
}

func TestEndToEndAndTheRelaySeesNothing(t *testing.T) {
	e := setup(t, nil)
	d, devSecrets := e.pairOK()
	ctx := context.Background()
	ms, err := d.Models(ctx)
	// Two models, and Auto first to choose between them.
	if err != nil || len(ms) != 3 || ms[0].ID != "auto" {
		t.Fatalf("models %v %v", ms, err)
	}
	for _, m := range ms {
		if m.Locality != omnillm.Remote {
			t.Fatal("from a device every model is remote")
		}
	}
	text, usage, err := ask(ctx, d, "fake/m1", sentinelPrompt, omnillm.Interactive)
	if err != nil || text != sentinelReply {
		t.Fatalf("reply %q err %v", text, err)
	}
	if usage.InputTokens != 100 || usage.OutputTokens != 20 || usage.CostMicroUSD == 0 {
		t.Fatalf("usage %+v", usage)
	}
	seen := e.tap.seen()
	if len(seen) < 1000 {
		t.Fatal("tap saw nothing; the test is broken")
	}
	for _, s := range []string{sentinelPrompt, sentinelReply, sentinelReply[:10], sentinelKey, "SENTINEL", "fake/m1", "Test Phone"} {
		if bytes.Contains(seen, []byte(s)) {
			t.Fatalf("the relay saw %q in clear", s)
		}
	}
	// The device never holds the provider key.
	names, _ := devSecrets.List(ctx, "")
	for _, n := range names {
		v, _ := devSecrets.Get(ctx, n)
		if bytes.Contains(v, []byte(sentinelKey)) {
			t.Fatal("device holds the provider key")
		}
	}
	// The store holds no secret.
	keys, _ := e.hubSt.List(ctx, "")
	for _, k := range keys {
		v, _ := e.hubSt.Get(ctx, k)
		if bytes.Contains(v, []byte(sentinelKey)) || bytes.Contains(v, []byte(sentinelPrompt)) || bytes.Contains(v, []byte(sentinelReply)) {
			t.Fatalf("store key %s holds a secret or content", k)
		}
	}
}

func TestPairingDeniedSavesNothing(t *testing.T) {
	e := setup(t, nil)
	sec, _, err := e.pair(func(a, b string) bool { return false }, hub.PairOptions{})
	if !errors.Is(err, omnillm.ErrPairingDenied) {
		t.Fatalf("want denied, got %v", err)
	}
	if names, _ := sec.List(context.Background(), ""); len(names) != 0 {
		t.Fatalf("denied device saved %v", names)
	}
	if devs, _ := e.hub.Devices(context.Background()); len(devs) != 0 {
		t.Fatal("hub saved a denied device")
	}
}

func TestPairingCodeIsSingleUse(t *testing.T) {
	e := setup(t, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	codes := make(chan string, 2)
	p, err := e.hub.Pair(ctx, hub.PairOptions{Confirm: func(ctx context.Context, r hub.PairRequest) (bool, error) {
		return r.Code == <-codes, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := device.Pair(ctx, device.Options{Secrets: secret.NewMemory()}, p.Code, "one", func(s string) { codes <- s }); err != nil {
		t.Fatal(err)
	}
	tctx, tcancel := context.WithTimeout(ctx, 5*time.Second) // the device retries a missing mailbox for 20 s
	defer tcancel()
	if _, err := device.Pair(tctx, device.Options{Secrets: secret.NewMemory()}, p.Code, "two", func(s string) { codes <- s }); err == nil {
		t.Fatal("a pairing code worked twice")
	}
	if devs, _ := e.hub.Devices(ctx); len(devs) != 1 {
		t.Fatalf("devices %d", len(devs))
	}
}

func TestTamperedPairingCodeFails(t *testing.T) {
	e := setup(t, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	confirmed := atomic.Bool{}
	p, err := e.hub.Pair(ctx, hub.PairOptions{Confirm: func(context.Context, hub.PairRequest) (bool, error) {
		confirmed.Store(true)
		return true, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	// Flip characters inside the code: the hub key, PSK or secret no longer
	// match, so pairing must fail without ever asking the owner.
	for _, i := range []int{20, 60, 100, 140} {
		b := []byte(p.Code)
		if b[i] == 'A' {
			b[i] = 'B'
		} else {
			b[i] = 'A'
		}
		_, _ = device.Pair(ctx, device.Options{Secrets: secret.NewMemory()}, string(b), "evil", nil)
	}
	if confirmed.Load() {
		t.Fatal("a tampered code reached the owner's confirmation")
	}
}

func TestStolenMailboxSecretWithoutDeviceKey(t *testing.T) {
	e := setup(t, nil)
	_, sec := e.pairOK()
	ctx := context.Background()
	// An attacker copies everything except the device private key.
	evil := secret.NewMemory()
	names, _ := sec.List(ctx, "")
	for _, n := range names {
		v, _ := sec.Get(ctx, n)
		_ = evil.Set(ctx, n, v)
	}
	other, _ := secret.NewMemory().Get(ctx, "x")
	_ = other
	fake := make([]byte, 64)
	copy(fake, bytes.Repeat([]byte{7}, 64))
	_ = evil.Set(ctx, "device/key", fake)
	d, err := device.Open(ctx, device.Options{Secrets: evil})
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	// Failing fast is not the point here (the device keeps retrying); 5 s
	// is enough to see that nothing comes back.
	actx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if _, _, err := ask(actx, d, "fake/m1", "hello", omnillm.Interactive); err == nil {
		t.Fatal("a device without its private key got an answer")
	}
	if e.fake.calls.Load() != 0 {
		t.Fatal("the provider was called for an unauthenticated device")
	}
}

func TestRevokeIsImmediate(t *testing.T) {
	e := setup(t, nil)
	d, _ := e.pairOK()
	ctx := context.Background()
	if _, _, err := ask(ctx, d, "fake/m1", "hello", omnillm.Interactive); err != nil {
		t.Fatal(err)
	}
	devs, _ := e.hub.Devices(ctx)
	if err := e.hub.Revoke(ctx, devs[0].ID); err != nil {
		t.Fatal(err)
	}
	before := e.fake.calls.Load()
	if _, _, err := ask(ctx, d, "fake/m1", "hello", omnillm.Interactive); err == nil {
		t.Fatal("revoked device still works")
	}
	if e.fake.calls.Load() != before {
		t.Fatal("revoked device reached the provider")
	}
}

func TestHubOfflineIsFast(t *testing.T) {
	e := setup(t, nil)
	d, _ := e.pairOK()
	ctx := context.Background()
	if _, _, err := ask(ctx, d, "fake/m1", "hello", omnillm.Interactive); err != nil {
		t.Fatal(err)
	}
	sctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	_ = e.hub.Stop(sctx)
	cancel()
	start := time.Now()
	_, _, err := ask(ctx, d, "fake/m1", "hello", omnillm.Interactive)
	if !errors.Is(err, omnillm.ErrHubOffline) {
		t.Fatalf("want hub_offline, got %v", err)
	}
	// The device answered a moment ago, so it rides out a blip for about
	// 10 s before saying so (device.offlineWaits); never much longer.
	if el := time.Since(start); el > 14*time.Second {
		t.Fatalf("offline took %s", el)
	}
}

func TestModelRules(t *testing.T) {
	e := setup(t, nil)
	ctx := context.Background()
	_, _, err := e.pair(func(a, b string) bool { return a == b }, hub.PairOptions{Models: []string{"fake/m1"}})
	if err != nil {
		t.Fatal(err)
	}
	// Re-open the paired device from the pairing above.
	devs, _ := e.hub.Devices(ctx)
	if len(devs[0].Models) != 1 {
		t.Fatal("model list not stored")
	}
	// Host calls: human-only models refuse Background.
	if err := e.hub.SetModel(ctx, "fake/m2", hub.ModelUpdate{HumanOnly: ptr(true)}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ask(ctx, e.hub, "fake/m2", "hello", omnillm.Background); !errors.Is(err, omnillm.ErrModelNotAllowed) {
		t.Fatalf("background on a human-only model: %v", err)
	}
	if _, _, err := ask(ctx, e.hub, "fake/m2", "hello", omnillm.Interactive); err != nil {
		t.Fatalf("interactive on a human-only model: %v", err)
	}
	if _, _, err := ask(ctx, e.hub, "fake/nope", "hello", omnillm.Interactive); !errors.Is(err, omnillm.ErrModelNotAllowed) {
		t.Fatalf("unknown model: %v", err)
	}
}

func TestDeviceModelAllowlist(t *testing.T) {
	e := setup(t, nil)
	ctx := context.Background()
	sec, _, err := e.pair(func(a, b string) bool { return a == b }, hub.PairOptions{Models: []string{"fake/m1"}})
	if err != nil {
		t.Fatal(err)
	}
	d, _ := device.Open(ctx, device.Options{Secrets: sec})
	defer d.Close()
	// On a slow machine the hub's relay link can drop and come back right
	// after pairing; that is "offline for a moment", not an empty list.
	var ms []omnillm.ModelInfo
	for start := time.Now(); time.Since(start) < 10*time.Second; time.Sleep(200 * time.Millisecond) {
		if ms, err = d.Models(ctx); !errors.Is(err, omnillm.ErrHubOffline) {
			break
		}
	}
	if err != nil || len(ms) != 1 || ms[0].ID != "fake/m1" {
		t.Fatalf("device sees %v (%v)", ms, err)
	}
	if _, _, err := ask(ctx, d, "fake/m2", "hello", omnillm.Interactive); !errors.Is(err, omnillm.ErrModelNotAllowed) {
		t.Fatalf("want model_not_allowed, got %v", err)
	}
}

func TestBudgets(t *testing.T) {
	// Worst case for one call: ~input + 4096 output tokens at the fallback
	// price (75 $/M) ≈ $0.31. A $0.31 budget allows exactly one call.
	e := setup(t, func(o *hub.Options) { o.HostBudget = 310_000; o.GlobalBudget = 1_000_000_000 })
	ctx := context.Background()
	if _, _, err := ask(ctx, e.hub, "fake/m1", "hello", omnillm.Interactive); err != nil {
		t.Fatal(err)
	}
	_, _, err := ask(ctx, e.hub, "fake/m1", "hello", omnillm.Interactive)
	var oe *omnillm.Error
	if !errors.As(err, &oe) || oe.Code != omnillm.CodeBudgetExceeded || oe.RetryAfter <= 0 {
		t.Fatalf("want budget_exceeded with a wait, got %v", err)
	}
	if omnillm.HTTPStatus(oe.Code) != 429 {
		t.Fatal("budget maps to 429")
	}
	if e.fake.calls.Load() != 1 {
		t.Fatalf("the refused call reached the provider: %d calls", e.fake.calls.Load())
	}
}

func TestOversizedCallRefusedBeforeSending(t *testing.T) {
	e := setup(t, nil) // $5 host budget
	ctx := context.Background()
	big := strings.Repeat("x", 3_000_000) // ~1M tokens at 15 $/M ≈ $15
	_, _, err := ask(ctx, e.hub, "fake/m1", big, omnillm.Interactive)
	if !errors.Is(err, omnillm.ErrBudgetExceeded) || e.fake.calls.Load() != 0 {
		t.Fatalf("a call costing more than the budget was sent: %v, %d calls", err, e.fake.calls.Load())
	}
}

func TestEarlyCancelIsStillCharged(t *testing.T) {
	e := setup(t, nil)
	ctx := context.Background()
	s, err := e.hub.Complete(ctx, omnillm.Request{Model: "fake/m1", Caller: omnillm.Interactive,
		Messages: []omnillm.Message{{Role: "user", Content: "hello " + strings.Repeat("y", 3000)}}})
	if err != nil {
		t.Fatal(err)
	}
	s.Close() // before any usage report
	u, _ := e.hub.Usage(ctx, "")
	charged := int64(0)
	for _, r := range u {
		if r.Who == "host" {
			charged = r.MicroUSD
		}
	}
	if charged <= 0 {
		t.Fatalf("a cancelled call was free: %+v", u)
	}
}

func TestGlobalBudget(t *testing.T) {
	e := setup(t, func(o *hub.Options) { o.GlobalBudget = 1 })
	d, _ := e.pairOK()
	ctx := context.Background()
	_, _, _ = ask(ctx, d, "fake/m1", "hello", omnillm.Interactive)
	if _, _, err := ask(ctx, e.hub, "fake/m1", "hello", omnillm.Interactive); !errors.Is(err, omnillm.ErrBudgetExceeded) {
		t.Fatalf("global budget ignored: %v", err)
	}
}

func TestUpstream429ReachesDeviceWithoutUpstreamText(t *testing.T) {
	e := setup(t, nil)
	d, _ := e.pairOK()
	e.fake.status.Store(429)
	_, _, err := ask(context.Background(), d, "fake/m1", "hello", omnillm.Interactive)
	var oe *omnillm.Error
	if !errors.As(err, &oe) || oe.Code != omnillm.CodeBusy || oe.RetryAfter != 45*time.Second {
		t.Fatalf("want busy 45s, got %v", err)
	}
	if strings.Contains(fmt.Sprint(oe, oe.Args, oe.Detail), "SENTINEL") {
		t.Fatal("upstream text reached the device")
	}
	if oe.Detail != "" {
		t.Fatal("Detail must not be sent to devices")
	}
	// No second upstream call during the cooldown.
	_, _, _ = ask(context.Background(), d, "fake/m1", "hello", omnillm.Interactive)
	if e.fake.calls.Load() != 1 {
		t.Fatalf("cooldown ignored: %d upstream calls", e.fake.calls.Load())
	}
}

func TestAppTokens(t *testing.T) {
	e := setup(t, nil)
	ctx := context.Background()
	_, tok, err := e.hub.AddApp(ctx, "script", nil, 0)
	if err != nil || !strings.HasPrefix(tok, "omni_app_") {
		t.Fatal(err)
	}
	apps, _ := e.hub.Apps(ctx)
	if bytes.Contains(apps[0].TokenHash, []byte(tok)) {
		t.Fatal("token stored in clear")
	}
	keys, _ := e.hubSt.List(ctx, "")
	for _, k := range keys {
		v, _ := e.hubSt.Get(ctx, k)
		if bytes.Contains(v, []byte(tok)) {
			t.Fatal("token stored in clear")
		}
	}
}

func TestRegionPause(t *testing.T) {
	region := atomic.Value{}
	region.Store("SG")
	e := setup(t, func(o *hub.Options) {
		o.EgressRegion = func(context.Context) (string, error) { return region.Load().(string), nil }
		o.RegionAllowed = func(p, c string) bool { return c != "CN" }
	})
	ctx := context.Background()
	if _, _, err := ask(ctx, e.hub, "fake/m1", "hello", omnillm.Interactive); err != nil {
		t.Fatal(err)
	}
	region.Store("JP")
	if _, _, err := ask(ctx, e.hub, "fake/m1", "hello", omnillm.Interactive); !errors.Is(err, omnillm.ErrRegionNotAllowed) {
		t.Fatalf("region change must pause, got %v", err)
	}
	region.Store("SG")
	if _, _, err := ask(ctx, e.hub, "fake/m1", "hello", omnillm.Interactive); !errors.Is(err, omnillm.ErrRegionNotAllowed) {
		t.Fatal("pause must hold until the owner confirms")
	}
	region.Store("JP")
	if err := e.hub.ConfirmRegion(ctx); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ask(ctx, e.hub, "fake/m1", "hello", omnillm.Interactive); err != nil {
		t.Fatalf("after confirm: %v", err)
	}
	region.Store("CN")
	_ = e.hub.ConfirmRegion(ctx)
	_, _, _ = ask(ctx, e.hub, "fake/m1", "hello", omnillm.Interactive) // pauses (JP→CN)
	_ = e.hub.ConfirmRegion(ctx)
	before := e.fake.calls.Load()
	if _, _, err := ask(ctx, e.hub, "fake/m1", "hello", omnillm.Interactive); !errors.Is(err, omnillm.ErrRegionNotAllowed) {
		t.Fatalf("unsupported region must refuse, got %v", err)
	}
	if e.fake.calls.Load() != before {
		t.Fatal("request sent from an unsupported region")
	}
}

func TestUnknownRegionRefuses(t *testing.T) {
	e := setup(t, func(o *hub.Options) {
		o.EgressRegion = func(context.Context) (string, error) { return "", errors.New("lookup failed") }
	})
	if _, _, err := ask(context.Background(), e.hub, "fake/m1", "hello", omnillm.Interactive); !errors.Is(err, omnillm.ErrRegionNotAllowed) {
		t.Fatalf("unknown region must not be guessed: %v", err)
	}
}

func TestBadProviderKeyIsNotSaved(t *testing.T) {
	e := setup(t, nil)
	ctx := context.Background()
	_, err := e.hub.AddProvider(ctx, provider.Config{Name: "other", Kind: provider.OpenAI, BaseURL: e.fake.URL, NeedsKey: true}, "sk-wrong-key-000000000000")
	if !errors.Is(err, omnillm.ErrCredential) {
		t.Fatalf("want credential problem, got %v", err)
	}
	if _, err := e.hubSec.Get(ctx, "provider/other/key"); err == nil {
		t.Fatal("a rejected key was saved")
	}
	ps, _ := e.hub.Providers(ctx)
	for _, p := range ps {
		if strings.Contains(p.KeyLast4+p.BaseURL, "SENTINEL") {
			t.Fatal("provider listing leaks the key")
		}
		if p.Name == "fake" && p.KeyLast4 != sentinelKey[len(sentinelKey)-4:] {
			t.Fatalf("last4 %q", p.KeyLast4)
		}
	}
}

func TestHumanCannotBeImpersonatedByApps(t *testing.T) {
	// Apps are always Background, whatever they claim (checked via the
	// local API package too); here: the hub's principal forcing.
	e := setup(t, nil)
	ctx := context.Background()
	_ = e.hub.SetModel(ctx, "fake/m2", hub.ModelUpdate{HumanOnly: ptr(true)})
	_, tok, _ := e.hub.AddApp(ctx, "bot", nil, 0)
	s, err := e.hub.CompleteAsApp(ctx, tok, omnillm.Request{Model: "fake/m2", Messages: []omnillm.Message{{Role: "user", Content: "hello"}}, Caller: omnillm.Interactive})
	if s != nil {
		s.Close()
	}
	if !errors.Is(err, omnillm.ErrModelNotAllowed) {
		t.Fatalf("app claiming Interactive reached a human-only model: %v", err)
	}
	if _, err := e.hub.CompleteAsApp(ctx, "omni_app_bogus", omnillm.Request{Model: "fake/m1", Messages: []omnillm.Message{{Role: "user", Content: "hello"}}}); !errors.Is(err, omnillm.ErrInvalidToken) {
		t.Fatalf("bogus token: %v", err)
	}
}

func TestToolCallAllowlist(t *testing.T) {
	e := setup(t, nil)
	_, sec := e.pairOK()
	ctx := context.Background()
	host := &fakeTools{}
	d, _ := device.Open(ctx, device.Options{Secrets: sec, Tools: host})
	defer d.Close()
	_, _ = d.Models(ctx) // connect
	devs, _ := e.hub.Devices(ctx)
	id := devs[0].ID
	res, err := e.hub.CallTool(ctx, id, "read_inbox", []byte(`{}`))
	if err != nil || string(res) != `{"ok":true}` {
		t.Fatalf("read-only tool: %s %v", res, err)
	}
	if _, err := e.hub.CallTool(ctx, id, "send_message", []byte(`{}`)); !errors.Is(err, omnillm.ErrToolNotAllowed) {
		t.Fatalf("write tool must be refused by default: %v", err)
	}
	if _, err := e.hub.CallTool(ctx, id, "not_a_tool", nil); !errors.Is(err, omnillm.ErrToolNotAllowed) {
		t.Fatalf("unknown tool: %v", err)
	}
	if host.writes.Load() != 0 {
		t.Fatal("a write tool ran")
	}
}

type fakeTools struct{ writes atomic.Int32 }

func (f *fakeTools) Tools() []device.Tool {
	return []device.Tool{{Name: "read_inbox", ReadOnly: true}, {Name: "send_message", ReadOnly: false}}
}

func (f *fakeTools) Call(_ context.Context, name string, _ json.RawMessage) (json.RawMessage, error) {
	if name == "send_message" {
		f.writes.Add(1)
	}
	return []byte(`{"ok":true}`), nil
}

func TestCredentialsOnlyInSecrets(t *testing.T) {
	e := setup(t, nil)
	ctx := context.Background()
	v, err := e.hubSec.Get(ctx, "provider/fake/key")
	if err != nil || string(v) != sentinelKey {
		t.Fatal("key not in the secret store")
	}
	if got, _ := e.fake.lastAuth.Load().(string); got != "Bearer "+sentinelKey {
		t.Fatal("provider did not receive the key")
	}
}

func providerCfg(url string) provider.Config {
	return provider.Config{Name: "fake", Kind: provider.OpenAI, BaseURL: url, NeedsKey: true}
}

func readFile(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func ptr[T any](v T) *T { return &v }

func hubModelHumanOnly() hub.ModelUpdate { return hub.ModelUpdate{HumanOnly: ptr(true)} }

func TestANewHubSaysSoOnceForTheRelaysDailyCount(t *testing.T) {
	e := setup(t, nil)
	rs := e.relay.Config.Handler.(*relay.Server)
	if today, _ := rs.Stats(); today.Hubs != 1 || today.New != 1 {
		t.Fatalf("a new hub's first connection: %+v", today)
	}
	// The same hub again (a restart): known, not new.
	ctx := context.Background()
	sctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	_ = e.hub.Stop(sctx)
	cancel()
	h, err := hub.New(ctx, hub.Options{Secrets: e.hubSec, Store: e.hubSt, RelayURL: e.relayWS, RelayToken: relayToken, Logf: t.Logf})
	if err != nil {
		t.Fatal(err)
	}
	if err := h.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c, cl := context.WithTimeout(ctx, 5*time.Second); defer cl(); _ = h.Stop(c) })
	deadline := time.Now().Add(10 * time.Second)
	for !h.RelayUp() && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if today, _ := rs.Stats(); today.Hubs != 1 || today.New != 1 {
		t.Fatalf("after a restart: %+v", today)
	}
}
