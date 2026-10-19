package hub_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	omnillm "github.com/wensuper/OmniLLM"
)

// An image from a device reaches an API provider as an inline image part,
// and a large one (near the 2 MB cap) makes it through the relay intact.
func TestImageFromDeviceReachesProvider(t *testing.T) {
	e := setup(t, nil)
	var body []byte
	e.fake.setHook(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		if r.URL.Path != "/models" {
			body, _ = io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewReader(body))
		}
		next(w, r)
	})
	d, _ := e.pairOK()
	png, err := os.ReadFile("../internal/provider/testdata/red.png")
	if err != nil {
		t.Fatal(err)
	}
	big := append(append([]byte{}, png...), bytes.Repeat([]byte{0}, omnillm.MaxImageBytes-len(png)-10)...)
	ctx := context.Background()
	s, err := d.Complete(ctx, omnillm.Request{Model: "fake/m1", Caller: omnillm.Interactive, Messages: []omnillm.Message{
		{Role: "user", Content: "what color?", Images: []omnillm.Image{{MIME: "image/png", Data: png}, {MIME: "image/png", Data: big}}}}})
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := s.Recv(); err != nil {
			break
		}
	}
	s.Close()
	if !strings.Contains(string(body), `"type":"image_url"`) || !strings.Contains(string(body), "data:image/png;base64,") || len(body) < omnillm.MaxImageBytes {
		t.Fatalf("provider got %d bytes: %.200s", len(body), body)
	}
}
