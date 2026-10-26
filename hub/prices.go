package hub

import (
	"net/url"
	"regexp"
	"strings"
	"time"

	omnillm "github.com/wensuper/OmniLLM"
	"github.com/wensuper/OmniLLM/internal/provider"
)

// Where the price spend is counted at comes from (ModelView.PriceSource).
const (
	PriceSourceSet       = "set"       // the owner set it (local and subscription models: 0)
	PriceSourcePublished = "published" // the provider's list price, built in (publishedPrices)
	PriceSourceEstimated = "estimated" // neither: FallbackIn/FallbackOut, counted high
)

// Price is what a model costs, USD per million tokens. A prompt with more
// than Over input tokens pays LongIn/LongOut for the whole call, as the
// providers that price by prompt length bill it (Over 0: one price).
type Price struct {
	In, Out         float64
	Over            int
	LongIn, LongOut float64
}

func (p Price) cost(u omnillm.Usage) int64 {
	in, out := p.In, p.Out
	if p.Over > 0 && u.InputTokens > p.Over {
		in, out = p.LongIn, p.LongOut
	}
	// USD per million tokens == micro-USD per token.
	return int64(float64(u.InputTokens)*in + float64(u.OutputTokens)*out)
}

// listed is one published price; From (YYYY-MM-DD, UTC) is the day a
// price the provider has announced starts. A model's entries are in date
// order and the last one that has started applies.
type listed struct {
	Price
	From string
}

// priceAsOf is the date the table below was read from the providers'
// pricing pages; design §10.3 lists the pages.
const priceAsOf = "2026-10-26"

