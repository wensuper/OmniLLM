package localapi

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func FuzzChatBody(f *testing.F) {
	f.Add(chatBody)
	f.Add(`{"model":"m/x","messages":[{"role":"user","content":[{"type":"text","text":"a"}]}],"response_format":{"type":"json_schema","json_schema":{"schema":{}}}}`)
	h := Handler(&fakeBackend{})
	f.Fuzz(func(t *testing.T, body string) {
		r := httptest.NewRequest("POST", "http://127.0.0.1/v1/chat/completions", strings.NewReader(body))
		r.Host = "127.0.0.1:1"
		r.Header.Set("Authorization", "Bearer "+good)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code == 500 {
			t.Fatalf("500 for %q: %s", body, w.Body)
		}
	})
}
