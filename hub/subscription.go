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
	// Codex names its model in its own catalog, read on this machine: no
	// call is sent, no usage counted.
	if c.Config().Kind == provider.Codex {
		if _, name, err := provider.CodexModel(ctx, c.Config().Path); err == nil {
			h.noteModel(m.ID, name)
			return true
		}
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

// DetectFailure is a source DetectNow found but could not connect: Kind
// is its kind ("claude-code", "codex", "ollama") and Err says why, for
// omnillm.Describe.
type DetectFailure struct {
	Kind string
	Err  error
}

// DetectNow is the owner asking, on purpose, to look again: every
// subscription program logged in on this machine (even one the owner
// removed earlier; asking is adding it again) and a running local Ollama
// that are not connected yet are connected now. It returns the kinds of
// what it connected ("claude-code", "codex", "ollama"; the page names
// them in its language) and, separately, what was found but could not be
// connected, each with its error (a running Ollama with no chat model is
// one, MsgNoChatModel). Finding nothing is not an error; err is
// set only when the request itself ended (ctx).
func (h *Hub) DetectNow(ctx context.Context) (connected []string, failed []DetectFailure, err error) {
	cfgs := h.detectSubscriptions(ctx, true)
	h.mu.Lock()
	_, haveOllama := h.clients["ollama"]
	h.mu.Unlock()
	if !haveOllama {
		if ids, err := h.DetectOllama(ctx); err == nil && len(ChatModels(ids)) > 0 {
			cfg, _ := provider.Default(provider.Ollama)
			cfgs = append(cfgs, cfg)
		} else if err == nil && len(ids) > 0 {
			// Running, but with embedding or rerank models only: found,
			// and the owner is told what to pull.
			failed = append(failed, DetectFailure{Kind: string(provider.Ollama),
				Err: &omnillm.Error{Code: omnillm.CodeBadRequest, Msg: omnillm.MsgNoChatModel}})
		}
	}
	for _, cfg := range cfgs {
		if _, err := h.AddProvider(ctx, cfg, ""); err != nil {
			h.opts.Logf("hub: %s not connected: %s", cfg.Name, omnillm.Describe("en", err))
			failed = append(failed, DetectFailure{Kind: string(cfg.Kind), Err: err})
			continue
		}
		connected = append(connected, string(cfg.Kind))
	}
	if err := ctx.Err(); err != nil {
		return connected, failed, err
	}
	return connected, failed, nil
}

// SubscriptionState is how far one subscription (Claude, ChatGPT) is from
// being usable here, so a page can show the one step left. The steps, in
// order: the official program installed (Installed), logged in where the
// hub can use the login (LoggedIn), connected (Connected). Desktop says
// the subscription's desktop app is on this computer (the same account
// signs in to the program; its name, "" for none); Elsewhere, Codex only, that Codex is logged in
// but keeps the login in the system keychain, out of the hub's reach.
// Removed: the owner removed it, so it is not connected again by itself.
// Problem is why connecting a logged-in program failed (for Describe).
type SubscriptionState struct {
	Kind      string `json:"kind"`
	Installed bool   `json:"installed"`
	LoggedIn  bool   `json:"logged_in"`
	Connected bool   `json:"connected"`
	Desktop   string `json:"desktop"`
	Elsewhere bool   `json:"elsewhere"`
	Removed   bool   `json:"removed"`
	Problem   error  `json:"-"`
}

// subStatesFor is how long a SubscriptionStates answer is reused.
const subStatesFor = 2 * time.Second

// SubscriptionStates says, for Claude and ChatGPT, what is done and what is
// left. A subscription that became logged in since (and that the owner
// did not remove) is connected on the way, so a page that polls this while
// someone logs in sees it connected without another click.
func (h *Hub) SubscriptionStates(ctx context.Context) []SubscriptionState {
	h.subMu.Lock()
	defer h.subMu.Unlock()
	if !h.subAt.IsZero() && time.Since(h.subAt) < subStatesFor && !h.subStale.Load() {
		return h.subLast
	}
	// Its own time limit, not the asking page's: a page closed meanwhile
	// must not leave a half-done answer (killed checks read as "signed
	// out") for the next one.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Minute)
	defer cancel()
	h.subStale.Store(false)
	var out []SubscriptionState
	for _, k := range []provider.Kind{provider.ClaudeCode, provider.Codex} {
		st := SubscriptionState{Kind: string(k), Desktop: provider.DesktopApp(k), Connected: h.subscription(string(k))}
		var removed bool
		st.Removed = store.GetJSON(ctx, h.opts.Store, removedKey(k), &removed) == nil && removed
		if cfg, err := provider.Default(k); err == nil {
			st.Installed = true
			st.LoggedIn = st.Connected || provider.CLILoggedIn(ctx, k, cfg.Path)
			if !st.LoggedIn && k == provider.Codex {
				// The owner's own Codex settings may reach the keychain:
				// asked at most every half minute, not on every poll.
				if time.Since(h.codexOwnAt) > 30*time.Second {
					h.codexOwn, h.codexOwnAt = provider.CodexLoggedInOwn(ctx, cfg.Path), time.Now()
				}
				st.Elsewhere = h.codexOwn
			}
			if st.LoggedIn && !st.Connected && !st.Removed {
				if _, err := h.AddProvider(ctx, cfg, ""); err != nil {
					st.Problem = err
				} else {
					st.Connected = true
				}
			}
		}
		out = append(out, st)
	}
	h.subAt, h.subLast = time.Now(), out
	return out
}

// ConnectSubscription connects one subscription ("claude-code" or
// "codex") whose program is signed in: the owner asked for it (a button,
// or omnillm login), so one removed earlier comes back too. Nothing else
// is looked for.
func (h *Hub) ConnectSubscription(ctx context.Context, kind string) error {
	k := provider.Kind(kind)
	if !k.Subscription() {
		return &omnillm.Error{Code: omnillm.CodeBadRequest, Msg: omnillm.MsgNotFound}
	}
	cfg, err := provider.Default(k)
	if err != nil {
		return &omnillm.Error{Code: omnillm.CodeCredentialProblem, Msg: omnillm.MsgCLINotFound, Args: []string{kind}}
	}
	_, err = h.AddProvider(ctx, cfg, "")
	return err
}

// loginsEvery is how often the hub asks the subscription programs whether
// they are still signed in; a device asking for its models asks again when
// the last look is older than loginsStale (in the background: the list it
// gets now says what is known).
var loginsEvery, loginsStale = 5 * time.Minute, 30 * time.Second

// watchLogins looks at the subscription programs' sign-in at start and
// every few minutes, so a signed-out Claude Code or Codex shows on every
// screen before a message fails on it, and clears once signed in again.
func (h *Hub) watchLogins(ctx context.Context) {
	h.checkLogins(ctx)
	t := time.NewTicker(loginsEvery)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			h.checkLogins(ctx)
		}
	}
}

