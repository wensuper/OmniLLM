package hub

// Auto: the model id "auto" lets the hub choose the model for each
// message (design §13.6). The hub alone knows what can answer right now:
// which sources are signed in, at their limit, switched off or kept off a
// device, and how much each subscription was used in the last hour. The
// choice is made by rules, never by asking a model, so it costs nothing,
// adds no wait, and can be explained:
//
//  1. Candidates: models the caller may use now (allowed for the device,
//     its source on and signed in, not at its limit, not kept out of Auto
//     by the owner; a subscription only for a person typing).
//  2. Tier: the last message says how hard it is (quick, normal, hard,
//     image), from its length, code, error logs and a few words.
//  3. Order: the tier's preferred models first, subscriptions (already
//     paid for) before API keys before models on this Mac; among equals
//     the source used least in the last hour, so no one subscription is
//     used up first.
//  4. Stay: a conversation keeps the model that answered it last (its
//     session goes on, only the new message is sent) unless that model
//     can't answer now or the conversation got harder.
//  5. Fall back: a model that fails before its first word (limit, signed
//     out, busy, can't read the image) is passed over for the next one.
//     A reply that has begun is never switched.

import (
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	omnillm "github.com/wensuper/OmniLLM"
	"github.com/wensuper/OmniLLM/internal/provider"
	"github.com/wensuper/OmniLLM/store"
)

// AutoModel is the model id that asks the hub to choose.
const AutoModel = "auto"

type autoTier int

const (
	tierQuick autoTier = iota
	tierNormal
	tierHard
	tierImage
)

func (t autoTier) String() string {
	return [...]string{"quick", "normal", "hard", "image"}[t]
}

// tierPrefs: for each tier, groups of equally good models by name, best
// group first. A model in no group comes after all of them.
var tierPrefs = map[autoTier][][]*regexp.Regexp{
	tierQuick: {
		{re(`flash.*low|flash-low`), re(`haiku`), re(`\bmini\b|-mini`)},
		{re(`flash`), re(`luna`), re(`sonnet`)},
	},
	tierNormal: {
		{re(`sonnet`), re(`gpt-6-sol`)},
		{re(`flash.*high|flash-high`), re(`gpt-6|gpt-5\.6`)},
		{re(`gemini`), re(`gpt-5`), re(`claude`)},
	},
	tierHard: {
		{re(`opus`), re(`gpt-6-astra`)},
		{re(`sonnet`), re(`gpt-6`), re(`thinking`), re(`\bpro\b|-pro`)},
	},
	tierImage: {
		{re(`sonnet`), re(`gpt-6-sol`)},
		{re(`opus`), re(`gpt-6`), re(`claude`), re(`gpt`)},
	},
}

func re(s string) *regexp.Regexp { return regexp.MustCompile(`(?i)` + s) }

// hardWords mark a message that needs the strongest models.
var hardWords = re(`证明|推导|架构|重构|算法|调试|排查|报错|优化一下|详细分析|一步一步|逐步|设计一个|写一个.*(程序|脚本|函数|类)|` +
	`\bprove\b|\bderive\b|refactor|architect|algorithm|debug|stack ?trace|step[- ]by[- ]step|in depth|design an?\b`)

// errorLog: lines a pasted error log or stack trace has.
var errorLog = re(`traceback \(most recent call last\)|exception in thread|panic: |\berror(:|\[)|^\s+at [\w.$]+\(|goroutine \d+ \[`)

var quickWords = re(`^(翻译|译成|translate|你好|hi\b|hello|谢谢|thanks|早上好|晚安)`)

// autoTierOf says how hard the conversation's last message is.
func autoTierOf(req omnillm.Request) autoTier {
	var last omnillm.Message
	total := 0
	for _, m := range req.Messages {
		total += len(m.Content)
		if m.Role == "user" {
			last = m
		}
	}
	if len(last.Images) > 0 {
		return tierImage
	}
	text := strings.TrimSpace(last.Content)
	n := utf8.RuneCountInString(text)
	switch {
	case strings.Contains(text, "```") || errorLog.MatchString(text) || hardWords.MatchString(text) || n > 1500 || total > 40000:
		return tierHard
	case n <= 20 && !strings.ContainsAny(text, "\n") || quickWords.MatchString(text) && n <= 300:
		return tierQuick
	}
	return tierNormal
}

// autoCand is a model Auto may choose, with what orders it.
type autoCand struct {
	info  omnillm.ModelInfo
	group int // the tier's preference group (lower first)
	cost  int // 0 a subscription, 1 an API key, 2 this Mac
	used  int // calls to its source through Auto in the last hour
}

