package hub

import (
	"context"
	"testing"
	"time"

	omnillm "github.com/wensuper/OmniLLM"
	"github.com/wensuper/OmniLLM/internal/provider"
	"github.com/wensuper/OmniLLM/secret"
	"github.com/wensuper/OmniLLM/store"
)

var priceDay = time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)

func cfgAt(kind provider.Kind, base string) provider.Config {
	return provider.Config{Name: "x", Kind: kind, BaseURL: base, NeedsKey: true}
}

// Each source finds its own models' prices, by the id its API lists.
func TestPublishedPriceLookup(t *testing.T) {
	anthropic := cfgAt(provider.Anthropic, "https://api.anthropic.com")
	openai := cfgAt(provider.OpenAI, "https://api.openai.com/v1")
	gemini := cfgAt(provider.OpenAICompatible, "https://generativelanguage.googleapis.com/v1beta/openai/")
	deepseek := cfgAt(provider.OpenAICompatible, "https://api.deepseek.com")
	groq := cfgAt(provider.OpenAICompatible, "https://api.groq.com/openai/v1")
	for _, c := range []struct {
		cfg     provider.Config
		id      string
		in, out float64
	}{
		{anthropic, "claude-sonnet-4-5-20250929", 3, 15}, // a dated snapshot is its model
		{anthropic, "claude-sonnet-5-5", 2, 10},
		{anthropic, "claude-3-5-haiku-20241022", 0.80, 4},
		{openai, "gpt-4o-mini", 0.15, 0.60},
		{openai, "gpt-4o-2024-08-06", 2.50, 10},
		{openai, "gpt-4o-2024-05-13", 5, 15}, // listed apart, its own price
		{gemini, "models/gemini-2.5-flash", 0.30, 2.50},
		{gemini, "gemini-2.5-flash-lite", 0.10, 0.40},
		{deepseek, "deepseek-v4-pro", 1.32, 3.96},
		{groq, "openai/gpt-oss-20b", 0.075, 0.30},
	} {
		p, ok := publishedPrice(c.cfg, c.id, priceDay)
		if !ok || p.In != c.in || p.Out != c.out {
			t.Errorf("%s at %s: %+v %v, want %v/%v", c.id, c.cfg.BaseURL, p, ok, c.in, c.out)
		}
	}
}

// Unknown models, look-alike ids and other servers stay on the estimate:
// a wrong published price would count too little.
func TestPublishedPriceRefuses(t *testing.T) {
	for _, c := range []struct {
		cfg provider.Config
		id  string
	}{
		{cfgAt(provider.OpenAI, "https://api.openai.com/v1"), "gpt-3.5-turbo-0613"}, // only date suffixes are stripped
		{cfgAt(provider.OpenAI, "https://api.openai.com/v1"), "gpt-4o-mini-tts"},    // no prefix matching
		{cfgAt(provider.OpenAICompatible, "https://generativelanguage.googleapis.com/v1beta/openai"), "gemini-2.5-flash-image"},
		{cfgAt(provider.OpenAICompatible, "https://api.groq.com/openai/v1"), "llama-3.3-70b-versatile"}, // contact sales
		{cfgAt(provider.OpenAICompatible, "https://api.deepseek.com"), "deepseek-chat"},                 // not on the page
		{cfgAt(provider.OpenAICompatible, "https://my-proxy.example.com/v1"), "gpt-4o-mini"},            // a proxy's prices are its own
		{cfgAt(provider.OpenAICompatible, "https://api.openai.com/v1"), "gpt-4o-mini"},                  // the host belongs to another kind
		{cfgAt(provider.OpenAI, "http://api.openai.com/v1"), "gpt-4o-mini"},
		{cfgAt(provider.OpenAI, "https://api.openai.com:8443/v1"), "gpt-4o-mini"},
		{cfgAt(provider.Anthropic, "https://api.anthropic.com"), "claude-mythos-5"}, // ids not on the page
	} {
		if p, ok := publishedPrice(c.cfg, c.id, priceDay); ok {
			t.Errorf("%s at %s got %+v", c.id, c.cfg.BaseURL, p)
		}
	}
}

// An announced price change applies from its day; prompts over a long-
// context threshold pay the long price for the whole call.
func TestPublishedPriceDateAndLength(t *testing.T) {
	gemini := cfgAt(provider.OpenAICompatible, "https://generativelanguage.googleapis.com/v1beta/openai")
	before, _ := publishedPrice(gemini, "gemini-3.8-flash", time.Date(2026, 12, 31, 9, 59, 0, 0, time.UTC))
	after, _ := publishedPrice(gemini, "gemini-3.8-flash", time.Date(2026, 12, 31, 10, 0, 0, 0, time.UTC)) // 1 January in UTC+14
	if before.In != 0.75 || after.In != 1.50 || after.Out != 7.50 {
		t.Fatalf("dated prices: before %+v after %+v", before, after)
	}
	haiku, _ := publishedPrice(cfgAt(provider.Anthropic, "https://api.anthropic.com"), "claude-haiku-5-5", priceDay)
	if got := haiku.cost(omnillm.Usage{InputTokens: 100000, OutputTokens: 1000}); got != 100000*0.10+1000*0.50 {
		t.Fatalf("at the threshold: %d", got)
	}
	if got := haiku.cost(omnillm.Usage{InputTokens: 100001, OutputTokens: 1000}); got != 52500 {
		t.Fatalf("over the threshold: %d", got)
	}
}