// freshLogins starts a sign-in check when the last one is old.
func (h *Hub) freshLogins() {
	if time.Since(time.Unix(0, h.loginsAt.Load())) >= loginsStale {
		go h.checkLogins(h.ctx) // runs one at a time by itself
	}
}

// checkLogins asks each connected subscription program whether it is
// signed in (its own status command; no model is called). Codex counts as
// signed in through the owner's own login too, as the hub runs it.
func (h *Hub) checkLogins(ctx context.Context) {
	if !h.loginsBusy.CompareAndSwap(false, true) {
		return
	}
	defer h.loginsBusy.Store(false)
	h.loginsAt.Store(time.Now().UnixNano())
	h.mu.Lock()
	var subs []*provider.Client
	for _, c := range h.clients {
		if c.Config().Kind.Subscription() {
			subs = append(subs, c)
		}
	}
	h.mu.Unlock()
	for _, c := range subs {
		cfg := c.Config()
		in := provider.CLILoggedIn(ctx, cfg.Kind, cfg.Path)
		if !in && cfg.Kind == provider.Codex {
			in = provider.CodexLoggedInOwn(ctx, cfg.Path)
		}
		if ctx.Err() != nil {
			return // a check cut short says nothing
		}
		c.SignedOut(!in)
	}
}