// autoState: which model answered each conversation last, and how much
// Auto used each source lately. In memory only: a restart forgets it.
type autoState struct {
	mu    sync.Mutex
	stick map[[32]byte]autoStick
	calls map[string][]time.Time // source → when
}

type autoStick struct {
	model string
	tier  autoTier
	at    time.Time
}

const (
	autoStickFor  = 24 * time.Hour
	autoStickMax  = 2000
	autoAttempts  = 3
	autoUsedSince = time.Hour
)

// autoKey names a conversation: who is talking and how it began. A new
// message adds to the end, so the key stays the same all along.
func autoKey(p principal, req omnillm.Request) [32]byte {
	h := sha256.New()
	h.Write([]byte(p.key))
	h.Write([]byte{0})
	for _, m := range req.Messages {
		if m.Role == "user" {
			h.Write([]byte(m.Content))
			break
		}
	}
	var k [32]byte
	copy(k[:], h.Sum(nil))
	return k
}

func (a *autoState) stuck(k [32]byte, now time.Time) (autoStick, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	s, ok := a.stick[k]
	if !ok || now.Sub(s.at) > autoStickFor {
		return autoStick{}, false
	}
	return s, true
}

func (a *autoState) keep(k [32]byte, s autoStick, source string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.stick == nil {
		a.stick, a.calls = map[[32]byte]autoStick{}, map[string][]time.Time{}
	}
	if len(a.stick) >= autoStickMax {
		// The oldest goes.
		var oldest [32]byte
		first := true
		for key, v := range a.stick {
			if first || v.at.Before(a.stick[oldest].at) {
				oldest, first = key, false
			}
		}
		delete(a.stick, oldest)
	}
	a.stick[k] = s
	a.calls[source] = append(a.recent(source, s.at), s.at)
}

// recent is the source's Auto calls in the last hour (a.mu held).
func (a *autoState) recent(source string, now time.Time) []time.Time {
	list := a.calls[source]
	i := 0
	for i < len(list) && now.Sub(list[i]) > autoUsedSince {
		i++
	}
	return list[i:]
}

func (a *autoState) usedLately(source string, now time.Time) int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.recent(source, now))
}

// autoNoVision: sources known not to take images.
func autoNoVision(kind string) bool { return kind == string(provider.Antigravity) }

// autoCandidates lists what may answer now, best first. stick is the
// model this conversation used last, tried first when it still fits.
func (h *Hub) autoCandidates(ctx context.Context, p principal, req omnillm.Request, tier autoTier, stick *autoStick) ([]autoCand, error) {
	all, err := h.Models(ctx)
	if err != nil {
		return nil, err
	}
	prefs := h.modelPrefs(ctx)
	off := h.sourcesOff(ctx)
	device := strings.HasPrefix(p.key, "device/")
	now := h.opts.Now()
	var out []autoCand
	for _, m := range all {
		mp := prefs[m.ID]
		switch {
		case len(p.models) > 0 && !contains(p.models, m.ID),
			off[m.Provider], mp.NoAuto,
			device && mp.Hidden,
			m.Problem != "",
			(m.HumanOnly || h.subscription(m.Provider)) && req.Caller != omnillm.Interactive,
			tier == tierImage && autoNoVision(m.Kind):
			continue
		}
		h.mu.Lock()
		c := h.clients[m.Provider]
		h.mu.Unlock()
		if c == nil {
			continue
		}
		cost := 1
		switch {
		case provider.Kind(m.Kind).Subscription():
			cost = 0
		case c.Config().Local():
			cost = 2
		}
		out = append(out, autoCand{info: m, group: autoGroup(tier, m), cost: cost, used: h.auto.usedLately(m.Provider, now)})
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.group != b.group {
			return a.group < b.group
		}
		if a.cost != b.cost {
			return a.cost < b.cost
		}
		if a.used != b.used {
			return a.used < b.used
		}
		if a.info.Rank != b.info.Rank {
			return a.info.Rank < b.info.Rank
		}
		return a.info.ID < b.info.ID
	})
	// The conversation's own model first, unless it got harder since.
	if stick != nil && tier <= stick.tier {
		for i, c := range out {
			if c.info.ID == stick.model {
				out = append([]autoCand{c}, append(out[:i:i], out[i+1:]...)...)
				break
			}
		}
	}
	return out, nil
}

// autoGroup is the model's preference group in the tier. Another
// maker's model offered through Antigravity (an older Claude or GPT)
// stands one group behind the maker's own.
func autoGroup(tier autoTier, m omnillm.ModelInfo) int {
	name := m.ID + " " + m.DisplayName + " " + m.Version
	for i, group := range tierPrefs[tier] {
		for _, r := range group {
			if r.MatchString(name) {
				if m.Kind == string(provider.Antigravity) && !strings.Contains(strings.ToLower(name), "gemini") {
					return i + 1
				}
				return i
			}
		}
	}
	return len(tierPrefs[tier])
}

