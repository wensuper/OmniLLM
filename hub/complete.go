package hub

import (
	"context"
	"errors"
	"io"
	"slices"
	"strings"
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
	// paused: the owner paused this device or app; the message saying so.
	paused string
}

// msg picks the sentence for whoever made the call: an app's own, or the
// device one (also for this Mac itself, and what older devices know).
func (p principal) msg(device, app string) string {
	if strings.HasPrefix(p.key, "app/") {
		return app
	}
	return device
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
	start := time.Now()
	// Paused by the owner: refused before anything else, so nothing is
	// sent and nothing is charged.
	if p.paused != "" {
		return nil, &omnillm.Error{Code: omnillm.CodeDevicePaused, Msg: p.paused}
	}
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
		return nil, &omnillm.Error{Code: omnillm.CodeModelNotAllowed, Msg: p.msg(omnillm.MsgModelNotAllowed, omnillm.MsgModelNotAllowedApp)}
	}
	// A model the owner took off the devices' list is gone for them, as if
	// removed: the device reloads its list and picks another.
	if strings.HasPrefix(p.key, "device/") && h.modelPrefs(ctx)[m.ID].Hidden {
		return nil, &omnillm.Error{Code: omnillm.CodeModelNotAllowed, Msg: omnillm.MsgUnknownModel}
	}
	if (m.HumanOnly || h.subscription(m.Provider)) && req.Caller != omnillm.Interactive {
		return nil, &omnillm.Error{Code: omnillm.CodeModelNotAllowed, Msg: omnillm.MsgHumanOnly}
	}
	if h.subscription(m.Provider) {
		// A device says itself that a person is waiting; how often it
		// asks is what the hub can check. A pace no person keeps up is
		// stopped before the account looks automated.
		if err := h.subs.allow(p.key, h.opts.Now()); err != nil {
			return nil, err
		}
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
	s, err := c.StreamChat(ctx, m.Upstream, req, cred, h.chats, p.key)
	cred.Key = ""
	if err != nil {
		h.unreserve(p, worst)
		h.audit(ctx, p, m, req.Caller, omnillm.Usage{}, string(omnillm.AsError(err).Code))
		return nil, err
	}
	h.inFlight.Add(1)
	return &accounted{Stream: s, h: h, p: p, m: m, caller: req.Caller, reserved: worst, inEst: inEst, start: start}, nil
}

// Busy reports a reply still on its way or a device being paired: what a
// restart (an update) would cut off.
func (h *Hub) Busy() bool {
	h.mu.Lock()
	pairing := len(h.pairings) > 0
	h.mu.Unlock()
	return pairing || h.inFlight.Load() > 0
}

// DefaultMaxTokens caps a reply when the caller sets no limit.
const DefaultMaxTokens = 4096

// estimateInput over-counts tokens: about 2 bytes a token, which is
// generous for English and still covers CJK text (3 bytes a character,
// often more than one token per character).
func estimateInput(req omnillm.Request) int {
	n := 0
	for _, m := range req.Messages {
		n += len(m.Content)/2 + 8
		n += 1600 * len(m.Images) // a large image is about 1,600 tokens
	}
	return n + len(req.JSONSchema)/2
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
	start    time.Time    // when the call was made
	first    atomic.Int64 // unix nanoseconds of the first text, 0 before
}

