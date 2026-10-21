// Package hub is the hub role: the only place that talks to model providers
// (principle 1), holds credentials (principle 3), and serves paired devices
// over end-to-end encrypted sessions through the relay (principle 2).
//
// A host (NEXA, or the standalone omnillm program) creates a Hub with New,
// passes in its secret store, store, transport, logger and locale, and
// decides when to Start and Stop it. The Hub opens no ports and writes
// nothing outside what the host gave it.
package hub

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	omnillm "github.com/wensuper/OmniLLM"
	"github.com/wensuper/OmniLLM/internal/e2e"
	"github.com/wensuper/OmniLLM/internal/egress"
	"github.com/wensuper/OmniLLM/internal/provider"
	"github.com/wensuper/OmniLLM/internal/redact"
	"github.com/wensuper/OmniLLM/internal/relaylink"
	"github.com/wensuper/OmniLLM/internal/relayproto"
	"github.com/wensuper/OmniLLM/secret"
	"github.com/wensuper/OmniLLM/store"
)

// Options is everything the host decides.
type Options struct {
	Secrets secret.Store // required
	Store   store.Store  // required
	// Transport carries every outbound request (providers and relay). NEXA
	// passes its egress; nil uses a guarded default that refuses private
	// addresses for remote providers.
	Transport http.RoundTripper
	// RelayURL and RelayToken connect the hub to its relay. Empty RelayURL
	// runs without devices (in-process use only).
	RelayURL   string
	RelayToken string
	// WebAppURL is the web client phones and tablets open from the pairing
	// QR code (default DefaultWebAppURL).
	WebAppURL string
	// Logf receives redacted operational logs. Optional.
	Logf func(format string, args ...any)
	// Limits per provider (nil: defaults, 2 concurrent).
	Limits func(provider string) egress.Limits
	// DefaultBudget per device / app per day, micro-USD (default $5).
	DefaultBudget int64
	// GlobalBudget for the whole hub per day, micro-USD (default $20).
	GlobalBudget int64
	// HostBudget for in-process calls by the host per day (default $5).
	HostBudget int64
	// EgressRegion, when set, reports the country of the hub's outbound
	// address. A change pauses upstream calls until ConfirmRegion.
	EgressRegion func(ctx context.Context) (string, error)
	// RegionAllowed, when set, says whether a provider serves a country.
	RegionAllowed func(provider, country string) bool
	// Now is for tests.
	Now func() time.Time
}