// autoRetry: an error that another model may not have.
func autoRetry(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	e := omnillm.AsError(err)
	switch e.Code {
	case omnillm.CodeDevicePaused, omnillm.CodeBadRequest:
		// Paused: no model may answer. A bad request is bad for all,
		// except what one model can't take.
		return e.Msg == omnillm.MsgNoVision || e.Msg == omnillm.MsgContextTooLong
	}
	return true
}

// completeAuto answers with the model the rules choose.
func (h *Hub) completeAuto(ctx context.Context, p principal, req omnillm.Request) (omnillm.Stream, error) {
	if p.paused != "" {
		return nil, &omnillm.Error{Code: omnillm.CodeDevicePaused, Msg: p.paused}
	}
	if p.forceBackground {
		req.Caller = omnillm.Background
	}
	tier := autoTierOf(req)
	key := autoKey(p, req)
	var stick *autoStick
	if s, ok := h.auto.stuck(key, h.opts.Now()); ok {
		stick = &s
		if s.tier > tier && s.tier != tierImage {
			tier = s.tier // a conversation doesn't get easier mid-way
		}
	}
	cands, err := h.autoCandidates(ctx, p, req, tier, stick)
	if err != nil {
		return nil, err
	}
	if len(cands) == 0 {
		return nil, &omnillm.Error{Code: omnillm.CodeModelNotAllowed, Msg: omnillm.MsgNoAutoModel}
	}
	var last error
	for i, c := range cands {
		if i == autoAttempts {
			break
		}
		r := req
		r.Model = c.info.ID
		s, err := h.complete(ctx, p, r)
		if err != nil {
			last = err
			if !autoRetry(err) {
				return nil, err
			}
			continue
		}
		// Read up to the first words: a failure before them passes the
		// message on to the next model.
		var held []omnillm.Chunk
		var serr error
		for {
			ch, err := s.Recv()
			if err != nil {
				serr = err
				break
			}
			held = append(held, ch)
			if ch.Text != "" {
				break
			}
		}
		if serr != nil && !errors.Is(serr, io.EOF) && autoRetry(serr) && i+1 < len(cands) && i+1 < autoAttempts {
			s.Close()
			last = serr
			h.opts.Logf("hub: auto %s: %s passed over (%s)", tier, c.info.ID, omnillm.AsError(serr).Code)
			continue
		}
		h.auto.keep(key, autoStick{model: c.info.ID, tier: tier, at: h.opts.Now()}, c.info.Provider)
		h.opts.Logf("hub: auto %s → %s", tier, c.info.ID)
		return &autoStream{Stream: s, held: append([]omnillm.Chunk{{Model: autoLabel(c.info)}}, held...), err: serr}, nil
	}
	return nil, last
}

// autoLabel is the chosen model's name for the reply ("Claude Sonnet 5.5",
// "GPT-6-Sol"); a later, more exact name from the model replaces it.
func autoLabel(m omnillm.ModelInfo) string {
	if m.Kind == string(provider.Codex) {
		if m.Version != "" {
			return m.Version
		}
		return "ChatGPT (Codex)"
	}
	return m.DisplayName
}

// autoStream passes on what was read to choose, then the rest.
type autoStream struct {
	omnillm.Stream
	held []omnillm.Chunk
	err  error // what ended the stream while choosing, after held
}

func (a *autoStream) Recv() (omnillm.Chunk, error) {
	if len(a.held) > 0 {
		ch := a.held[0]
		a.held = a.held[1:]
		return ch, nil
	}
	if a.err != nil {
		return omnillm.Chunk{}, a.err
	}
	return a.Stream.Recv()
}

// SetModelAuto keeps a model in Auto's choices or out of them.
func (h *Hub) SetModelAuto(ctx context.Context, id string, on bool) error {
	if _, err := h.getModel(ctx, id); err != nil {
		return err
	}
	h.deviceMu.Lock()
	defer h.deviceMu.Unlock()
	var mp modelPref
	_ = store.GetJSON(ctx, h.opts.Store, "modelpref/"+id, &mp)
	mp.NoAuto = !on
	if mp == (modelPref{}) {
		return h.opts.Store.Delete(ctx, "modelpref/"+id)
	}
	return store.PutJSON(ctx, h.opts.Store, "modelpref/"+id, mp)
}

// autoInfo is Auto as a device lists it: first, when there is something
// to choose between. Each screen names it in its own language (Kind).
func autoInfo() omnillm.ModelInfo {
	return omnillm.ModelInfo{ID: AutoModel, DisplayName: "Auto", Provider: AutoModel, Kind: AutoModel, Locality: omnillm.Remote}
}
