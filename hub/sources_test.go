package hub

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	omnillm "github.com/wensuper/OmniLLM"
	"github.com/wensuper/OmniLLM/internal/provider"
	"github.com/wensuper/OmniLLM/secret"
	"github.com/wensuper/OmniLLM/store"
)

func TestAddProviderSaysWhatIsWrongBeforeAsking(t *testing.T) {
	ctx := context.Background()
	var calls atomic.Int32
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"m1"}]}`))
	}))
	defer up.Close()
	h, err := New(ctx, Options{Secrets: secret.NewMemory(), Store: store.NewMemory()})
	if err != nil {
		t.Fatal(err)
	}
	cfg := func(name, base string) provider.Config {
		return provider.Config{Name: name, Kind: provider.OpenAICompatible, BaseURL: base, NeedsKey: true}
	}
	for _, c := range []struct {
		cfg  provider.Config
		key  string
		want string
	}{
		{cfg("my groq", up.URL), "sk-0123456789abcdef", omnillm.MsgBadSource},
		{cfg("groq_1", up.URL), "sk-0123456789abcdef", omnillm.MsgBadSource},
		{cfg("groq", "api.groq.com/openai/v1"), "sk-0123456789abcdef", omnillm.MsgBadSource},
		{cfg("groq", up.URL), "sk-12", omnillm.MsgKeyMalformed},
		{cfg("groq", up.URL), "sk-0123456789\nabcdef", omnillm.MsgKeyMalformed},
		{cfg("groq", up.URL), "sk-0123 456789abcdef", omnillm.MsgKeyMalformed},
	} {
		_, err := h.AddProvider(ctx, c.cfg, c.key)
		if e := omnillm.AsError(err); e == nil || e.Msg != c.want {
			t.Errorf("%q %q: %v", c.cfg.Name, c.key, err)
		}
		if strings.Contains(omnillm.Describe("en", err), "model name") {
			t.Errorf("%q: %s", c.cfg.Name, omnillm.Describe("en", err))
		}
	}
	if n := calls.Load(); n != 0 {
		t.Fatalf("%d requests sent with a bad name or key", n)
	}
	if _, err := h.AddProvider(ctx, cfg("groq-2", up.URL), "  sk-0123456789abcdef\n"); err != nil {
		t.Fatal(err)
	}
}

// failingStore refuses to keep one key.
type failingStore struct {
	store.Store
	key string
}

func (f failingStore) Put(ctx context.Context, key string, v []byte) error {
	if key == f.key {
		return errors.New("disk full")
	}
	return f.Store.Put(ctx, key, v)
}

func TestDetectNowReportsKindsAndFailures(t *testing.T) {
	path, _ := fakeClaude(t)
	t.Setenv("PATH", filepath.Dir(path))
	t.Setenv("HOME", t.TempDir())
	t.Setenv("CODEX_HOME", t.TempDir())
	ctx := context.Background()
	h, err := New(ctx, Options{Secrets: secret.NewMemory(), Store: failingStore{store.NewMemory(), "provider/claude-code"}, Transport: noOllama})
	if err != nil {
		t.Fatal(err)
	}
	kinds, failed, err := h.DetectNow(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range kinds {
		if k == "claude-code" {
			t.Fatalf("connected although it failed: %v", kinds)
		}
	}
	if len(failed) != 1 || failed[0].Kind != "claude-code" || failed[0].Err == nil || omnillm.Describe("zh", failed[0].Err) == "" {
		t.Fatalf("%+v", failed)
	}
	// The request ending is the only error.
	cctx, cancel := context.WithCancel(ctx)
	cancel()
	if _, _, err := h.DetectNow(cctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("%v", err)
	}
}

func TestCodexIsTheLatestDefault(t *testing.T) {
	h := &Hub{clients: map[string]*provider.Client{}}
	mi := h.info(Model{ID: "codex/default", Provider: "codex", Upstream: "default", DisplayName: "default"})
	if mi.Kind != "codex" || !mi.Latest || mi.Version != "" {
		t.Fatalf("%+v", mi)
	}
}