func (a *accounted) Recv() (omnillm.Chunk, error) {
	c, err := a.Stream.Recv()
	if c.Model != "" {
		a.h.noteModel(a.m.ID, c.Model)
	}
	a.outBytes.Add(int64(len(c.Text)))
	if c.Text != "" {
		a.first.CompareAndSwap(0, time.Now().UnixNano())
	}
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
		a.h.inFlight.Add(-1)
		u := a.Stream.Usage()
		if u.InputTokens == 0 {
			u.InputTokens = a.inEst
		}
		// No reported output (a cancelled stream, or a server that does not
		// report usage) is charged from what was received.
		if est := int(a.outBytes.Load()/2) + 1; u.OutputTokens == 0 || (outcome != "ok" && u.OutputTokens < est) {
			u.OutputTokens = est
		}
		u.CostMicroUSD = a.m.cost(u)
		a.mu.Lock()
		a.usage = u
		a.mu.Unlock()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		a.h.settleCharge(ctx, a.p, a.m.ID, a.reserved, u)
		a.h.audit(ctx, a.p, a.m, a.caller, u, outcome)
		// Timing only, never content: how long people wait is what to watch.
		ttft := "-"
		if f := a.first.Load(); f != 0 {
			ttft = time.Duration(f - a.start.UnixNano()).Round(100 * time.Millisecond).String()
		}
		a.h.opts.Logf("hub: call %s %s first text %s total %s (%s)", a.m.Provider, a.m.Upstream, ttft,
			time.Since(a.start).Round(100*time.Millisecond), outcome)
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

// modelUsageKey: the same day's use per model (model ids hold a "/" too,
// so they are kept apart from the per-device lines).
func modelUsageKey(day, model string) string { return "usagemodel/" + day + "/" + model }

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
	// An unreadable usage record must not read as "nothing spent".
	if err := getSpend(ctx, h.opts.Store, usageKey(day, p.key), &mine); err != nil {
		return err
	}
	if err := getSpend(ctx, h.opts.Store, usageKey(day, "_total"), &total); err != nil {
		return err
	}
	if h.reserved == nil {
		h.reserved = map[string]int64{}
	}
	if h.opts.GlobalBudget != NoLimit && total.MicroUSD+h.reserved["_total"]+worst > h.opts.GlobalBudget {
		return h.overBudget(total.MicroUSD+h.reserved["_total"], h.opts.GlobalBudget, omnillm.MsgBudgetGlobal)
	}
	// NoLimit: only the hub's daily total applies to this device.
	if p.budget != NoLimit && mine.MicroUSD+h.reserved[p.key]+worst > p.budget {
		return h.overBudget(mine.MicroUSD+h.reserved[p.key], p.budget, p.msg(omnillm.MsgBudgetDevice, omnillm.MsgBudgetApp))
	}
	h.reserved[p.key] += worst
	h.reserved["_total"] += worst
	return nil
}

// overBudget says why a call does not fit: the limit is used up (msg), or
// something is left but this call's worst case would pass it, which a
// shorter chat may still fit.
func (h *Hub) overBudget(used, limit int64, msg string) error {
	if used < limit {
		msg = omnillm.MsgBudgetWouldExceed
	}
	return &omnillm.Error{Code: omnillm.CodeBudgetExceeded, Msg: msg, RetryAfter: h.untilTomorrow()}
}

func (h *Hub) unreserve(p principal, worst int64) {
	h.accountMu.Lock()
	defer h.accountMu.Unlock()
	h.reserved[p.key] -= worst
	h.reserved["_total"] -= worst
}

func getSpend(ctx context.Context, st store.Store, k string, s *spend) error {
	err := store.GetJSON(ctx, st, k, s)
	if err == nil || errors.Is(err, store.ErrNotFound) {
		return nil
	}
	return &omnillm.Error{Code: omnillm.CodeInternal, Msg: "internal", Detail: "usage record unreadable"}
}

// settleCharge releases a call's reservation and records what it really
// cost in one step, so no concurrent reserve sees neither.
func (h *Hub) settleCharge(ctx context.Context, p principal, model string, worst int64, u omnillm.Usage) {
	h.accountMu.Lock()
	defer h.accountMu.Unlock()
	h.reserved[p.key] -= worst
	h.reserved["_total"] -= worst
	h.chargeLocked(ctx, p, model, u)
}

// chargeLocked records a call for whoever made it, for the day's total and
// for the model it used.
func (h *Hub) chargeLocked(ctx context.Context, p principal, model string, u omnillm.Usage) {
	day := h.day()
	for _, k := range []string{usageKey(day, p.key), usageKey(day, "_total"), modelUsageKey(day, model)} {
		var s spend
		if err := getSpend(ctx, h.opts.Store, k, &s); err != nil {
			// Never overwrite a record we could not read.
			h.opts.Logf("hub: usage record unreadable; this call is not recorded")
			continue
		}
		s.MicroUSD += u.CostMicroUSD
		s.InputTokens += int64(u.InputTokens)
		s.OutputTokens += int64(u.OutputTokens)
		s.Calls++
		if err := store.PutJSON(ctx, h.opts.Store, k, s); err != nil {
			h.opts.Logf("hub: recording usage failed: %v", err)
		}
	}
}

// Spend is one row of today's usage.
type Spend struct {
	Who string `json:"who"` // host, device/<id>, app/<id>, _total
	// Name is the device's or app's name for people (empty for host and
	// _total); Removed: that device or app is gone now.
	Name         string `json:"name,omitempty"`
	Removed      bool   `json:"removed,omitempty"`
	MicroUSD     int64  `json:"micro_usd"`
	InputTokens  int64  `json:"input_tokens"`
	OutputTokens int64  `json:"output_tokens"`
	Calls        int64  `json:"calls"`
}

