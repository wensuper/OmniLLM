package hub

import (
	"context"
	"errors"
	"io"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	omnillm "github.com/wensuper/OmniLLM"
	"github.com/wensuper/OmniLLM/internal/provider"
	"github.com/wensuper/OmniLLM/store"
)

// principal is who a call is charged to and what it may use.
type principal struct {
	key    string // "host", "device/<id>", "app/<id>"
	models []string
	budget int64
	// forceBackground: app tokens are always Background (principle 1).
	forceBackground bool
}

func (h *Hub) hostPrincipal() principal {
	return principal{key: "host", budget: h.opts.HostBudget}
}

// Complete is the in-process call for the host (design §2.4). The request's
// Caller says whether a person is waiting; NEXA's judging passes Background.
func (h *Hub) Complete(ctx context.Context, req omnillm.Request) (omnillm.Stream, error) {
	return h.complete(ctx, h.hostPrincipal(), req)
}

func (h *Hub) complete(ctx context.Context, p principal, req omnillm.Request) (omnillm.Stream, error) {
	if p.forceBackground {
		req.Caller = omnillm.Background
	}
	// Every call has a reply cap, so its worst-case cost is known.
	if req.MaxTokens == 0 {
		req.MaxTokens = DefaultMaxTokens
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	m, err := h.getModel(ctx, req.Model)
	if err != nil {
		return nil, err
	}
	if len(p.models) > 0 && !slices.Contains(p.models, m.ID) {
		return nil, &omnillm.Error{Code: omnillm.CodeModelNotAllowed, Msg: omnillm.MsgModelNotAllowed}
	}
	if m.HumanOnly && req.Caller != omnillm.Interactive {
		return nil, &omnillm.Error{Code: omnillm.CodeModelNotAllowed, Msg: omnillm.MsgHumanOnly}
	}
	h.mu.Lock()
	c := h.clients[m.Provider]
	h.mu.Unlock()
	if c == nil {
		return nil, &omnillm.Error{Code: omnillm.CodeCredentialProblem, Msg: omnillm.MsgCredentialMissing, Args: []string{m.Provider}}
	}
	if !c.Config().Local() {
		if err := h.checkRegion(ctx, m.Provider); err != nil {
			return nil, err
		}
	}
	inEst := estimateInput(req)
	worst := m.cost(omnillm.Usage{InputTokens: inEst, OutputTokens: req.MaxTokens})
	if err := h.reserve(ctx, p, worst); err != nil {
		return nil, err
	}
	var cred provider.Credential
	if c.Config().NeedsKey {
		k, err := h.opts.Secrets.Get(ctx, providerKey(m.Provider))
		if err != nil {
			h.unreserve(p, worst)
			return nil, &omnillm.Error{Code: omnillm.CodeCredentialProblem, Msg: omnillm.MsgCredentialMissing, Args: []string{m.Provider}}
		}
		cred.Key = string(k)
	}
	s, err := c.Stream(ctx, m.Upstream, req, cred)
	cred.Key = ""
	if err != nil {
		h.unreserve(p, worst)
		h.audit(ctx, p, m, req.Caller, omnillm.Usage{}, string(omnillm.AsError(err).Code))
		return nil, err
	}
	return &accounted{Stream: s, h: h, p: p, m: m, caller: req.Caller, reserved: worst, inEst: inEst}, nil
}

// DefaultMaxTokens caps a reply when the caller sets no limit.
const DefaultMaxTokens = 4096

// estimateInput over-counts tokens from bytes (about 3 bytes a token).
func estimateInput(req omnillm.Request) int {
	n := 0
	for _, m := range req.Messages {
		n += len(m.Content)/3 + 8
	}
	return n + len(req.JSONSchema)/3
}

// accounted charges usage when the stream ends, however it ends. When the
// provider reported no usage (a cancelled stream), it charges an estimate:
// the provider still bills for what it received.
type accounted struct {
	omnillm.Stream
	h        *Hub
	p        principal
	m        Model
	caller   omnillm.Caller
	reserved int64
	inEst    int
	outBytes atomic.Int64
	once     sync.Once
	mu       sync.Mutex
	usage    omnillm.Usage
}

func (a *accounted) Recv() (omnillm.Chunk, error) {
	c, err := a.Stream.Recv()
	a.outBytes.Add(int64(len(c.Text)))
	if err != nil {
		outcome := "ok"
		if !errors.Is(err, io.EOF) {
			outcome = string(omnillm.AsError(err).Code)
		}
		a.settle(outcome)
	}
	return c, err
}

func (a *accounted) Usage() omnillm.Usage {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.usage
}

func (a *accounted) Close() error {
	err := a.Stream.Close()
	a.settle("closed")
	return err
}

func (a *accounted) settle(outcome string) {
	a.once.Do(func() {
		u := a.Stream.Usage()
		if u.InputTokens == 0 {
			u.InputTokens = a.inEst
		}
		if est := int(a.outBytes.Load()/3) + 1; u.OutputTokens < est && outcome != "ok" {
			u.OutputTokens = est
		}
		u.CostMicroUSD = a.m.cost(u)
		a.mu.Lock()
		a.usage = u
		a.mu.Unlock()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		a.h.unreserve(a.p, a.reserved)
		a.h.charge(ctx, a.p, u)
		a.h.audit(ctx, a.p, a.m, a.caller, u, outcome)
	})
}

// ---------- budgets ----------

type spend struct {
	MicroUSD     int64 `json:"micro_usd"`
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
	Calls        int64 `json:"calls"`
}

func (h *Hub) day() string { return h.opts.Now().Format("2006-01-02") }

func usageKey(day, who string) string { return "usage/" + day + "/" + who }

func (h *Hub) untilTomorrow() time.Duration {
	now := h.opts.Now()
	y, mo, d := now.Date()
	return time.Date(y, mo, d+1, 0, 0, 0, 0, now.Location()).Sub(now)
}

// reserve sets aside a call's worst-case cost before it is sent. Spent plus
// everything reserved may never pass the principal's or the hub's budget,
// so concurrent calls cannot overshoot it.
func (h *Hub) reserve(ctx context.Context, p principal, worst int64) error {
	h.accountMu.Lock()
	defer h.accountMu.Unlock()
	day := h.day()
	var mine, total spend
	_ = store.GetJSON(ctx, h.opts.Store, usageKey(day, p.key), &mine)
	_ = store.GetJSON(ctx, h.opts.Store, usageKey(day, "_total"), &total)
	if h.reserved == nil {
		h.reserved = map[string]int64{}
	}
	if total.MicroUSD+h.reserved["_total"]+worst > h.opts.GlobalBudget {
		return &omnillm.Error{Code: omnillm.CodeBudgetExceeded, Msg: omnillm.MsgBudgetGlobal, RetryAfter: h.untilTomorrow()}
	}
	if mine.MicroUSD+h.reserved[p.key]+worst > p.budget {
		return &omnillm.Error{Code: omnillm.CodeBudgetExceeded, Msg: omnillm.MsgBudgetDevice, RetryAfter: h.untilTomorrow()}
	}
	h.reserved[p.key] += worst
	h.reserved["_total"] += worst
	return nil
}

func (h *Hub) unreserve(p principal, worst int64) {
	h.accountMu.Lock()
	defer h.accountMu.Unlock()
	h.reserved[p.key] -= worst
	h.reserved["_total"] -= worst
}

func (h *Hub) charge(ctx context.Context, p principal, u omnillm.Usage) {
	h.accountMu.Lock()
	defer h.accountMu.Unlock()
	day := h.day()
	for _, who := range []string{p.key, "_total"} {
		var s spend
		_ = store.GetJSON(ctx, h.opts.Store, usageKey(day, who), &s)
		s.MicroUSD += u.CostMicroUSD
		s.InputTokens += int64(u.InputTokens)
		s.OutputTokens += int64(u.OutputTokens)
		s.Calls++
		if err := store.PutJSON(ctx, h.opts.Store, usageKey(day, who), s); err != nil {
			h.opts.Logf("hub: recording usage failed: %v", err)
		}
	}
}

// Spend is one row of today's usage.
type Spend struct {
	Who          string `json:"who"` // host, device/<id>, app/<id>, _total
	MicroUSD     int64  `json:"micro_usd"`
	InputTokens  int64  `json:"input_tokens"`
	OutputTokens int64  `json:"output_tokens"`
	Calls        int64  `json:"calls"`
}

// Usage returns usage for day ("2006-01-02"; empty means today).
func (h *Hub) Usage(ctx context.Context, day string) ([]Spend, error) {
	if day == "" {
		day = h.day()
	}
	keys, err := h.opts.Store.List(ctx, "usage/"+day+"/")
	if err != nil {
		return nil, err
	}
	var out []Spend
	for _, k := range keys {
		var s spend
		if store.GetJSON(ctx, h.opts.Store, k, &s) == nil {
			out = append(out, Spend{Who: k[len("usage/"+day+"/"):], MicroUSD: s.MicroUSD,
				InputTokens: s.InputTokens, OutputTokens: s.OutputTokens, Calls: s.Calls})
		}
	}
	return out, nil
}

// ---------- audit: metadata only, never content ----------

// AuditEntry is one call. It holds no prompt or reply text.
type AuditEntry struct {
	Time     time.Time `json:"time"`
	Who      string    `json:"who"`
	Model    string    `json:"model"`
	Caller   string    `json:"caller"`
	In       int       `json:"in"`
	Out      int       `json:"out"`
	MicroUSD int64     `json:"micro_usd"`
	Outcome  string    `json:"outcome"`
}

const maxAuditPerDay = 2000

func (h *Hub) audit(ctx context.Context, p principal, m Model, caller omnillm.Caller, u omnillm.Usage, outcome string) {
	h.accountMu.Lock()
	defer h.accountMu.Unlock()
	k := "audit/" + h.day()
	var entries []AuditEntry
	_ = store.GetJSON(ctx, h.opts.Store, k, &entries)
	entries = append(entries, AuditEntry{Time: h.opts.Now().UTC(), Who: p.key, Model: m.ID, Caller: caller.String(),
		In: u.InputTokens, Out: u.OutputTokens, MicroUSD: u.CostMicroUSD, Outcome: outcome})
	if len(entries) > maxAuditPerDay {
		entries = entries[len(entries)-maxAuditPerDay:]
	}
	if err := store.PutJSON(ctx, h.opts.Store, k, entries); err != nil {
		h.opts.Logf("hub: audit write failed: %v", err)
	}
}

// Audit returns the entries for day (empty: today).
func (h *Hub) Audit(ctx context.Context, day string) ([]AuditEntry, error) {
	if day == "" {
		day = h.day()
	}
	var entries []AuditEntry
	err := store.GetJSON(ctx, h.opts.Store, "audit/"+day, &entries)
	if errors.Is(err, store.ErrNotFound) {
		return nil, nil
	}
	return entries, err
}

// ---------- egress region (principle 1) ----------

type regionState struct {
	Confirmed string `json:"confirmed"`
	Paused    bool   `json:"paused"`
	Seen      string `json:"seen"`
}

func (h *Hub) loadRegion(ctx context.Context) error {
	err := store.GetJSON(ctx, h.opts.Store, "region", &h.region)
	if errors.Is(err, store.ErrNotFound) {
		return nil
	}
	return err
}

func (h *Hub) checkRegion(ctx context.Context, providerName string) error {
	if h.opts.EgressRegion == nil {
		return nil
	}
	country, err := h.opts.EgressRegion(ctx)
	if err != nil || country == "" {
		// Unknown region: do not guess; the call is not sent.
		return &omnillm.Error{Code: omnillm.CodeRegionNotAllowed, Msg: omnillm.MsgRegionChanged}
	}
	h.accountMu.Lock()
	st := h.region
	changed := false
	if st.Confirmed == "" {
		st.Confirmed = country
		changed = true
	}
	if country != st.Confirmed && !st.Paused {
		st.Paused, st.Seen = true, country
		changed = true
	}
	if changed {
		h.region = st
		_ = store.PutJSON(ctx, h.opts.Store, "region", st)
	}
	h.accountMu.Unlock()
	if st.Paused {
		return &omnillm.Error{Code: omnillm.CodeRegionNotAllowed, Msg: omnillm.MsgRegionChanged}
	}
	if h.opts.RegionAllowed != nil && !h.opts.RegionAllowed(providerName, country) {
		return &omnillm.Error{Code: omnillm.CodeRegionNotAllowed, Msg: omnillm.MsgRegionUnsupported, Args: []string{providerName}}
	}
	return nil
}

// RegionStatus reports the confirmed region and whether calls are paused.
func (h *Hub) RegionStatus() (confirmed, seen string, paused bool) {
	h.accountMu.Lock()
	defer h.accountMu.Unlock()
	return h.region.Confirmed, h.region.Seen, h.region.Paused
}

// ConfirmRegion accepts the newly seen region and resumes upstream calls.
// Only the owner on the hub may call this (host UI or local control).
func (h *Hub) ConfirmRegion(ctx context.Context) error {
	h.accountMu.Lock()
	defer h.accountMu.Unlock()
	if h.region.Seen != "" {
		h.region.Confirmed = h.region.Seen
	}
	h.region.Paused, h.region.Seen = false, ""
	return store.PutJSON(ctx, h.opts.Store, "region", h.region)
}
