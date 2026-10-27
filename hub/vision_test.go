package hub_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	omnillm "github.com/wensuper/OmniLLM"
)

// A text-only model refuses pictures the way llama.cpp does (a 500); the
// hub leaves earlier pictures out and answers, and remembers the model.
func TestTextOnlyModelSkipsEarlierPictures(t *testing.T) {
	e := setup(t, nil)
	var withImage, withoutImage atomic.Int32
	var lastBody atomic.Value
	e.fake.setHook(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		if r.URL.Path == "/models" {
			next(w, r)
			return
		}
		b, _ := io.ReadAll(r.Body)
		lastBody.Store(string(b))
		if bytes.Contains(b, []byte("image_url")) {
			withImage.Add(1)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(500)
			_, _ = io.WriteString(w, `{"error":{"code":500,"message":"image input is not supported - hint: if this is unexpected, you may need to provide the mmproj","type":"server_error"}}`)
			return
		}
		withoutImage.Add(1)
		r.Body = io.NopCloser(bytes.NewReader(b))
		next(w, r)
	})
	ctx := context.Background()
	png := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x02\x00\x00\x00\x90wS\xde\x00\x00\x00\x0cIDATx\x9cc\xf8\xff\xff?\x00\x05\xfe\x02\xfe\xa7V\xfa\x8a\x00\x00\x00\x00IEND\xaeB`\x82")
	pic := omnillm.Message{Role: "user", Content: "look", Images: []omnillm.Image{{MIME: "image/png", Data: png}}}
	run := func(msgs ...omnillm.Message) (string, error) {
		s, err := e.hub.Complete(ctx, omnillm.Request{Model: "fake/m1", Caller: omnillm.Interactive, Messages: msgs})
		if err != nil {
			return "", err
		}
		defer s.Close()
		var b strings.Builder
		for {
			ch, err := s.Recv()
			if errors.Is(err, io.EOF) {
				return b.String(), nil
			}
			if err != nil {
				return b.String(), err
			}
			b.WriteString(ch.Text)
		}
	}

	// The picture is from an earlier turn: answered without it.
	if text, err := run(pic, omnillm.Message{Role: "assistant", Content: "a cat"}, omnillm.Message{Role: "user", Content: "hello, and now?"}); err != nil || text == "" {
		t.Fatalf("earlier picture: %q %v", text, err)
	}
	if withImage.Load() != 1 || withoutImage.Load() != 1 {
		t.Fatalf("calls with/without image: %d/%d, want 1/1", withImage.Load(), withoutImage.Load())
	}
	if b, _ := lastBody.Load().(string); !strings.Contains(b, "left out") {
		t.Fatalf("the model is not told a picture was there: %s", b)
	}

	// Known now: the next turn leaves it out at once, no refused call.
	if _, err := run(pic, omnillm.Message{Role: "assistant", Content: "a cat"}, omnillm.Message{Role: "user", Content: "hello, more"}); err != nil {
		t.Fatal(err)
	}
	if withImage.Load() != 1 {
		t.Fatalf("a known text-only model got the picture again (%d)", withImage.Load())
	}

	// A picture in the new message: say the model can't read it.
	_, err := run(omnillm.Message{Role: "user", Content: "what is this?", Images: pic.Images})
	if oe := omnillm.AsError(err); oe == nil || oe.Msg != omnillm.MsgNoVision {
		t.Fatalf("new picture: %v, want the no-vision message", err)
	}
}
