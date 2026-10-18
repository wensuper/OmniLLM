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
	"sort"
	"strings"
	"sync"
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
	lim  *egress.Limiter
	key  e2e.KeyPair

	mu          sync.Mutex
	clients     map[string]*provider.Client
	link        *relaylink.HubLink
	sessions    map[relayproto.ID]*mailboxConn
	pairings    map[relayproto.ID]*pairing
	devSessions []sessionEntry
	deviceMu    sync.Mutex       // serializes device record writes
	reserved    map[string]int64 // worst-case cost of calls in flight (accountMu)
	accountMu   sync.Mutex
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
	if opts.DefaultBudget <= 0 {
		opts.DefaultBudget = 5 * dollar
	}
	if opts.GlobalBudget <= 0 {
		opts.GlobalBudget = 20 * dollar
	}
	if opts.HostBudget <= 0 {
		opts.HostBudget = 5 * dollar
	}
	h := &Hub{opts: opts, lim: egress.New(opts.Limits), clients: map[string]*provider.Client{},
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
	}
	return nil
}

// AddProvider connects a provider. For providers that need a key, the key is
// checked with a minimal call (listing models) before anything is saved, and
// the models it can use are added. The key is stored only in Secrets.
func (h *Hub) AddProvider(ctx context.Context, cfg provider.Config, apiKey string) ([]omnillm.ModelInfo, error) {
	apiKey = strings.TrimSpace(apiKey)
	if err := cfg.Validate(); err != nil {
		return nil, &omnillm.Error{Code: omnillm.CodeBadRequest, Msg: "bad_model", Detail: err.Error()}
	}
	if cfg.NeedsKey && (len(apiKey) < 8 || len(apiKey) > 512 || strings.ContainsAny(apiKey, " \t\r\n")) {
		return nil, &omnillm.Error{Code: omnillm.CodeCredentialProblem, Msg: omnillm.MsgCredentialBad, Args: []string{cfg.Name, redact.Last4(apiKey)}}
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
	h.mu.Lock()
	h.clients[cfg.Name] = c
	h.mu.Unlock()
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
		if err := h.putModel(ctx, m); err != nil {
			return nil, err
		}
		added = append(added, h.info(m))
	}
	h.opts.Logf("hub: provider %s connected with %d models", cfg.Name, len(added))
	return added, nil
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
	return nil
}

// Providers lists connected providers without their keys.
func (h *Hub) Providers(ctx context.Context) ([]ProviderInfo, error) {
	h.mu.Lock()
	var cfgs []provider.Config
	for _, c := range h.clients {
		cfgs = append(cfgs, c.Config())
	}
	h.mu.Unlock()
	sort.Slice(cfgs, func(i, j int) bool { return cfgs[i].Name < cfgs[j].Name })
	var out []ProviderInfo
	for _, c := range cfgs {
		pi := ProviderInfo{Name: c.Name, Kind: c.Kind, BaseURL: c.BaseURL, Local: c.Local()}
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
		m.HumanOnly = *u.HumanOnly
	}
	return h.putModel(ctx, m)
}

// RemoveModel hides a model.
func (h *Hub) RemoveModel(ctx context.Context, id string) error {
	return h.opts.Store.Delete(ctx, "model/"+id)
}

func (h *Hub) info(m Model) omnillm.ModelInfo {
	loc := omnillm.Remote
	h.mu.Lock()
	c := h.clients[m.Provider]
	h.mu.Unlock()
	if c != nil && c.Config().Local() {
		loc = omnillm.Local
	}
	return omnillm.ModelInfo{ID: m.ID, DisplayName: m.DisplayName, Provider: m.Provider, Locality: loc, HumanOnly: m.HumanOnly}
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
