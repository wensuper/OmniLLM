package localapi

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/wensuper/OmniLLM"
)

type thinkStream struct {
	chunks []omnillm.Chunk
	i      int
}

func (s *thinkStream) Recv() (omnillm.Chunk, error) {
	if s.i < len(s.chunks) {
		s.i++
		return s.chunks[s.i-1], nil
	}
	return omnillm.Chunk{}, io.EOF
}
func (s *thinkStream) Usage() omnillm.Usage { return omnillm.Usage{} }
func (s *thinkStream) FinishReason() string { return "stop" }
func (s *thinkStream) Close() error         { return nil }

type thinkBackend struct{ fakeBackend }

func (b *thinkBackend) Complete(context.Context, string, omnillm.Request) (omnillm.Stream, error) {
	return &thinkStream{chunks: []omnillm.Chunk{{Think: "first "}, {Think: "then"}, {Text: "answer"}}}, nil
}

// A streamed reply passes the model's reasoning on as reasoning_content,
// all of it (the first piece too, which is read before the 200), and the
// answer as content.
func TestStreamPassesReasoning(t *testing.T) {
	w := do(t, Handler(&thinkBackend{}), "POST", "/v1/chat/completions",
		`{"model":"ollama/qwen3","stream":true,"messages":[{"role":"user","content":"hi"}]}`, nil)
	var think, text strings.Builder
	for _, l := range strings.Split(w.Body.String(), "\n") {
		if !strings.HasPrefix(l, "data: {") {
			continue
		}
		var ev struct {
			Choices []struct {
				Delta struct {
					Content          string `json:"content"`
					ReasoningContent string `json:"reasoning_content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(l[6:]), &ev); err != nil {
			t.Fatal(err)
		}
		for _, c := range ev.Choices {
			think.WriteString(c.Delta.ReasoningContent)
			text.WriteString(c.Delta.Content)
		}
	}
	if think.String() != "first then" || text.String() != "answer" {
		t.Fatalf("reasoning %q, answer %q", think.String(), text.String())
	}
}