// publishedPrices are the providers' standard (pay-as-you-go, base input,
// no cache, no batch) prices, by the API's host and the model id it lists.
// Only models whose page gives a plain per-token text price are here:
// a model missing (or priced "contact sales") stays on the high estimate,
// because counting too little would let a device run past its daily limit.
// Where a page gives two prices by hour or by input kind, the higher one.
var publishedPrices = map[string]map[string][]listed{
	// platform.claude.com/docs/en/about-claude/pricing
	"api.anthropic.com": {
		"claude-fable-5-1":  {{Price: Price{In: 10, Out: 50}}},
		"claude-fable-5":    {{Price: Price{In: 10, Out: 50}}},
		"claude-opus-5-5":   {{Price: Price{In: 4, Out: 20}}},
		"claude-opus-5":     {{Price: Price{In: 5, Out: 25}}},
		"claude-opus-4-8":   {{Price: Price{In: 5, Out: 25}}},
		"claude-opus-4-7":   {{Price: Price{In: 5, Out: 25}}},
		"claude-opus-4-6":   {{Price: Price{In: 5, Out: 25}}},
		"claude-opus-4-5":   {{Price: Price{In: 5, Out: 25}}},
		"claude-opus-4-1":   {{Price: Price{In: 15, Out: 75}}},
		"claude-opus-4":     {{Price: Price{In: 15, Out: 75}}},
		"claude-sonnet-5-5": {{Price: Price{In: 2, Out: 10}}},
		"claude-sonnet-5":   {{Price: Price{In: 2, Out: 10}}},
		"claude-sonnet-4-6": {{Price: Price{In: 3, Out: 15}}},
		"claude-sonnet-4-5": {{Price: Price{In: 3, Out: 15}}},
		"claude-sonnet-4":   {{Price: Price{In: 3, Out: 15}}},
		"claude-haiku-5-5":  {{Price: Price{In: 0.10, Out: 0.50, Over: 100000, LongIn: 0.50, LongOut: 2.50}}},
		"claude-haiku-4-5":  {{Price: Price{In: 1, Out: 5}}},
		"claude-3-5-haiku":  {{Price: Price{In: 0.80, Out: 4}}},
	},
	// developers.openai.com/api/docs/pricing (Standard); long context is
	// over 272K input tokens.
	"api.openai.com": {
		"gpt-6-astra":            {{Price: Price{In: 10, Out: 50, Over: 272000, LongIn: 20, LongOut: 75}}},
		"gpt-6.1-sol":            {{Price: Price{In: 2, Out: 10, Over: 272000, LongIn: 4, LongOut: 15}}},
		"gpt-6-sol":              {{Price: Price{In: 2, Out: 10, Over: 272000, LongIn: 4, LongOut: 15}}},
		"gpt-6-luna":             {{Price: Price{In: 0.10, Out: 0.50, Over: 272000, LongIn: 0.20, LongOut: 0.75}}},
		"gpt-5.6-sol":            {{Price: Price{In: 4, Out: 20, Over: 272000, LongIn: 8, LongOut: 30}}},
		"gpt-5.6-terra":          {{Price: Price{In: 2, Out: 12, Over: 272000, LongIn: 4, LongOut: 18}}},
		"gpt-5.6-luna":           {{Price: Price{In: 0.20, Out: 1.20, Over: 272000, LongIn: 0.40, LongOut: 1.80}}},
		"gpt-5.5":                {{Price: Price{In: 5, Out: 30, Over: 272000, LongIn: 10, LongOut: 45}}},
		"gpt-5.5-pro":            {{Price: Price{In: 30, Out: 180, Over: 272000, LongIn: 60, LongOut: 270}}},
		"gpt-5.4":                {{Price: Price{In: 2.50, Out: 15, Over: 272000, LongIn: 5, LongOut: 22.50}}},
		"gpt-5.4-pro":            {{Price: Price{In: 30, Out: 180, Over: 272000, LongIn: 60, LongOut: 270}}},
		"gpt-5.4-mini":           {{Price: Price{In: 0.75, Out: 4.50}}},
		"gpt-5.4-nano":           {{Price: Price{In: 0.20, Out: 1.25}}},
		"gpt-5.2":                {{Price: Price{In: 1.75, Out: 14}}},
		"gpt-5.2-pro":            {{Price: Price{In: 21, Out: 168}}},
		"gpt-5.1":                {{Price: Price{In: 1.25, Out: 10}}},
		"gpt-5":                  {{Price: Price{In: 1.25, Out: 10}}},
		"gpt-5-mini":             {{Price: Price{In: 0.25, Out: 2}}},
		"gpt-5-nano":             {{Price: Price{In: 0.05, Out: 0.40}}},
		"gpt-5-pro":              {{Price: Price{In: 15, Out: 120}}},
		"gpt-4.1":                {{Price: Price{In: 2, Out: 8}}},
		"gpt-4.1-mini":           {{Price: Price{In: 0.40, Out: 1.60}}},
		"gpt-4.1-nano":           {{Price: Price{In: 0.10, Out: 0.40}}},
		"gpt-4o":                 {{Price: Price{In: 2.50, Out: 10}}},
		"gpt-4o-2024-05-13":      {{Price: Price{In: 5, Out: 15}}},
		"gpt-4o-mini":            {{Price: Price{In: 0.15, Out: 0.60}}},
		"o1":                     {{Price: Price{In: 15, Out: 60}}},
		"o1-pro":                 {{Price: Price{In: 150, Out: 600}}},
		"o3-pro":                 {{Price: Price{In: 20, Out: 80}}},
		"o3":                     {{Price: Price{In: 2, Out: 8}}},
		"o4-mini":                {{Price: Price{In: 1.10, Out: 4.40}}},
		"o3-mini":                {{Price: Price{In: 1.10, Out: 4.40}}},
		"gpt-4-turbo-2024-04-09": {{Price: Price{In: 10, Out: 30}}},
		"gpt-4-0613":             {{Price: Price{In: 30, Out: 60}}},
		"gpt-3.5-turbo":          {{Price: Price{In: 0.50, Out: 1.50}}},
		"gpt-3.5-turbo-0125":     {{Price: Price{In: 0.50, Out: 1.50}}},
	},
	// ai.google.dev/gemini-api/docs/pricing (paid tier, Standard; the
	// text/image price, the one a chat pays). Long context is over 200K.
	"generativelanguage.googleapis.com": {
		"gemini-4-argon":                     {{Price: Price{In: 2, Out: 10}}, {Price: Price{In: 4, Out: 20}, From: "2026-12-01"}},
		"gemini-3.8-flash":                   {{Price: Price{In: 0.75, Out: 3.75}}, {Price: Price{In: 1.50, Out: 7.50}, From: "2027-01-01"}},
		"gemini-3.6-flash":                   {{Price: Price{In: 0.75, Out: 3.75}}, {Price: Price{In: 1.50, Out: 7.50}, From: "2027-01-01"}},
		"gemini-3.5-flash-lite":              {{Price: Price{In: 0.30, Out: 2.50}}},
		"gemini-3.1-pro-preview":             {{Price: Price{In: 2, Out: 12, Over: 200000, LongIn: 4, LongOut: 18}}},
		"gemini-3.1-pro-preview-customtools": {{Price: Price{In: 2, Out: 12, Over: 200000, LongIn: 4, LongOut: 18}}},
		"gemini-3.1-flash-lite":              {{Price: Price{In: 0.25, Out: 1.50}}},
		"gemini-3-flash-preview":             {{Price: Price{In: 0.50, Out: 3}}},
		"gemini-2.5-pro":                     {{Price: Price{In: 1.25, Out: 10, Over: 200000, LongIn: 2.50, LongOut: 15}}},
		"gemini-2.5-flash":                   {{Price: Price{In: 0.30, Out: 2.50}}},
		"gemini-2.5-flash-lite":              {{Price: Price{In: 0.10, Out: 0.40}}},
	},
	// api-docs.deepseek.com/quick_start/pricing: the peak-hour, cache-miss
	// price (off-peak is half). deepseek-v4-flash* are billed as Flash.
	"api.deepseek.com": {
		"deepseek-flash":               {{Price: Price{In: 0.30, Out: 1.20}}},
		"deepseek-v4-flash":            {{Price: Price{In: 0.30, Out: 1.20}}},
		"deepseek-v4-flash-vision-exp": {{Price: Price{In: 0.30, Out: 1.20}}},
		"deepseek-v4-pro":              {{Price: Price{In: 1.32, Out: 3.96}}},
	},
	// console.groq.com/docs/models; Llama models are "contact sales".
	"api.groq.com": {
		"openai/gpt-oss-120b":          {{Price: Price{In: 0.15, Out: 0.60}}},
		"openai/gpt-oss-20b":           {{Price: Price{In: 0.075, Out: 0.30}}},
		"openai/gpt-oss-safeguard-20b": {{Price: Price{In: 0.075, Out: 0.30}}},
		"qwen/qwen3.8-27b":             {{Price: Price{In: 0.80, Out: 4}}},
	},
}

