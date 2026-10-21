package hub

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"time"

	omnillm "github.com/wensuper/OmniLLM"

	"github.com/wensuper/OmniLLM/internal/egress"
	"github.com/wensuper/OmniLLM/internal/provider"
	"github.com/wensuper/OmniLLM/store"
)

// DefaultWebAppURL is where the project publishes the web client.
const DefaultWebAppURL = "https://omni.ixxzh.com/"

// DefaultRelayURL is the project's public connection service.
const DefaultRelayURL = "wss://relay.ixxzh.com"

// subscription reports whether a provider runs an official program logged
// in with the owner's subscription (Claude Code, Codex).
func (h *Hub) subscription(name string) bool {
	h.mu.Lock()
	c := h.clients[name]
	h.mu.Unlock()
	return c != nil && c.Config().Kind.Subscription()
}

// subscriptionLimits paces subscription providers like one person typing:
// one call at a time, a few a minute, a short queue. A host's own limits
// for other providers are kept.
func subscriptionLimits(host func(string) egress.Limits) func(string) egress.Limits {
	return func(p string) egress.Limits {
		if provider.Kind(p).Subscription() {
			// A person may send, stop and resend a few in a row; the
			// sustained pace and the per-device caps stay low.
			return egress.Limits{MaxConcurrent: 1, RequestsPerMinute: 12, Burst: 4, MaxQueue: 4}
		}
		if host != nil {
			return host(p)
		}
		return egress.Limits{}
	}
}

func subscriptionModelName(k provider.Kind, id string) string {
	if k == provider.Codex || id == "" {
		return "ChatGPT (Codex)"
	}
	return "Claude " + strings.ToUpper(id[:1]) + id[1:]
}

// removedKey marks a subscription the owner removed: auto-detection skips
// it until it is added again on purpose.
func removedKey(k provider.Kind) string { return "settings/removed/" + string(k) }

// Per-device pace for subscription models: more than a person chats.
const (
	subsPerHour = 60
	subsPerDay  = 300
)

// subscriptionPace counts each device's subscription calls over the last
// day.
type subscriptionPace struct {
	mu    sync.Mutex
	calls map[string][]time.Time
}

func (s *subscriptionPace) allow(who string, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.calls == nil {
		s.calls = map[string][]time.Time{}
	}
	list := s.calls[who]
	i := 0
	for i < len(list) && now.Sub(list[i]) >= 24*time.Hour {
		i++
	}
	list = list[i:]
	hour := 0
	for _, t := range list {
		if now.Sub(t) < time.Hour {
			hour++
		}
	}
	var wait time.Duration
	switch {
	case len(list) >= subsPerDay:
		wait = list[0].Add(24 * time.Hour).Sub(now)
	case hour >= subsPerHour:
		wait = list[len(list)-hour].Add(time.Hour).Sub(now)
	}
	if wait > 0 {
		s.calls[who] = list
		return &omnillm.Error{Code: omnillm.CodeBusy, Msg: omnillm.MsgSubscriptionPace, RetryAfter: wait}
	}
	s.calls[who] = append(list, now)
	return nil
}

// ---------- model versions before first use ----------

// Claude Code takes family aliases ("sonnet") that follow Anthropic's
// newest models, and only a reply says which model answered. So that a
// model list shows "Claude Sonnet 4.5" before anyone has used it, the hub
// asks each alias once with a minimal prompt when the subscription is
// connected and at start, but only for an alias whose version no call has
// reported in the last week, and never more than once per alias per week.
// Cost: at most 3 tiny calls a week (a few tokens each), sent one after
// another through the same pacing and limiter as everything else, behind
// any person, and not charged to any device's daily limit.
//
// Codex's replies do not name their model (its JSON events carry no model
// field), so Codex is not asked; its name stays "ChatGPT (Codex)".

const probeEvery = 7 * 24 * time.Hour

// probeDelay leaves the first minute after start to people; probeGap keeps
// the probes from using up the subscription's burst allowance.
var probeDelay, probeGap = 30 * time.Second, 20 * time.Second

// probeLaterLocked starts the probe for c in the background, if the hub is
// running (h.mu held). Start does it for sources added before it. At most
// one probe runs per client: a source removed and added again is a new
// client and is probed, while the old client's probe stops.
func (h *Hub) probeLaterLocked(c *provider.Client) {
	if h.ctx == nil || c.Config().Kind != provider.ClaudeCode || h.probing[c] {
		return
	}
	if h.probing == nil {
		h.probing = map[*provider.Client]bool{}
	}
	h.probing[c] = true
	ctx := h.ctx
	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		defer func() {
			h.mu.Lock()
			delete(h.probing, c)
			h.mu.Unlock()
		}()
		h.probeModels(ctx, c)
	}()
}

// probeRunning reports whether any model-version probe is running.
func (h *Hub) probeRunning() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.probing) > 0
}

// stillConnected reports whether c is still the client for its source
// (not removed or replaced since the probe began).
func (h *Hub) stillConnected(c *provider.Client) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.clients[c.Config().Name] == c
}

func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return true
	case <-ctx.Done():
		return false
	}
}

