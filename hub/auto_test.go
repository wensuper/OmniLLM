package hub

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	omnillm "github.com/wensuper/OmniLLM"
	"github.com/wensuper/OmniLLM/internal/provider"
	"github.com/wensuper/OmniLLM/secret"
	"github.com/wensuper/OmniLLM/store"
)

func TestAutoTierOf(t *testing.T) {
	u := func(s string) omnillm.Request {
		return omnillm.Request{Messages: []omnillm.Message{{Role: "user", Content: s}}}
	}
	for _, c := range []struct {
		req  omnillm.Request
		want autoTier
	}{
		{u("你好"), tierQuick},
		{u("translate: good morning, how are you today, my friend? It has been a long time."), tierQuick},
		{u("北京今天适合出去玩吗？顺便推荐几个适合带孩子去的公园，最好离地铁近一点，人少一点的。"), tierNormal},
		{u("帮我重构这段代码，让它更容易测试"), tierHard},
		{u("why does this fail?\n```go\nfunc main() {}\n```"), tierHard},
		{u("Traceback (most recent call last):\n  File \"x.py\", line 1"), tierHard},
		{u(strings.Repeat("长", 1600)), tierHard},
		{omnillm.Request{Messages: []omnillm.Message{{Role: "user", Content: "这是什么", Images: []omnillm.Image{{MIME: "image/png", Data: []byte("x")}}}}}, tierImage},
		// The last user message decides, not an earlier hard one.
		{omnillm.Request{Messages: []omnillm.Message{{Role: "user", Content: "```x```"}, {Role: "assistant", Content: "ok"}, {Role: "user", Content: "谢谢"}}}, tierQuick},
	} {
		if got := autoTierOf(c.req); got != c.want {
			t.Errorf("%q: %s, want %s", c.req.Messages[len(c.req.Messages)-1].Content[:min(30, len(c.req.Messages[len(c.req.Messages)-1].Content))], got, c.want)
		}
	}
}

// autoUpstream is an OpenAI-compatible source with these models; each
// reply says which model wrote it. fail: models that answer 429.
type autoUpstream struct {
	*httptest.Server
	mu   sync.Mutex
	fail map[string]bool
	seen []string
}

