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

// autoKey names a conversation: who is talking, its first message and
// the start of its first reply (two chats that both began with "hi" got
// different replies). A new message adds to the end, so the key stays the
// same all along. reply: the first reply's start when the request has no
// reply yet (the turn that is answering it).
func autoKey(p principal, req omnillm.Request, reply string) [32]byte {
	user, seen := "", false
	for _, m := range req.Messages {
		if m.Role == "user" && !seen {
			user, seen = m.Content, true
		} else if m.Role == "assistant" && seen {
			reply = m.Content
			break
		}
	}
	if len(reply) > autoKeyReply {
		reply = reply[:autoKeyReply]
	}
	h := sha256.New()
	for _, s := range []string{p.key, user, reply} {
		h.Write([]byte(s))
		h.Write([]byte{0})
	}
	var k [32]byte
	copy(k[:], h.Sum(nil))
	return k
}

// autoKeyReply: how much of the first reply names a conversation.
const autoKeyReply = 400

// firstTurn: the request holds no reply yet.
func firstTurn(req omnillm.Request) bool {
	for _, m := range req.Messages {
		if m.Role == "assistant" {
			return false
		}
	}
	return true
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
		if mp.Name != "" {
			m.DisplayName, m.Renamed = mp.Name, true // the owner's name signs the reply
		}
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
	key := autoKey(p, req, "")
	var stick *autoStick
	if s, ok := h.auto.stuck(key, h.opts.Now()); ok && !firstTurn(req) {
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
		// Only people's models are there and a program asked: say that,
		// not "signed out" (picking one by hand would fail the same way).
		if req.Caller != omnillm.Interactive {
			r := req
			r.Caller = omnillm.Interactive
			if people, err := h.autoCandidates(ctx, p, r, tier, nil); err == nil && len(people) > 0 {
				return nil, &omnillm.Error{Code: omnillm.CodeModelNotAllowed, Msg: omnillm.MsgHumanOnly}
			}
		}
		return nil, &omnillm.Error{Code: omnillm.CodeModelNotAllowed, Msg: omnillm.MsgNoAutoModel}
	}
	a := &autoStream{h: h, p: p, req: req, cands: cands, tier: tier, key: key}
	if err := a.next(ctx); err != nil {
		return nil, err
	}
	return a, nil
}

// autoLabel is the chosen model's name for the reply ("Claude Sonnet 5.5",
// "GPT-6-Sol"); a later, more exact name from the model replaces it.
func autoLabel(m omnillm.ModelInfo) string {
	if m.Renamed {
		return m.DisplayName
	}
	if m.Kind == string(provider.Codex) {
		if m.Version != "" {
			return m.Version
		}
		return "ChatGPT (Codex)"
	}
	return m.DisplayName
}

// autoStream is Auto's reply. It says which model was chosen at once
// (a Chunk with only Model, before any words, so a screen can name the
// model that is thinking), and says it again if that model fails before
// its first word and the next one takes over. Once words come, it is
// that model's reply to the end.
type autoStream struct {
	h     *Hub
	p     principal
	req   omnillm.Request
	cands []autoCand
	tier  autoTier
	key   [32]byte

	i       int // the candidate answering
	tried   int
	s       omnillm.Stream
	told    bool            // its name was passed on
	started bool            // its first words came
	kept    bool            // the chat's model is remembered
	reply   strings.Builder // a first turn's reply start, which names the chat
	ctx     context.Context
}

// next starts the next candidate that takes the request; the last error
// when none does.
func (a *autoStream) next(ctx context.Context) error {
	a.ctx = ctx
	var last error
	for a.tried < len(a.cands) && a.tried < autoAttempts {
		c := a.cands[a.tried]
		a.tried++
		r := a.req
		r.Model = c.info.ID
		s, err := a.h.complete(ctx, a.p, r)
		if err != nil {
			last = err
			if !autoRetry(err) {
				return err
			}
			continue
		}
		a.i, a.s, a.told = a.tried-1, s, false
		a.h.opts.Logf("hub: auto %s → %s", a.tier, c.info.ID)
		return nil
	}
	if last == nil {
		last = &omnillm.Error{Code: omnillm.CodeModelNotAllowed, Msg: omnillm.MsgNoAutoModel}
	}
	return last
}

func (a *autoStream) Recv() (omnillm.Chunk, error) {
	for {
		if !a.told {
			a.told = true
			return omnillm.Chunk{Model: autoLabel(a.cands[a.i].info)}, nil
		}
		ch, err := a.s.Recv()
		// One name per model that answers: the one sent above. A second
		// name always means another model took over (the screens rely on
		// it), and a name the owner gave stays.
		ch.Model = ""
		if ch.Text == "" && err == nil {
			continue
		}
		if !a.started && err != nil && !errors.Is(err, io.EOF) && autoRetry(err) && a.tried < len(a.cands) && a.tried < autoAttempts {
			// Failed before its first word: the next one takes over.
			a.h.opts.Logf("hub: auto %s: %s passed over (%s)", a.tier, a.cands[a.i].info.ID, omnillm.AsError(err).Code)
			a.s.Close()
			if nerr := a.next(a.ctx); nerr != nil {
				return omnillm.Chunk{}, nerr
			}
			continue
		}
		if ch.Text != "" {
			a.started = true
		}
		a.remember(ch.Text, err)
		return ch, err
	}
}

// remember keeps which model answers this chat: on a later turn at its
// first words; on the first turn once the reply's start (part of the
// chat's name) is known or the reply has ended.
func (a *autoStream) remember(text string, err error) {
	if a.kept || (err != nil && !errors.Is(err, io.EOF)) {
		return
	}
	key := a.key
	if firstTurn(a.req) {
		if a.reply.Len() < autoKeyReply {
			a.reply.WriteString(text)
		}
		if a.reply.Len() < autoKeyReply && err == nil {
			return
		}
		key = autoKey(a.p, a.req, a.reply.String())
	} else if !a.started {
		return
	}
	a.kept = true
	c := a.cands[a.i].info
	a.h.auto.keep(key, autoStick{model: c.ID, tier: a.tier, at: a.h.opts.Now()}, c.Provider)
}

func (a *autoStream) Usage() omnillm.Usage { return a.s.Usage() }
func (a *autoStream) FinishReason() string { return a.s.FinishReason() }
func (a *autoStream) Close() error         { return a.s.Close() }

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