// probeModels asks each alias that needs it, one at a time.
func (h *Hub) probeModels(ctx context.Context, c *provider.Client) {
	if !sleepCtx(ctx, probeDelay) {
		return
	}
	// The no-tools check first (it is usually done by now); a program that
	// fails it, or is signed out, is not asked anything.
	if err := c.Prepare(ctx); err != nil {
		return
	}
	ms, err := h.listModels(ctx)
	if err != nil {
		return
	}
	first := true
	for _, m := range ms {
		if m.Provider != c.Config().Name || !h.needsProbe(ctx, m.ID) {
			continue
		}
		if !first && !sleepCtx(ctx, probeGap) {
			return
		}
		first = false
		// The owner may have removed the source, or the model, while
		// this probe waited: then nothing is sent.
		if !h.stillConnected(c) {
			return
		}
		if _, err := h.getModel(ctx, m.ID); err != nil {
			continue
		}
		if !h.probe(ctx, c, m) {
			return
		}
	}
}

// needsProbe: the version is unknown or a week old, and the alias was not
// asked in the last week.
func (h *Hub) needsProbe(ctx context.Context, id string) bool {
	now := h.opts.Now()
	h.mu.Lock()
	known, seen := h.resolved[id] != "", h.seen[id]
	h.mu.Unlock()
	if known && now.Sub(seen) < probeEvery {
		return false
	}
	var asked time.Time
	if store.GetJSON(ctx, h.opts.Store, "probed/"+id, &asked) == nil && now.Sub(asked) < probeEvery {
		return false
	}
	return true
}

// probe sends the minimal prompt for one model and keeps the version the
// reply names. It reports whether to go on with the next alias.
func (h *Hub) probe(ctx context.Context, c *provider.Client, m Model) bool {
	if err := h.checkRegion(ctx, m.Provider); err != nil {
		return false
	}
	// Kept before the call, so a call that fails half-way still counts
	// toward once a week.
	var before time.Time
	hadBefore := store.GetJSON(ctx, h.opts.Store, "probed/"+m.ID, &before) == nil
	if err := store.PutJSON(ctx, h.opts.Store, "probed/"+m.ID, h.opts.Now()); err != nil {
		return false
	}
	cctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	// Background: behind every person in the limiter's queue. Sent to the
	// provider directly, so no device's daily limit or pace is touched.
	req := omnillm.Request{Model: m.ID, Caller: omnillm.Background, MaxTokens: 16, Messages: []omnillm.Message{
		{Role: "system", Content: "Reply with: ok"}, {Role: "user", Content: "ok"}}}
	s, err := c.Stream(cctx, m.Upstream, req, provider.Credential{})
	if err != nil {
		if errors.Is(err, omnillm.ErrBusy) {
			// The limiter refused it: nothing was sent, so it does not count.
			if hadBefore {
				_ = store.PutJSON(ctx, h.opts.Store, "probed/"+m.ID, before)
			} else {
				_ = h.opts.Store.Delete(ctx, "probed/"+m.ID)
			}
		}
		return false
	}
	defer s.Close()
	for {
		ch, err := s.Recv()
		if ch.Model != "" {
			h.noteModel(m.ID, ch.Model)
		}
		if err != nil {
			return errors.Is(err, io.EOF)
		}
	}
}

// DetectSubscriptions lists the subscription programs installed and logged
// in on this machine that are not connected yet, except those the owner
// removed.
func (h *Hub) DetectSubscriptions(ctx context.Context) []provider.Config {
	return h.detectSubscriptions(ctx, false)
}

func (h *Hub) detectSubscriptions(ctx context.Context, evenRemoved bool) []provider.Config {
	var out []provider.Config
	for _, k := range []provider.Kind{provider.ClaudeCode, provider.Codex} {
		if h.subscription(string(k)) {
			continue
		}
		var removed bool
		if !evenRemoved && store.GetJSON(ctx, h.opts.Store, removedKey(k), &removed) == nil && removed {
			continue
		}
		cfg, err := provider.Default(k)
		if err != nil {
			continue
		}
		if provider.CLILoggedIn(ctx, k, cfg.Path) {
			out = append(out, cfg)
		}
	}
	return out
}

// DetectNow is the owner asking, on purpose, to look again: every
// subscription program logged in on this machine (even one the owner
// removed earlier; asking is adding it again) and a running local Ollama
// that are not connected yet are connected now. It returns the names of
// what it connected ("Claude Code", "Codex", "Ollama": program names, the
// same in every language). An error is returned only when something was
// found and none of it could be connected.
func (h *Hub) DetectNow(ctx context.Context) ([]string, error) {
	cfgs := h.detectSubscriptions(ctx, true)
	h.mu.Lock()
	_, haveOllama := h.clients["ollama"]
	h.mu.Unlock()
	if !haveOllama {
		if ids, err := h.DetectOllama(ctx); err == nil && len(ChatModels(ids)) > 0 {
			cfg, _ := provider.Default(provider.Ollama)
			cfgs = append(cfgs, cfg)
		}
	}
	var names []string
	var firstErr error
	for _, cfg := range cfgs {
		if _, err := h.AddProvider(ctx, cfg, ""); err != nil {
			h.opts.Logf("hub: %s not connected: %s", cfg.Name, omnillm.Describe("en", err))
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		names = append(names, programName(cfg.Kind))
	}
	if len(names) == 0 && firstErr != nil {
		return nil, firstErr
	}
	return names, nil
}

// programName is the program a source runs, as people know it.
func programName(k provider.Kind) string {
	switch k {
	case provider.ClaudeCode:
		return "Claude Code"
	case provider.Codex:
		return "Codex"
	case provider.Ollama:
		return "Ollama"
	}
	return string(k)
}
