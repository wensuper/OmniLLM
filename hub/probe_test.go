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
	dir := t.TempDir()
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
	for h.probing.Load() {
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
	start := func() *Hub {
		h, err := New(ctx, Options{Secrets: sec, Store: st, Lang: "zh", Now: clock})
		if err != nil {
			t.Fatal(err)
		}
		if err := h.Start(ctx); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = h.Stop(ctx) })
		return h
	}
	h := start()
	h.probing.Store(true) // hold the probe back to see the names before it
	ms, err := h.AddProvider(ctx, provider.Config{Name: "claude-code", Kind: provider.ClaudeCode, Path: path}, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range ms {
		if !strings.HasSuffix(m.DisplayName, "（最新版）") {
			t.Fatalf("unknown version named %q", m.DisplayName)
		}
	}
	h.probing.Store(false)
	h.mu.Lock()
	h.probeLaterLocked(h.clients["claude-code"])
	h.mu.Unlock()
	waitProbes(t, h)
	if got := probes(t, log); len(got) != 3 {
		t.Fatalf("probes %v", got)
	}
	all, _ := h.Models(ctx)
	for _, m := range all {
		if !strings.HasPrefix(m.DisplayName, "Claude ") || !strings.HasSuffix(m.DisplayName, " 4.5") {
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
