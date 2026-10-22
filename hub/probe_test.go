package hub

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	omnillm "github.com/wensuper/OmniLLM"

	"github.com/wensuper/OmniLLM/internal/egress"
	"github.com/wensuper/OmniLLM/internal/provider"
	"github.com/wensuper/OmniLLM/secret"
	"github.com/wensuper/OmniLLM/store"
)

// fakeClaude is a stand-in Claude Code that names the exact model behind
// each alias and logs every prompt it is sent.
func fakeClaude(t *testing.T) (path, log string) {
	t.Helper()
	// Not t.TempDir: adding a subscription starts its safety check in the
	// background (prepare), which may still run this program and append to
	// its log after the test returns; t.TempDir's cleanup then fails on a
	// directory that is not empty.
	dir, err := os.MkdirTemp("", "fakeclaude-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	log = filepath.Join(dir, "calls")
	script := `#!/bin/sh
if [ "$1" = auth ]; then echo '{"loggedIn":true}'; exit 0; fi
if [ "$1" = --version ]; then echo 9.9.9; exit 0; fi
m=default
prev=
for a in "$@"; do [ "$prev" = --model ] && m=$a; prev=$a; done
p=$(cat)
echo "$m $p" | head -c 40 >> ` + log + `
echo >> ` + log + `
echo '{"type":"system","subtype":"init","model":"claude-'$m'-4-5-20250929"}'
echo '{"type":"result","is_error":false,"result":"ok","usage":{"input_tokens":1,"output_tokens":1}}'
`
	path = filepath.Join(dir, "claude")
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	return path, log
}

func probes(t *testing.T, log string) []string {
	b, _ := os.ReadFile(log)
	var out []string
	for _, l := range strings.Split(string(b), "\n") {
		if strings.HasSuffix(l, " ok") { // the probe's prompt; the canary's is longer
			out = append(out, strings.TrimSuffix(l, " ok"))
		}
	}
	return out
}

func waitProbes(t *testing.T, h *Hub) {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	time.Sleep(50 * time.Millisecond)
	for h.probeRunning() {
		if time.Now().After(deadline) {
			t.Fatal("probe never finished")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestModelVersionsAreAskedOncePerWeek(t *testing.T) {
	defer func(d, g time.Duration) { probeDelay, probeGap = d, g }(probeDelay, probeGap)
	probeDelay, probeGap = 0, 0
	path, log := fakeClaude(t)
	st, sec := store.NewMemory(), secret.NewMemory()
	now := time.Now()
	clock := func() time.Time { return now }
	ctx := context.Background()
	open := func() *Hub {
		h, err := New(ctx, Options{Secrets: sec, Store: st, Now: clock})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = h.Stop(ctx) })
		return h
	}
	start := func() *Hub {
		h := open()
		if err := h.Start(ctx); err != nil {
			t.Fatal(err)
		}
		return h
	}
	// Added before Start, so no probe runs yet: the names before it.
	h := open()
	ms, err := h.AddProvider(ctx, provider.Config{Name: "claude-code", Kind: provider.ClaudeCode, Path: path}, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range ms {
		if !m.Latest || m.Kind != "claude-code" || strings.ContainsAny(m.DisplayName, "（(") || !strings.HasPrefix(m.DisplayName, "Claude ") {
			t.Fatalf("unknown version: %+v", m)
		}
	}
	if err := h.Start(ctx); err != nil {
		t.Fatal(err)
	}
	waitProbes(t, h)
	if got := probes(t, log); len(got) != 3 {
		t.Fatalf("probes %v", got)
	}
	all, _ := h.Models(ctx)
	for _, m := range all {
		if m.Latest || !strings.HasPrefix(m.DisplayName, "Claude ") || !strings.HasSuffix(m.DisplayName, " 4.5") {
			t.Fatalf("after the probe: %q", m.DisplayName)
		}
	}
	// Not charged to anyone.
	if u, _ := h.Usage(ctx, ""); len(u) != 0 {
		t.Fatalf("probe was charged: %+v", u)
	}
	_ = h.Stop(ctx)

	// A restart within the week asks nothing.
	h = start()
	waitProbes(t, h)
	if got := probes(t, log); len(got) != 3 {
		t.Fatalf("asked again within a week: %v", got)
	}
	// A real call keeps an alias fresh; eight days on, only the others
	// are asked again.
	now = now.Add(8 * 24 * time.Hour)
	h.noteModel("claude-code/opus", "claude-opus-4-5")
	_ = h.Stop(ctx)
	h = start()
	waitProbes(t, h)
	got := probes(t, log)
	if len(got) != 5 || strings.Contains(strings.Join(got[3:], " "), "opus") {
		t.Fatalf("after a week: %v", got)
	}
}

func TestProbeIsNotCountedWhenTheLimiterRefusesIt(t *testing.T) {
	path, log := fakeClaude(t)
	ctx := context.Background()
	h, err := New(ctx, Options{Secrets: secret.NewMemory(), Store: store.NewMemory()})
	if err != nil {
		t.Fatal(err)
	}
	c, _ := provider.New(provider.Config{Name: "claude-code", Kind: provider.ClaudeCode, Path: path}, nil, h.lim)
	// The subscription is cooling down after its usage limit.
	rel, err := h.lim.Acquire(ctx, "claude-code", omnillm.Interactive)
	if err != nil {
		t.Fatal(err)
	}
	rel(egress.LimitReached, time.Hour)
	m := Model{ID: "claude-code/sonnet", Provider: "claude-code", Upstream: "sonnet"}
	if h.probe(ctx, c, m) {
		t.Fatal("went on after a refusal")
	}
	if len(probes(t, log)) != 0 {
		t.Fatal("a call went out")
	}
	if !h.needsProbe(ctx, m.ID) {
		t.Fatal("a refused probe counted toward the week")
	}
}

func TestRemovedSubscriptionIsNotDetectedAgain(t *testing.T) {
	path, _ := fakeClaude(t)
	t.Setenv("PATH", filepath.Dir(path))
	t.Setenv("HOME", t.TempDir())
	t.Setenv("CODEX_HOME", t.TempDir())
	ctx := context.Background()
	h, err := New(ctx, Options{Secrets: secret.NewMemory(), Store: store.NewMemory()})
	if err != nil {
		t.Fatal(err)
	}
	detected := func() bool {
		for _, c := range h.DetectSubscriptions(ctx) {
			if c.Kind == provider.ClaudeCode {
				return true
			}
		}
		return false
	}
	if !detected() {
		t.Fatal("a logged-in Claude Code was not detected")
	}
	if _, err := h.AddProvider(ctx, provider.Config{Name: "claude-code", Kind: provider.ClaudeCode, Path: path}, ""); err != nil {
		t.Fatal(err)
	}
	if err := h.RemoveProvider(ctx, "claude-code"); err != nil {
		t.Fatal(err)
	}
	if detected() {
		t.Fatal("removed by the owner, then detected again")
	}
	// Added again on purpose: offered again after a later removal is undone.
	if _, err := h.AddProvider(ctx, provider.Config{Name: "claude-code", Kind: provider.ClaudeCode, Path: path}, ""); err != nil {
		t.Fatal(err)
	}
	var removed bool
	if store.GetJSON(ctx, h.opts.Store, removedKey(provider.ClaudeCode), &removed) == nil {
		t.Fatal("the removal mark outlived an explicit add")
	}
}

func TestProbeFollowsTheConnectedClient(t *testing.T) {
	defer func(d, g time.Duration) { probeDelay, probeGap = d, g }(probeDelay, probeGap)
	probeDelay, probeGap = 0, 0
	path, log := fakeClaude(t)
	ctx := context.Background()
	h, err := New(ctx, Options{Secrets: secret.NewMemory(), Store: store.NewMemory()})
	if err != nil {
		t.Fatal(err)
	}
	cfg := provider.Config{Name: "claude-code", Kind: provider.ClaudeCode, Path: path}
	if _, err := h.AddProvider(ctx, cfg, ""); err != nil {
		t.Fatal(err)
	}
	old := h.clients["claude-code"]
	// Added again (say, removed and re-added): the old client's probe
	// sends nothing.
	if _, err := h.AddProvider(ctx, cfg, ""); err != nil {
		t.Fatal(err)
	}
	h.probeModels(ctx, old)
	if got := probes(t, log); len(got) != 0 {
		t.Fatalf("a replaced client was probed: %v", got)
	}
	// A probe still running for the old client does not keep the new one
	// from being probed.
	if err := h.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = h.Stop(ctx) })
	waitProbes(t, h)
	if got := probes(t, log); len(got) != 3 {
		t.Fatalf("probes %v", got)
	}
	h.mu.Lock()
	h.probing[old] = true
	h.probeLaterLocked(old) // already running: not started twice
	n := len(h.probing)
	delete(h.probing, old)
	h.mu.Unlock()
	if n != 1 {
		t.Fatalf("probing %d clients", n)
	}
	// A model removed while the probe waits is not asked about.
	_ = h.opts.Store.Delete(ctx, "probed/claude-code/opus")
	h.mu.Lock()
	delete(h.resolved, "claude-code/opus")
	h.mu.Unlock()
	if err := h.RemoveModel(ctx, "claude-code/opus"); err != nil {
		t.Fatal(err)
	}
	h.probeModels(ctx, h.clients["claude-code"])
	if got := probes(t, log); len(got) != 3 {
		t.Fatalf("a removed model was probed: %v", got)
	}
}

func TestDetectNowConnectsEvenARemovedSubscription(t *testing.T) {
	path, _ := fakeClaude(t)
	t.Setenv("PATH", filepath.Dir(path))
	t.Setenv("HOME", t.TempDir())
	t.Setenv("CODEX_HOME", t.TempDir())
	ctx := context.Background()
	h, err := New(ctx, Options{Secrets: secret.NewMemory(), Store: store.NewMemory(), Transport: noOllama})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.AddProvider(ctx, provider.Config{Name: "claude-code", Kind: provider.ClaudeCode, Path: path}, ""); err != nil {
		t.Fatal(err)
	}
	if err := h.RemoveProvider(ctx, "claude-code"); err != nil {
		t.Fatal(err)
	}
	names, failed, err := h.DetectNow(ctx)
	if err != nil || len(failed) != 0 {
		t.Fatal(err, failed)
	}
	found := false
	for _, n := range names {
		found = found || n == "claude-code"
	}
	if !found || !h.subscription("claude-code") {
		t.Fatalf("not connected again on request: %v", names)
	}
	var removed bool
	if store.GetJSON(ctx, h.opts.Store, removedKey(provider.ClaudeCode), &removed) == nil {
		t.Fatal("the removal mark outlived an explicit check")
	}
	// Asked again: nothing new to connect, and that is not an error.
	names, failed, err = h.DetectNow(ctx)
	for _, n := range names {
		if n == "claude-code" {
			t.Fatalf("connected twice: %v", names)
		}
	}
	if err != nil || len(failed) != 0 {
		t.Fatal(err, failed)
	}
}

func TestSignedOutSubscriptionShowsBeforeAnyCall(t *testing.T) {
	path, _ := fakeClaude(t)
	// The same stand-in, but its sign-in follows a flag file.
	flag := filepath.Join(filepath.Dir(path), "signed-out")
	b, _ := os.ReadFile(path)
	s := strings.Replace(string(b), `if [ "$1" = auth ]; then echo '{"loggedIn":true}'; exit 0; fi`,
		`if [ "$1" = auth ]; then if [ -e `+flag+` ]; then echo '{"loggedIn":false}'; else echo '{"loggedIn":true}'; fi; exit 0; fi`, 1)
	if err := os.WriteFile(path, []byte(s), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", filepath.Dir(path))
	t.Setenv("HOME", t.TempDir())
	ctx := context.Background()
	h, err := New(ctx, Options{Secrets: secret.NewMemory(), Store: store.NewMemory()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.AddProvider(ctx, provider.Config{Name: "claude-code", Kind: provider.ClaudeCode, Path: path}, ""); err != nil {
		t.Fatal(err)
	}
	problems := func() map[string]bool {
		ms, _ := h.Models(ctx)
		out := map[string]bool{}
		for _, m := range ms {
			out[m.Problem] = true
		}
		return out
	}
	h.checkLogins(ctx)
	if p := problems(); p["login"] {
		t.Fatalf("signed in, but the models say login: %v", p)
	}
	if err := os.WriteFile(flag, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	h.checkLogins(ctx)
	if p := problems(); !p["login"] || len(p) != 1 {
		t.Fatalf("signed out, but the models don't say so: %v", p)
	}
	_ = os.Remove(flag)
	h.checkLogins(ctx)
	if p := problems(); p["login"] {
		t.Fatalf("signed in again, but the models still say login: %v", p)
	}
}