// publishedHosts says which kind of source may use a host's prices: only
// the provider's own API, never a proxy or another server that happens to
// serve a model of the same name (its prices are its own).
var publishedHosts = map[string]provider.Kind{
	"api.anthropic.com":                 provider.Anthropic,
	"api.openai.com":                    provider.OpenAI,
	"generativelanguage.googleapis.com": provider.OpenAICompatible,
	"api.deepseek.com":                  provider.OpenAICompatible,
	"api.groq.com":                      provider.OpenAICompatible,
}

// snapshotDate is a dated snapshot's suffix ("-20250929", "-2024-08-06"),
// priced as the model it is a snapshot of unless listed on its own.
// Shorter suffixes ("-0613") are not stripped: OpenAI priced some of
// those apart, and a wrong match would count too little.
var snapshotDate = regexp.MustCompile(`-(\d{8}|\d{4}-\d{2}-\d{2})$`)

// publishedPrice is the provider's list price for a model id at cfg's
// source, on the day now: the one place that decides it (principle 1).
func publishedPrice(cfg provider.Config, upstream string, now time.Time) (Price, bool) {
	u, err := url.Parse(cfg.BaseURL)
	if err != nil || u.Scheme != "https" || u.Port() != "" {
		return Price{}, false
	}
	host := strings.ToLower(u.Hostname())
	if kind, ok := publishedHosts[host]; !ok || kind != cfg.Kind {
		return Price{}, false
	}
	models := publishedPrices[host]
	id := strings.TrimPrefix(upstream, "models/") // Gemini lists "models/gemini-…"
	entries, ok := models[id]
	if !ok {
		entries, ok = models[snapshotDate.ReplaceAllString(id, "")]
	}
	if !ok || len(entries) == 0 {
		return Price{}, false
	}
	// A change starts on its day wherever that day comes first (UTC+14),
	// so the hub is never behind a provider's increase; the table only
	// holds increases (TestPublishedPricesWellFormed), so starting early
	// can only count more.
	day := now.UTC().Add(14 * time.Hour).Format("2006-01-02")
	p := entries[0].Price
	for _, e := range entries[1:] {
		if e.From <= day {
			p = e.Price
		}
	}
	return p, true
}

// priceOf is the price a model's spend is counted at and where it comes
// from: the owner's own price wins, then the provider's published price,
// then the high estimate.
func (h *Hub) priceOf(m Model) (Price, string) {
	if m.PriceSet {
		return Price{In: m.PriceIn, Out: m.PriceOut}, PriceSourceSet
	}
	h.mu.Lock()
	c := h.clients[m.Provider]
	h.mu.Unlock()
	if c != nil {
		if p, ok := publishedPrice(c.Config(), m.Upstream, h.opts.Now()); ok {
			return p, PriceSourcePublished
		}
	}
	return Price{In: FallbackIn, Out: FallbackOut}, PriceSourceEstimated
}
