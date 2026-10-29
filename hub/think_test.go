package hub_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/wensuper/OmniLLM"
)

// A thinking model's reasoning reaches the device as Think, ahead of and
// apart from the answer, through the same end-to-end session.
func TestReasoningReachesTheDevice(t *testing.T) {
	e := setup(t, nil)
	e.fake.setHook(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		if r.URL.Path == "/models" {
			next(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		for _, d := range []string{
			`{"choices":[{"delta":{"content":"","reasoning":"1 plus 1 "}}]}`,
			`{"choices":[{"delta":{"content":"","reasoning":"is 2."}}]}`,
			`{"choices":[{"delta":{"content":"2"},"finish_reason":"stop"}]}`,
		} {
			fmt.Fprintf(w, "data: %s\n\n", d)
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
	})
	d, _ := e.pairOK()
	s, err := d.Complete(context.Background(), omnillm.Request{Model: "fake/m1", Messages: []omnillm.Message{{Role: "user", Content: "1+1?"}}, Caller: omnillm.Interactive})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var think, text string
	for {
		c, err := s.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if c.Think != "" && text != "" {
			t.Fatal("reasoning came after the answer had started")
		}
		think += c.Think
		text += c.Text
	}
	if think != "1 plus 1 is 2." || text != "2" {
		t.Fatalf("think %q, text %q; want the reasoning and the answer apart", think, text)
	}
}
