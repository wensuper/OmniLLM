package hub

import (
	"context"
	"strings"
	"sync"
	"time"

	omnillm "github.com/wensuper/OmniLLM"

	"github.com/wensuper/OmniLLM/internal/egress"
	"github.com/wensuper/OmniLLM/internal/provider"
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
			return egress.Limits{MaxConcurrent: 1, RequestsPerMinute: 12, MaxQueue: 4}
		}
		if host != nil {
			return host(p)
		}
		return egress.Limits{}
	}
}

func subscriptionModelName(k provider.Kind, id string) string {
	if k == provider.Codex {
		return "Codex (ChatGPT)"
	}
	return "Claude " + strings.ToUpper(id[:1]) + id[1:]
}

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
		return &omnillm.Error{Code: omnillm.CodeBusy, Msg: omnillm.MsgSubscriptionPace,
			Args: []string{omnillm.RoundDuration("en", wait)}, RetryAfter: wait}
	}
	s.calls[who] = append(list, now)
	return nil
}

// DetectSubscriptions lists the subscription programs installed and logged
// in on this machine that are not connected yet.
func (h *Hub) DetectSubscriptions(ctx context.Context) []provider.Config {
	var out []provider.Config
	for _, k := range []provider.Kind{provider.ClaudeCode, provider.Codex} {
		if h.subscription(string(k)) {
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