// Every entry of the table is well formed (principle 14).
func TestPublishedPricesWellFormed(t *testing.T) {
	if _, err := time.Parse("2006-01-02", priceAsOf); err != nil {
		t.Fatal(err)
	}
	for host := range publishedHosts {
		if len(publishedPrices[host]) == 0 {
			t.Errorf("%s has no prices", host)
		}
	}
	for host, models := range publishedPrices {
		if _, ok := publishedHosts[host]; !ok {
			t.Errorf("%s has prices but no source kind", host)
		}
		for id, entries := range models {
			if len(entries) == 0 || entries[0].From != "" {
				t.Errorf("%s %s: the first price must have no start day", host, id)
			}
			if snapshotDate.MatchString(id) && models[snapshotDate.ReplaceAllString(id, "")] == nil && host != "api.openai.com" {
				t.Errorf("%s %s: a dated id that is not an exception", host, id)
			}
			last := ""
			for i, e := range entries {
				if i > 0 {
					if _, err := time.Parse("2006-01-02", e.From); err != nil || e.From <= last {
						t.Errorf("%s %s: start days out of order: %q", host, id, e.From)
					}
					last = e.From
					if prev := entries[i-1].Price; e.In < prev.In || e.Out < prev.Out || e.LongIn < prev.LongIn || e.LongOut < prev.LongOut {
						t.Errorf("%s %s: a later price is lower (starting early would count too little)", host, id)
					}
				}
				p := e.Price
				if p.In <= 0 || p.Out <= 0 || p.Out > 1000 || p.In > 1000 {
					t.Errorf("%s %s: price %+v", host, id, p)
				}
				if p.Over != 0 && (p.Over < 1000 || p.LongIn < p.In || p.LongOut < p.Out) {
					t.Errorf("%s %s: long-context price %+v", host, id, p)
				}
				if p.Over == 0 && (p.LongIn != 0 || p.LongOut != 0) {
					t.Errorf("%s %s: long price without a threshold %+v", host, id, p)
				}
			}
		}
	}
}

// The owner's price wins over the published one, which wins over the
// estimate; the owner's model list says which it is.
func TestPriceOfOrder(t *testing.T) {
	ctx := context.Background()
	h, err := New(ctx, Options{Secrets: secret.NewMemory(), Store: store.NewMemory(), Now: func() time.Time { return priceDay }})
	if err != nil {
		t.Fatal(err)
	}
	c, err := provider.New(cfgAt(provider.OpenAI, "https://api.openai.com/v1"), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	h.mu.Lock()
	h.clients["openai"] = c
	h.mu.Unlock()
	for _, m := range []Model{
		{ID: "openai/gpt-4o-mini", Provider: "openai", Upstream: "gpt-4o-mini"},
		{ID: "openai/gpt-4o", Provider: "openai", Upstream: "gpt-4o", PriceIn: 1, PriceOut: 2, PriceSet: true},
		{ID: "openai/gpt-new", Provider: "openai", Upstream: "gpt-new"},
		{ID: "gone/gpt-4o-mini", Provider: "gone", Upstream: "gpt-4o-mini"}, // its source is not connected
	} {
		if err := h.putModel(ctx, m); err != nil {
			t.Fatal(err)
		}
	}
	want := map[string]struct {
		in, out float64
		src     string
	}{
		"openai/gpt-4o-mini": {0.15, 0.60, PriceSourcePublished},
		"openai/gpt-4o":      {1, 2, PriceSourceSet},
		"openai/gpt-new":     {FallbackIn, FallbackOut, PriceSourceEstimated},
		"gone/gpt-4o-mini":   {FallbackIn, FallbackOut, PriceSourceEstimated},
	}
	for id, w := range want {
		m, err := h.getModel(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if p, src := h.priceOf(m); p.In != w.in || p.Out != w.out || src != w.src {
			t.Errorf("%s: %+v %s, want %+v", id, p, src, w)
		}
	}
	m, _ := h.getModel(ctx, "openai/gpt-4o-mini")
	p, _ := h.priceOf(m)
	if got := p.cost(omnillm.Usage{InputTokens: 1000, OutputTokens: 1000}); got != 750 {
		t.Fatalf("1k in + 1k out on gpt-4o-mini: %d micro-USD, want 750", got)
	}
}
