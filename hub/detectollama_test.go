package hub

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	omnillm "github.com/wensuper/OmniLLM"
	"github.com/wensuper/OmniLLM/secret"
	"github.com/wensuper/OmniLLM/store"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// A running Ollama with only embedding or rerank models is found, not
// connected, and the owner is told what to pull.
func TestDetectNowOllamaWithoutChatModels(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	t.Setenv("CODEX_HOME", t.TempDir())
	models := `{"data":[{"id":"nomic-embed-text"},{"id":"bge-reranker"}]}`
	tr := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}},
			Body: io.NopCloser(strings.NewReader(models)), Request: r}, nil
	})
	ctx := context.Background()
	h, err := New(ctx, Options{Secrets: secret.NewMemory(), Store: store.NewMemory(), Transport: tr})
	if err != nil {
		t.Fatal(err)
	}
	kinds, failed, err := h.DetectNow(ctx)
	if err != nil || len(kinds) != 0 {
		t.Fatalf("%v %v", kinds, err)
	}
	if len(failed) != 1 || failed[0].Kind != "ollama" || omnillm.AsError(failed[0].Err).Msg != omnillm.MsgNoChatModel {
		t.Fatalf("%+v", failed)
	}
	// With a chat model it connects.
	models = `{"data":[{"id":"nomic-embed-text"},{"id":"qwen3"}]}`
	kinds, failed, err = h.DetectNow(ctx)
	if err != nil || len(failed) != 0 || len(kinds) != 1 || kinds[0] != "ollama" {
		t.Fatalf("%v %+v %v", kinds, failed, err)
	}
}

// noOllama is a machine with no Ollama running, whatever this one has.
var noOllama = roundTripFunc(func(r *http.Request) (*http.Response, error) {
	return nil, errors.New("connection refused")
})