func newAutoUpstream(t *testing.T, models ...string) *autoUpstream {
	u := &autoUpstream{fail: map[string]bool{}}
	u.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/models") {
			var d []map[string]string
			for _, m := range models {
				d = append(d, map[string]string{"id": m})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"data": d})
			return
		}
		var in struct {
			Model string `json:"model"`
		}
		_ = json.NewDecoder(r.Body).Decode(&in)
		u.mu.Lock()
		u.seen = append(u.seen, in.Model)
		bad := u.fail[in.Model]
		u.mu.Unlock()
		if bad {
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"message":"rate limit"}}`))
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"from %s\"}}]}\n\n", in.Model)
		fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":3,\"completion_tokens\":2}}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(u.Close)
	return u
}

func autoHub(t *testing.T) (*Hub, *autoUpstream, *autoUpstream) {
	provider.AllowLoopbackForTests()
	ctx := context.Background()
	h, err := New(ctx, Options{Secrets: secret.NewMemory(), Store: store.NewMemory()})
	if err != nil {
		t.Fatal(err)
	}
	a := newAutoUpstream(t, "claude-haiku-4", "claude-sonnet-5", "claude-opus-5")
	b := newAutoUpstream(t, "gpt-6-sol")
	for name, u := range map[string]*autoUpstream{"anth": a, "oai": b} {
		cfg := provider.Config{Name: name, Kind: provider.OpenAICompatible, BaseURL: u.URL, NeedsKey: true}
		if _, err := h.AddProvider(ctx, cfg, "sk-0123456789abcdef"); err != nil {
			t.Fatal(err)
		}
	}
	return h, a, b
}

// ask sends msgs through Auto and returns who answered and the text.
func ask(t *testing.T, h *Hub, msgs ...omnillm.Message) (label, text string, err error) {
	t.Helper()
	s, err := h.Complete(context.Background(), omnillm.Request{Model: AutoModel, Caller: omnillm.Interactive, Messages: msgs})
	if err != nil {
		return "", "", err
	}
	defer s.Close()
	var b strings.Builder
	for {
		ch, err := s.Recv()
		if ch.Model != "" && label == "" {
			label = ch.Model
		}
		b.WriteString(ch.Text)
		if errors.Is(err, io.EOF) {
			return label, b.String(), nil
		}
		if err != nil {
			return label, b.String(), err
		}
	}
}

func user(s string) omnillm.Message { return omnillm.Message{Role: "user", Content: s} }

func TestAutoChoosesByTierAndSpreads(t *testing.T) {
	h, _, _ := autoHub(t)
	if _, text, err := ask(t, h, user("你好")); err != nil || text != "from claude-haiku-4" {
		t.Fatalf("quick: %q %v", text, err)
	}
	if _, text, err := ask(t, h, user("帮我重构这段代码，让它更容易测试")); err != nil || text != "from claude-opus-5" {
		t.Fatalf("hard: %q %v", text, err)
	}
	// Normal: Sonnet and GPT-6-Sol are equals; anth was used twice in the
	// last hour, oai not at all.
	if _, text, err := ask(t, h, user("北京今天适合出去玩吗？顺便推荐几个适合带孩子去的公园，最好离地铁近一点。")); err != nil || text != "from gpt-6-sol" {
		t.Fatalf("normal, spread: %q %v", text, err)
	}
}

func TestAutoStaysWithTheConversation(t *testing.T) {
	h, _, _ := autoHub(t)
	first := user("北京今天适合出去玩吗？顺便推荐几个适合带孩子去的公园，最好离地铁近一点。")
	_, r1, err := ask(t, h, first)
	if err != nil {
		t.Fatal(err)
	}
	// Other conversations use the other source meanwhile.
	for i := 0; i < 3; i++ {
		_, _, _ = ask(t, h, user(fmt.Sprintf("另一个对话 %d：周末去哪里吃饭比较好，有没有推荐的餐厅和菜？", i)))
	}
	// A quick follow-up stays; a harder one moves up.
	_, r2, err := ask(t, h, first, omnillm.Message{Role: "assistant", Content: r1}, user("好的"))
	if err != nil || r2 != r1 {
		t.Fatalf("follow-up moved: %q → %q %v", r1, r2, err)
	}
	_, r3, err := ask(t, h, first, omnillm.Message{Role: "assistant", Content: r1}, user("好的"), omnillm.Message{Role: "assistant", Content: r2},
		user("帮我写一个脚本，详细分析一下这些公园的人流，一步一步来"))
	if err != nil || r3 != "from claude-opus-5" {
		t.Fatalf("harder follow-up: %q %v", r3, err)
	}
}

func TestAutoPassesOverAFailingModel(t *testing.T) {
	h, a, _ := autoHub(t)
	a.mu.Lock()
	a.fail["claude-opus-5"] = true
	a.mu.Unlock()
	label, text, err := ask(t, h, user("帮我重构这段代码，让它更容易测试"))
	if err != nil || text == "" || strings.Contains(text, "opus") {
		t.Fatalf("%q %q %v", label, text, err)
	}
	if !strings.Contains(label, strings.TrimPrefix(text, "from ")) {
		t.Fatalf("label %q for %q", label, text)
	}
}

func TestAutoLeavesOutWhatTheOwnerSays(t *testing.T) {
	h, _, _ := autoHub(t)
	ctx := context.Background()
	if err := h.SetModelAuto(ctx, "anth/claude-haiku-4", false); err != nil {
		t.Fatal(err)
	}
	if _, text, err := ask(t, h, user("你好")); err != nil || text == "from claude-haiku-4" {
		t.Fatalf("%q %v", text, err)
	}
	ms, _ := h.OwnerModels(ctx)
	for _, m := range ms {
		if m.ID == "anth/claude-haiku-4" && !m.NoAuto {
			t.Fatal("NoAuto not shown to the owner")
		}
	}
	// Everything out: Auto says so, in words.
	for _, id := range []string{"anth/claude-sonnet-5", "anth/claude-opus-5", "oai/gpt-6-sol"} {
		_ = h.SetModelAuto(ctx, id, false)
	}
	if _, _, err := ask(t, h, user("你好")); omnillm.AsError(err).Msg != omnillm.MsgNoAutoModel {
		t.Fatalf("%v", err)
	}
	if err := h.SetModelAuto(ctx, "anth/claude-haiku-4", true); err != nil {
		t.Fatal(err)
	}
	if _, text, _ := ask(t, h, user("你好")); text != "from claude-haiku-4" {
		t.Fatal(text)
	}
}

// A device sees Auto first, and Auto only chooses what the device may use.
func TestAutoForADevice(t *testing.T) {
	h, _, _ := autoHub(t)
	ctx := context.Background()
	d := Device{ID: "d1", Models: []string{"oai/gpt-6-sol", "anth/claude-haiku-4"}, Budget: NoLimit}
	ms, err := h.modelsFor(ctx, d)
	if err != nil || len(ms) != 3 || ms[0].ID != AutoModel || ms[0].Kind != AutoModel {
		t.Fatalf("%+v %v", ms, err)
	}
	s, err := h.complete(ctx, d.principal(), omnillm.Request{Model: AutoModel, Caller: omnillm.Interactive,
		Messages: []omnillm.Message{user("帮我重构这段代码，让它更容易测试")}})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var b strings.Builder
	for {
		ch, err := s.Recv()
		b.WriteString(ch.Text)
		if err != nil {
			break
		}
	}
	if !strings.Contains(b.String(), "gpt-6-sol") && !strings.Contains(b.String(), "haiku") {
		t.Fatalf("chose a model the device may not use: %q", b.String())
	}
	// One model only: nothing to choose, no Auto.
	if ms, _ := h.modelsFor(ctx, Device{ID: "d2", Models: []string{"oai/gpt-6-sol"}}); len(ms) != 1 || ms[0].ID == AutoModel {
		t.Fatalf("%+v", ms)
	}
	// Paused: refused as for any model.
	if _, err := h.complete(ctx, Device{ID: "d1", Paused: true}.principal(), omnillm.Request{Model: AutoModel, Caller: omnillm.Interactive,
		Messages: []omnillm.Message{user("hi")}}); omnillm.AsError(err).Code != omnillm.CodeDevicePaused {
		t.Fatal(err)
	}
}

func TestAutoGroupOrder(t *testing.T) {
	g := func(tier autoTier, id, name, kind string) int {
		return autoGroup(tier, omnillm.ModelInfo{ID: id, DisplayName: name, Kind: kind})
	}
	if a, b := g(tierHard, "claude-code/opus", "Claude Opus 5.5", "claude-code"), g(tierHard, "antigravity/claude-opus-4-6-thinking", "Claude Opus 4.6 (Thinking)", "antigravity"); a >= b {
		t.Fatalf("Antigravity's Claude must stand behind the Claude subscription: %d %d", a, b)
	}
	if a, b := g(tierNormal, "claude-code/sonnet", "Claude Sonnet 5.5", "claude-code"), g(tierNormal, "antigravity/default", "Gemini 3.8 Flash (High)", "antigravity"); a >= b {
		t.Fatalf("Sonnet before Gemini Flash for normal: %d %d", a, b)
	}
	if a, b := g(tierQuick, "antigravity/gemini-3.8-flash-low", "Gemini 3.8 Flash (Low)", "antigravity"), g(tierQuick, "claude-code/opus", "Claude Opus 5.5", "claude-code"); a >= b {
		t.Fatalf("quick: Flash Low before Opus: %d %d", a, b)
	}
}