// Hub is the hub role.
type Hub struct {
	opts Options
	// resolved: model id -> its name with version, as last reported (mu).
	resolved map[string]string
	// seen: model id -> when a call last reported its version (mu).
	seen map[string]time.Time
	// probing: clients whose model-version probe is running (mu).
	probing map[*provider.Client]bool
	lim     *egress.Limiter
	subs    subscriptionPace
	key     e2e.KeyPair
	// chats: kept sessions of people's conversations with subscription
	// programs (provider/chats.go).
	chats *provider.Chats

	mu          sync.Mutex
	clients     map[string]*provider.Client
	link        *relaylink.HubLink
	sessions    map[relayproto.ID]*mailboxConn
	pairings    map[relayproto.ID]*pairing
	devSessions []sessionEntry
	deviceMu    sync.Mutex       // serializes device record writes
	reserved    map[string]int64 // worst-case cost of calls in flight (accountMu)
	accountMu   sync.Mutex
	inFlight    atomic.Int64 // calls whose reply is still coming (Busy)
	region      regionState

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

const (
	keyHubStatic = "hub/static_key"
	dollar       = int64(1_000_000)
)

// New creates a hub. It loads or creates the hub's key and loads providers.
func New(ctx context.Context, opts Options) (*Hub, error) {
	if opts.Secrets == nil || opts.Store == nil {
		return nil, errors.New("hub: Secrets and Store are required")
	}
	if opts.Logf == nil {
		opts.Logf = func(string, ...any) {}
	}
	logf := opts.Logf
	opts.Logf = func(format string, args ...any) { logf("%s", redact.String(fmt.Sprintf(format, args...))) }
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.WebAppURL == "" {
		opts.WebAppURL = DefaultWebAppURL
	}
	if opts.DefaultBudget <= 0 {
		opts.DefaultBudget = 5 * dollar
	}
	if opts.GlobalBudget <= 0 {
		opts.GlobalBudget = 20 * dollar
	}
	if opts.HostBudget <= 0 {
		opts.HostBudget = 5 * dollar
	}
	h := &Hub{opts: opts, lim: egress.New(subscriptionLimits(opts.Limits)), clients: map[string]*provider.Client{},
		sessions: map[relayproto.ID]*mailboxConn{}, pairings: map[relayproto.ID]*pairing{}}
	if err := h.loadKey(ctx); err != nil {
		return nil, err
	}
	if err := h.loadProviders(ctx); err != nil {
		return nil, err
	}
	if err := h.loadRegion(ctx); err != nil {
		return nil, err
	}
	h.loadResolved(ctx)
	h.loadChats(ctx)
	// The owner's own daily total, if they set one in the console.
	var total int64
	if err := store.GetJSON(ctx, h.opts.Store, dailyTotalKey, &total); err == nil && (total > 0 || total == NoLimit) {
		h.opts.GlobalBudget = total
	}
	return h, nil
}

func (h *Hub) loadKey(ctx context.Context) error {
	b, err := h.opts.Secrets.Get(ctx, keyHubStatic)
	if errors.Is(err, secret.ErrNotFound) {
		k, err := e2e.GenerateKey()
		if err != nil {
			return err
		}
		if err := h.opts.Secrets.Set(ctx, keyHubStatic, k.Marshal()); err != nil {
			return err
		}
		h.key = k
		return nil
	}
	if err != nil {
		return err
	}
	h.key, err = e2e.UnmarshalKey(b)
	return err
}

// PublicKey is the hub's static public key (safe to show).
func (h *Hub) PublicKey() []byte { return append([]byte{}, h.key.Public...) }

// Start connects to the relay (if configured) and serves devices until Stop.
// A Hub is started once; a failed Start may be retried.
func (h *Hub) Start(ctx context.Context) error {
	h.mu.Lock()
	started := h.cancel != nil
	h.mu.Unlock()
	if started {
		return errors.New("hub: already started")
	}
	// Apps with the same name (from before names were made unique) get a
	// number once, so they can be told apart everywhere.
	_, _ = h.Apps(ctx)
	var link *relaylink.HubLink
	if h.opts.RelayURL != "" {
		var err error
		link, err = relaylink.NewHubLink(h.opts.RelayURL, h.opts.RelayToken, h.opts.Transport, h.opts.Logf)
		if err != nil {
			return err
		}
		devs, err := h.listDevices(ctx)
		if err != nil {
			return err
		}
		for _, t := range h.tombstones(ctx) {
			devs = append(devs, t.Device)
		}
		for _, d := range devs {
			id, sh, err := d.mailbox()
			if err != nil {
				continue
			}
			_ = link.Claim(ctx, id, sh)
		}
	}
	h.mu.Lock()
	if h.cancel != nil {
		h.mu.Unlock()
		return errors.New("hub: already started")
	}
	h.ctx, h.cancel = context.WithCancel(context.WithoutCancel(ctx))
	h.link = link
	h.mu.Unlock()
	if link != nil {
		h.wg.Add(2)
		go func() { defer h.wg.Done(); link.Run(h.ctx) }()
		go func() { defer h.wg.Done(); h.dispatch(link) }()
	}
	h.mu.Lock()
	for _, c := range h.clients {
		h.probeLaterLocked(c)
	}
	h.mu.Unlock()
	h.chats.Sweep() // here, not in the loop: Start's caller owns the clock
	h.wg.Add(1)
	go func() { defer h.wg.Done(); h.sweepChats(h.ctx) }()
	return nil
}

// Stop ends every session and the relay link, and waits for goroutines.
func (h *Hub) Stop(ctx context.Context) error {
	h.mu.Lock()
	cancel := h.cancel
	for _, s := range h.sessions {
		s.close()
	}
	h.mu.Unlock()
	if cancel == nil {
		return nil
	}
	cancel()
	done := make(chan struct{})
	go func() { h.wg.Wait(); close(done) }()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

const dailyTotalKey = "settings/daily_total"

// DailyTotal is the hub's daily limit for everything together, micro-USD
// (or NoLimit).
func (h *Hub) DailyTotal() int64 {
	h.accountMu.Lock()
	defer h.accountMu.Unlock()
	return h.opts.GlobalBudget
}

// SetDailyTotal changes and keeps the hub's daily total (micro-USD, or
// NoLimit: then only each device's and app's own limit applies).
func (h *Hub) SetDailyTotal(ctx context.Context, v int64) error {
	if v != NoLimit && v <= 0 {
		return &omnillm.Error{Code: omnillm.CodeBadRequest, Msg: omnillm.MsgBadBudget}
	}
	if err := store.PutJSON(ctx, h.opts.Store, dailyTotalKey, v); err != nil {
		return err
	}
	h.accountMu.Lock()
	h.opts.GlobalBudget = v
	h.accountMu.Unlock()
	return nil
}

const uiLangKey = "settings/ui_lang"

// UILang is the language the owner chose for the hub's own screens (the
// console and the menu bar): "zh" or "en", or "" when not chosen (each
// screen follows its system's language then).
func (h *Hub) UILang(ctx context.Context) string {
	var l string
	if err := store.GetJSON(ctx, h.opts.Store, uiLangKey, &l); err != nil || (l != "zh" && l != "en") {
		return ""
	}
	return l
}

// SetUILang keeps the owner's choice of language: "zh", "en", or "" to
// follow the system again.
func (h *Hub) SetUILang(ctx context.Context, l string) error {
	if l != "" && l != "zh" && l != "en" {
		return errors.New("hub: language must be zh, en or empty")
	}
	return store.PutJSON(ctx, h.opts.Store, uiLangKey, l)
}

// RelayUp reports whether devices can reach the hub right now.
func (h *Hub) RelayUp() bool {
	h.mu.Lock()
	l := h.link
	h.mu.Unlock()
	return l != nil && l.Up()
}

// ---------- providers, credentials, models ----------

// ProviderInfo is safe to show: the key appears only as its last four.
type ProviderInfo struct {
	Name     string        `json:"name"`
	Kind     provider.Kind `json:"kind"`
	BaseURL  string        `json:"base_url"`
	KeyLast4 string        `json:"key_last4,omitempty"`
	Local    bool          `json:"local"`
	// Subscription: an official program logged in with the owner's
	// subscription (no key here; the login stays in that program).
	Subscription bool   `json:"subscription,omitempty"`
	DisplayName  string `json:"display_name"`
	// Problem is what the owner should fix, as the last call showed it:
	// "login" (the program is signed out) or "limit" (the subscription's
	// usage limit; ResetsAt says when it resets, unix seconds). The next
	// successful call clears it.
	Problem  string `json:"problem,omitempty"`
	ResetsAt int64  `json:"resets_at,omitempty"`
}

func providerKey(name string) string { return "provider/" + name + "/key" }

func (h *Hub) loadProviders(ctx context.Context) error {
	keys, err := h.opts.Store.List(ctx, "provider/")
	if err != nil {
		return err
	}
	for _, k := range keys {
		var cfg provider.Config
		if err := store.GetJSON(ctx, h.opts.Store, k, &cfg); err != nil {
			return err
		}
		c, err := provider.New(cfg, h.opts.Transport, h.lim)
		if err != nil {
			h.opts.Logf("hub: skipping provider %s: %v", cfg.Name, err)
			continue
		}
		h.clients[cfg.Name] = c
		prepare(c)
	}
	return nil
}

// prepare runs a subscription program's safety check in the background,
// so a person's first message does not wait for it.
func prepare(c *provider.Client) {
	if !c.Config().Kind.Subscription() {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
		defer cancel()
		_ = c.Prepare(ctx)
	}()
}

// AddProvider connects a provider. For providers that need a key, the key is
// checked with a minimal call (listing models) before anything is saved, and
// the models it can use are added. The key is stored only in Secrets.
func (h *Hub) AddProvider(ctx context.Context, cfg provider.Config, apiKey string) ([]omnillm.ModelInfo, error) {
	apiKey = strings.TrimSpace(apiKey)
	if err := cfg.Validate(); err != nil {
		return nil, &omnillm.Error{Code: omnillm.CodeBadRequest, Msg: omnillm.MsgBadSource, Detail: err.Error()}
	}
	if !sourceName.MatchString(cfg.Name) && !h.connected(cfg.Name) {
		// A new name only; one connected before keeps working with a new key.
		return nil, &omnillm.Error{Code: omnillm.CodeBadRequest, Msg: omnillm.MsgBadSource, Detail: "provider: bad name"}
	}
	// A key cut short or pasted with a line break is caught here, before
	// any request is sent with it.
	if cfg.NeedsKey && (len(apiKey) < 8 || len(apiKey) > 512 || strings.ContainsAny(apiKey, " \t\r\n")) {
		return nil, &omnillm.Error{Code: omnillm.CodeCredentialProblem, Msg: omnillm.MsgKeyMalformed}
	}
	c, err := provider.New(cfg, h.opts.Transport, h.lim)
	if err != nil {
		return nil, err
	}
	ids, err := c.ListModels(ctx, provider.Credential{Key: apiKey})
	if err != nil {
		return nil, err
	}
	if cfg.Kind == provider.Ollama {
		// Local servers also list embedding and rerank models, which
		// cannot hold a conversation.
		ids = ChatModels(ids)
		if len(ids) == 0 {
			return nil, &omnillm.Error{Code: omnillm.CodeBadRequest, Msg: omnillm.MsgNoChatModel}
		}
	}
	if cfg.NeedsKey {
		if err := h.opts.Secrets.Set(ctx, providerKey(cfg.Name), []byte(apiKey)); err != nil {
			return nil, err
		}
	}
	if err := store.PutJSON(ctx, h.opts.Store, "provider/"+cfg.Name, cfg); err != nil {
		return nil, err
	}
	if cfg.Kind.Subscription() {
		// Added on purpose: auto-detection may offer it again later.
		_ = h.opts.Store.Delete(ctx, removedKey(cfg.Kind))
	}
	h.mu.Lock()
	h.clients[cfg.Name] = c
	h.mu.Unlock()
	prepare(c)
	defer func() {
		h.mu.Lock()
		h.probeLaterLocked(c)
		h.mu.Unlock()
	}()
	var added []omnillm.ModelInfo
	for _, id := range ids {
		// Re-adding a provider (a new key) keeps each model's settings:
		// prices and HumanOnly are the owner's, not the provider's.
		if old, err := h.getModel(ctx, cfg.Name+"/"+id); err == nil {
			added = append(added, h.info(old))
			continue
		}
		m := Model{ID: cfg.Name + "/" + id, Provider: cfg.Name, Upstream: id, DisplayName: id}
		if cfg.Local() {
			m.PriceIn, m.PriceOut = 0, 0
			m.PriceSet = true
		}
		if cfg.Kind.Subscription() {
			// Paid for by the subscription, not per token; for people only.
			m.PriceIn, m.PriceOut, m.PriceSet, m.HumanOnly = 0, 0, true, true
			m.DisplayName = subscriptionModelName(cfg.Kind, id)
		}
		if err := h.putModel(ctx, m); err != nil {
			return nil, err
		}
		added = append(added, h.info(m))
	}
	h.opts.Logf("hub: provider %s connected with %d models", cfg.Name, len(added))
	return added, nil
}

// sourceName is a model source's name as the owner types it ("groq").
var sourceName = regexp.MustCompile(`^[A-Za-z0-9-]{1,40}$`)

// connected reports whether a source of that name is connected now.
func (h *Hub) connected(name string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.clients[name] != nil
}

// RemoveProvider deletes the key and the provider's models.
func (h *Hub) RemoveProvider(ctx context.Context, name string) error {
	if err := h.opts.Secrets.Delete(ctx, providerKey(name)); err != nil {
		return err
	}
	if err := h.opts.Store.Delete(ctx, "provider/"+name); err != nil {
		return err
	}
	ms, _ := h.listModels(ctx)
	for _, m := range ms {
		if m.Provider == name {
			_ = h.opts.Store.Delete(ctx, "model/"+m.ID)
		}
	}
	h.mu.Lock()
	delete(h.clients, name)
	h.mu.Unlock()
	if k := provider.Kind(name); k.Subscription() {
		h.chats.ForgetKind(k)
		// The owner removed it: auto-detection must not bring it back.
		_ = store.PutJSON(ctx, h.opts.Store, removedKey(k), true)
	}
	return nil
}

// Providers lists connected providers without their keys.
func (h *Hub) Providers(ctx context.Context) ([]ProviderInfo, error) {
	h.mu.Lock()
	var clients []*provider.Client
	for _, c := range h.clients {
		clients = append(clients, c)
	}
	h.mu.Unlock()
	sort.Slice(clients, func(i, j int) bool { return clients[i].Config().Name < clients[j].Config().Name })
	var out []ProviderInfo
	for _, cl := range clients {
		c := cl.Config()
		pi := ProviderInfo{Name: c.Name, Kind: c.Kind, BaseURL: c.BaseURL, Local: c.Local(),
			Subscription: c.Kind.Subscription(), DisplayName: c.Name}
		if p, at := cl.Problem(); p != "" {
			pi.Problem = p
			if p == provider.ProblemLimit {
				pi.ResetsAt = at.Unix()
			}
		}
		if c.Kind.Subscription() {
			pi.DisplayName = provider.DisplayName(c.Kind)
		}
		if c.NeedsKey {
			if k, err := h.opts.Secrets.Get(ctx, providerKey(c.Name)); err == nil {
				pi.KeyLast4 = redact.Last4(string(k))
			}
		}
		out = append(out, pi)
	}
	return out, nil
}

// Model is a model the hub can serve.
type Model struct {
	ID          string `json:"id"`
	Provider    string `json:"provider"`
	Upstream    string `json:"upstream"`
	DisplayName string `json:"display_name"`
	// Prices in USD per million tokens. Unset prices are estimated high.
	PriceIn  float64 `json:"price_in"`
	PriceOut float64 `json:"price_out"`
	PriceSet bool    `json:"price_set"`
	// HumanOnly models (subscription logins, later) refuse Background.
	HumanOnly bool `json:"human_only,omitempty"`
}

// Conservative estimate when no price is set: better to over-count.
const fallbackIn, fallbackOut = 15.0, 75.0

func (m Model) cost(u omnillm.Usage) int64 {
	in, out := m.PriceIn, m.PriceOut
	if !m.PriceSet {
		in, out = fallbackIn, fallbackOut
	}
	// USD per million tokens == micro-USD per token.
	return int64(float64(u.InputTokens)*in + float64(u.OutputTokens)*out)
}

func validModelID(id string) bool {
	if id == "" || len(id) > omnillm.MaxModelIDLength || !strings.Contains(id, "/") {
		return false
	}
	return !strings.ContainsAny(id, " \t\r\n")
}

func (h *Hub) putModel(ctx context.Context, m Model) error {
	if !validModelID(m.ID) {
		return &omnillm.Error{Code: omnillm.CodeBadRequest, Msg: "bad_model"}
	}
	return store.PutJSON(ctx, h.opts.Store, "model/"+m.ID, m)
}

func (h *Hub) getModel(ctx context.Context, id string) (Model, error) {
	var m Model
	if !validModelID(id) {
		return m, &omnillm.Error{Code: omnillm.CodeModelNotAllowed, Msg: omnillm.MsgUnknownModel}
	}
	if err := store.GetJSON(ctx, h.opts.Store, "model/"+id, &m); err != nil {
		return m, &omnillm.Error{Code: omnillm.CodeModelNotAllowed, Msg: omnillm.MsgUnknownModel}
	}
	return m, nil
}

func (h *Hub) listModels(ctx context.Context) ([]Model, error) {
	keys, err := h.opts.Store.List(ctx, "model/")
	if err != nil {
		return nil, err
	}
	var out []Model
	for _, k := range keys {
		var m Model
		if store.GetJSON(ctx, h.opts.Store, k, &m) == nil {
			out = append(out, m)
		}
	}
	return out, nil
}

// ModelUpdate changes only the fields that are set (nil leaves a field as
// it is), so updating a price can never clear HumanOnly by accident.
type ModelUpdate struct {
	DisplayName *string
	PriceIn     *float64 // USD per million input tokens
	PriceOut    *float64 // USD per million output tokens
	HumanOnly   *bool
}

// SetModel changes a model's display name, prices or human-only flag.
func (h *Hub) SetModel(ctx context.Context, id string, u ModelUpdate) error {
	m, err := h.getModel(ctx, id)
	if err != nil {
		return err
	}
	if u.DisplayName != nil {
		m.DisplayName = *u.DisplayName
	}
	if (u.PriceIn == nil) != (u.PriceOut == nil) {
		return &omnillm.Error{Code: omnillm.CodeBadRequest, Msg: "bad_model", Detail: "set both prices"}
	}
	if u.PriceIn != nil {
		if *u.PriceIn < 0 || *u.PriceOut < 0 {
			return &omnillm.Error{Code: omnillm.CodeBadRequest, Msg: "bad_model"}
		}
		m.PriceIn, m.PriceOut, m.PriceSet = *u.PriceIn, *u.PriceOut, true
	}
	if u.HumanOnly != nil {
		if !*u.HumanOnly && h.subscription(m.Provider) {
			// A subscription login is for people only (design §5.2); the
			// owner cannot turn that off.
			return &omnillm.Error{Code: omnillm.CodeBadRequest, Msg: omnillm.MsgHumanOnly}
		}
		m.HumanOnly = *u.HumanOnly
	}
	return h.putModel(ctx, m)
}

// RemoveModel hides a model.
func (h *Hub) RemoveModel(ctx context.Context, id string) error {
	return h.opts.Store.Delete(ctx, "model/"+id)
}

func (h *Hub) info(m Model) omnillm.ModelInfo {
	loc := omnillm.Local
	h.mu.Lock()
	c := h.clients[m.Provider]
	resolved := h.resolved[m.ID]
	h.mu.Unlock()
	kind := provider.Kind(m.Provider)
	if c != nil {
		kind = c.Config().Kind
	}
	if c == nil || !c.Config().Local() {
		loc = omnillm.Remote
	}
	mi := omnillm.ModelInfo{ID: m.ID, DisplayName: m.DisplayName, Provider: m.Provider, Locality: loc,
		HumanOnly: m.HumanOnly, Kind: string(kind)}
	// No words in a language here: each screen adds the suffix or the
	// Codex name in its own language from Latest and Kind.
	switch kind {
	case provider.ClaudeCode:
		// An alias follows the newest model; until a call (or the
		// weekly probe) reports which one, it is the family alone.
		mi.DisplayName, mi.Latest = subscriptionModelName(provider.ClaudeCode, m.Upstream), true
	case provider.Codex:
		// Codex's replies do not name their model, and the hub runs it
		// with --ignore-user-config and no -m: the model in the owner's
		// ~/.codex/config.toml is not the one that answers. So no
		// Version, only Codex's own newest default.
		mi.DisplayName, mi.Latest = subscriptionModelName(provider.Codex, m.Upstream), true
	}
	// The version the provider last said it answered with ("Claude Opus"
	// behind the alias "opus" is "Claude Opus 5.5" today).
	if resolved != "" {
		mi.DisplayName, mi.Latest = resolved, false
	}
	return mi
}

// noteModel remembers the exact model a provider answered with for one of
// our model ids, named for people with its version, and keeps it so the
// model list shows the version after a restart too.
func (h *Hub) noteModel(id, exact string) {
	name := provider.PrettyModel(exact)
	now := h.opts.Now()
	h.mu.Lock()
	same := h.resolved[id] == name
	if h.resolved == nil {
		h.resolved = map[string]string{}
	}
	if h.seen == nil {
		h.seen = map[string]time.Time{}
	}
	h.resolved[id] = name
	// When it was last confirmed decides whether the weekly probe asks;
	// kept at most once a day, not on every call.
	keepSeen := !same || now.Sub(h.seen[id]) > 24*time.Hour
	h.seen[id] = now
	h.mu.Unlock()
	if !same {
		_ = store.PutJSON(context.Background(), h.opts.Store, "resolved/"+id, name)
	}
	if keepSeen {
		_ = store.PutJSON(context.Background(), h.opts.Store, "seen/"+id, now)
	}
}

func (h *Hub) loadResolved(ctx context.Context) {
	keys, _ := h.opts.Store.List(ctx, "resolved/")
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.resolved == nil {
		h.resolved = map[string]string{}
	}
	for _, k := range keys {
		var n string
		if store.GetJSON(ctx, h.opts.Store, k, &n) == nil {
			h.resolved[strings.TrimPrefix(k, "resolved/")] = n
		}
	}
	if h.seen == nil {
		h.seen = map[string]time.Time{}
	}
	keys, _ = h.opts.Store.List(ctx, "seen/")
	for _, k := range keys {
		var t time.Time
		if store.GetJSON(ctx, h.opts.Store, k, &t) == nil {
			h.seen[strings.TrimPrefix(k, "seen/")] = t
		}
	}
}

// Models lists every model, with where its data goes as seen from this
// machine (the host's own view).
func (h *Hub) Models(ctx context.Context) ([]omnillm.ModelInfo, error) {
	ms, err := h.listModels(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]omnillm.ModelInfo, 0, len(ms))
	for _, m := range ms {
		out = append(out, h.info(m))
	}
	return out, nil
}

// DetectOllama reports the models of a local Ollama on this machine, if one
// is running (first-run wizard). It sends nothing off the machine.
func (h *Hub) DetectOllama(ctx context.Context) ([]string, error) {
	cfg, _ := provider.Default(provider.Ollama)
	c, err := provider.New(cfg, h.opts.Transport, h.lim)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return c.ListModels(ctx, provider.Credential{})
}

// RelayConfigured reports whether a relay address was given at all.
func (h *Hub) RelayConfigured() bool { return h.opts.RelayURL != "" }

// ChatModels drops models that cannot chat (embedding, rerank) from a
// local model list, by their well-known names.
func ChatModels(ids []string) []string {
	var out []string
	for _, id := range ids {
		l := strings.ToLower(id)
		chat := true
		for _, w := range []string{"embed", "bge-", "bge:", "rerank", "minilm", "e5-", "gte-", "colbert", "clip"} {
			if strings.Contains(l, w) {
				chat = false
			}
		}
		if chat {
			out = append(out, id)
		}
	}
	return out
}