// Usage returns usage for day ("2006-01-02"; empty means today).
func (h *Hub) Usage(ctx context.Context, day string) ([]Spend, error) {
	return h.UsageBetween(ctx, day, day)
}

// Day is the date back days before today, on this Mac's clock, as Usage
// takes it (0: today, 1: yesterday).
func (h *Hub) Day(back int) string { return h.opts.Now().AddDate(0, 0, -back).Format("2006-01-02") }

// maxUsageDays bounds how many days one usage question adds up.
const maxUsageDays = 31

// days lists the dates from first to last, both included (empty: today).
func (h *Hub) days(first, last string) ([]string, error) {
	if first == "" {
		first = h.day()
	}
	if last == "" {
		last = h.day()
	}
	a, err1 := time.Parse("2006-01-02", first)
	b, err2 := time.Parse("2006-01-02", last)
	if err1 != nil || err2 != nil || b.Before(a) {
		return nil, &omnillm.Error{Code: omnillm.CodeBadRequest, Msg: "bad_messages", Detail: "bad day range"}
	}
	var out []string
	for d := a; !d.After(b) && len(out) < maxUsageDays; d = d.AddDate(0, 0, 1) {
		out = append(out, d.Format("2006-01-02"))
	}
	return out, nil
}

// sumUsage adds up the records under prefix+day+"/" for every day.
func (h *Hub) sumUsage(ctx context.Context, prefix string, days []string) ([]Spend, error) {
	byWho := map[string]*Spend{}
	var order []string
	for _, day := range days {
		p := prefix + day + "/"
		keys, err := h.opts.Store.List(ctx, p)
		if err != nil {
			return nil, err
		}
		for _, k := range keys {
			var s spend
			if store.GetJSON(ctx, h.opts.Store, k, &s) != nil {
				continue
			}
			who := k[len(p):]
			o := byWho[who]
			if o == nil {
				o = &Spend{Who: who}
				byWho[who] = o
				order = append(order, who)
			}
			o.MicroUSD += s.MicroUSD
			o.InputTokens += s.InputTokens
			o.OutputTokens += s.OutputTokens
			o.Calls += s.Calls
		}
	}
	out := make([]Spend, 0, len(order))
	for _, who := range order {
		out = append(out, *byWho[who])
	}
	return out, nil
}

// UsageBetween adds up usage per device or app from day first to day last
// (both included; empty means today), named as Usage names them.
func (h *Hub) UsageBetween(ctx context.Context, first, last string) ([]Spend, error) {
	days, err := h.days(first, last)
	if err != nil {
		return nil, err
	}
	out, err := h.sumUsage(ctx, "usage/", days)
	if err != nil {
		return nil, err
	}
	h.nameSpends(ctx, out)
	return out, nil
}

// ModelUsageBetween adds up usage per model over the same days. Who is the
// model id and Name its name for people; Removed: the model is gone now.
func (h *Hub) ModelUsageBetween(ctx context.Context, first, last string) ([]Spend, error) {
	days, err := h.days(first, last)
	if err != nil {
		return nil, err
	}
	out, err := h.sumUsage(ctx, "usagemodel/", days)
	if err != nil {
		return nil, err
	}
	names := map[string]string{}
	ms, _ := h.OwnerModels(ctx)
	for _, m := range ms {
		names[m.ID] = m.DisplayName
	}
	for i := range out {
		if n, ok := names[out[i].Who]; ok {
			out[i].Name = n
			continue
		}
		out[i].Removed = true
	}
	return out, nil
}

// nameSpends fills in who each line is, for people: a device or app by its
// name, one removed since by the name it had.
func (h *Hub) nameSpends(ctx context.Context, out []Spend) {
	names := map[string]string{}
	devs, _ := h.listDevices(ctx)
	for _, d := range devs {
		names["device/"+d.ID] = d.Name
	}
	apps, _ := h.Apps(ctx)
	for _, a := range apps {
		names["app/"+a.ID] = a.Name
	}
	gone := map[string]string{}
	for _, t := range h.tombstones(ctx) {
		gone["device/"+t.Device.ID] = t.Device.Name
	}
	for i := range out {
		w := out[i].Who
		if n, ok := names[w]; ok {
			out[i].Name = n
			continue
		}
		if strings.HasPrefix(w, "device/") || strings.HasPrefix(w, "app/") {
			out[i].Removed = true
			out[i].Name = gone[w]
		}
	}
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
