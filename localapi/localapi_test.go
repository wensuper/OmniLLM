package localapi

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	omnillm "github.com/wensuper/OmniLLM"
)

const good = "omni_app_good"

type fakeStream struct {
	parts []string
	err   error
	i     int
}

func (s *fakeStream) Recv() (omnillm.Chunk, error) {
	if s.i < len(s.parts) {
		s.i++
		return omnillm.Chunk{Text: s.parts[s.i-1]}, nil
	}
	if s.err != nil {
		return omnillm.Chunk{}, s.err
	}
	return omnillm.Chunk{}, io.EOF
}
func (s *fakeStream) Usage() omnillm.Usage { return omnillm.Usage{InputTokens: 3, OutputTokens: 2} }
func (s *fakeStream) FinishReason() string { return "stop" }
func (s *fakeStream) Close() error         { return nil }

type fakeBackend struct {
	last omnillm.Request
	err  error
	mid  error
}

func (b *fakeBackend) Models(_ context.Context, tok string) ([]omnillm.ModelInfo, error) {
	if tok != good {
		return nil, omnillm.ErrInvalidToken
	}
	return []omnillm.ModelInfo{{ID: "anthropic/claude", Provider: "anthropic", Locality: omnillm.Remote, DisplayName: "Claude"},
		{ID: "ollama/qwen3", Provider: "ollama", Locality: omnillm.Local}}, nil
}

func (b *fakeBackend) Complete(_ context.Context, tok string, req omnillm.Request) (omnillm.Stream, error) {
	if tok != good {
		return nil, &omnillm.Error{Code: omnillm.CodeInvalidToken, Msg: omnillm.MsgInvalidToken}
	}
	b.last = req
	if b.err != nil {
		return nil, b.err
	}
	return &fakeStream{parts: []string{"Hel", "lo"}, err: b.mid}, nil
}

func do(t *testing.T, h http.Handler, method, path, body string, hdr map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, "http://127.0.0.1:8765"+path, strings.NewReader(body))
	r.Host = "127.0.0.1:8765"
	r.Header.Set("Authorization", "Bearer "+good)
	for k, v := range hdr {
		if v == "" {
			r.Header.Del(k)
		} else {
			r.Header.Set(k, v)
		}
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

const chatBody = `{"model":"anthropic/claude","messages":[{"role":"user","content":"hi"}]}`

func TestTokenRequired(t *testing.T) {
	h := Handler(&fakeBackend{})
	for _, auth := range []string{"", "Bearer wrong", "Basic abc"} {
		w := do(t, h, "POST", "/v1/chat/completions", chatBody, map[string]string{"Authorization": auth})
		if w.Code != 401 {
			t.Fatalf("auth %q: %d", auth, w.Code)
		}
	}
	if w := do(t, h, "GET", "/v1/models", "", map[string]string{"Authorization": ""}); w.Code != 401 {
		t.Fatalf("models without token: %d", w.Code)
	}
}

func TestBrowsersRefused(t *testing.T) {
	h := Handler(&fakeBackend{})
	for _, hdr := range []map[string]string{{"Origin": "https://evil.example"}, {"Origin": "null"}, {"Sec-Fetch-Site": "cross-site"}} {
		w := do(t, h, "POST", "/v1/chat/completions", chatBody, hdr)
		if w.Code != 403 || w.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Fatalf("%v: %d %v", hdr, w.Code, w.Header())
		}
	}
	w := do(t, h, "OPTIONS", "/v1/chat/completions", "", nil)
	if w.Header().Get("Access-Control-Allow-Origin") != "" || w.Code == 200 || w.Code == 204 {
		t.Fatalf("preflight answered: %d", w.Code)
	}
}

func TestDNSRebindingRefused(t *testing.T) {
	h := Handler(&fakeBackend{})
	for _, host := range []string{"evil.example:8765", "127.0.0.1.evil.example", "192.168.1.10:8765"} {
		r := httptest.NewRequest("GET", "http://x/v1/models", nil)
		r.Host = host
		r.Header.Set("Authorization", "Bearer "+good)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusMisdirectedRequest {
			t.Fatalf("host %q: %d", host, w.Code)
		}
	}
	for _, host := range []string{"localhost:1", "[::1]:2", "127.0.0.1"} {
		r := httptest.NewRequest("GET", "http://x/v1/models", nil)
		r.Host = host
		r.Header.Set("Authorization", "Bearer "+good)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Fatalf("host %q: %d", host, w.Code)
		}
	}
}

func TestModelsCarryLocality(t *testing.T) {
	w := do(t, Handler(&fakeBackend{}), "GET", "/v1/models", "", nil)
	var out struct {
		Data []struct {
			ID string `json:"id"`
			X  struct {
				Locality string `json:"locality"`
			} `json:"x_omnillm"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil || len(out.Data) != 2 {
		t.Fatal(w.Body.String())
	}
	if out.Data[0].X.Locality != "remote" || out.Data[1].X.Locality != "local" {
		t.Fatal(w.Body.String())
	}
}

func TestChatNonStream(t *testing.T) {
	b := &fakeBackend{}
	w := do(t, Handler(b), "POST", "/v1/chat/completions", chatBody, nil)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"content":"Hello"`) || !strings.Contains(w.Body.String(), `"total_tokens":5`) {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	if b.last.Caller != omnillm.Background {
		t.Fatal("local API calls must be Background")
	}
}

func TestChatStream(t *testing.T) {
	w := do(t, Handler(&fakeBackend{}), "POST", "/v1/chat/completions",
		`{"model":"anthropic/claude","stream":true,"messages":[{"role":"user","content":[{"type":"text","text":"hi"}]}]}`, nil)
	if w.Code != 200 || w.Header().Get("Content-Type") != "text/event-stream" {
		t.Fatalf("%d %v", w.Code, w.Header())
	}
	var text strings.Builder
	done := false
	sc := bufio.NewScanner(strings.NewReader(w.Body.String()))
	for sc.Scan() {
		l := sc.Text()
		if !strings.HasPrefix(l, "data: ") {
			continue
		}
		d := l[6:]
		if d == "[DONE]" {
			done = true
			continue
		}
		var ev struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		_ = json.Unmarshal([]byte(d), &ev)
		for _, c := range ev.Choices {
			text.WriteString(c.Delta.Content)
		}
	}
	if text.String() != "Hello" || !done {
		t.Fatalf("stream text %q done %v", text.String(), done)
	}
}

func TestStatusMapping(t *testing.T) {
	cases := []struct {
		err    error
		status int
		code   string
		retry  string
	}{
		{&omnillm.Error{Code: omnillm.CodeModelNotAllowed, Msg: omnillm.MsgModelNotAllowed}, 403, "model_not_allowed", ""},
		{&omnillm.Error{Code: omnillm.CodeBusy, Msg: omnillm.MsgBusyQueue, RetryAfter: 1500 * time.Millisecond}, 429, "busy", "2"},
		{&omnillm.Error{Code: omnillm.CodeBudgetExceeded, Msg: omnillm.MsgBudgetDevice, RetryAfter: time.Hour}, 429, "budget_exceeded", "3600"},
		{&omnillm.Error{Code: omnillm.CodeHubOffline, Msg: omnillm.MsgHubOffline}, 503, "hub_offline", ""},
		{&omnillm.Error{Code: omnillm.CodeUnsupportedFormat, Msg: omnillm.MsgUnsupportedFormat}, 400, "unsupported_response_format", ""},
		{errors.New("raw internal error with sk-SECRET0123456789"), 500, "internal", ""},
	}
	for _, c := range cases {
		for _, stream := range []bool{false, true} {
			body := chatBody
			if stream {
				body = strings.Replace(chatBody, `{"model"`, `{"stream":true,"model"`, 1)
			}
			w := do(t, Handler(&fakeBackend{err: c.err}), "POST", "/v1/chat/completions", body, map[string]string{"Accept-Language": "zh-CN"})
			if w.Code != c.status || !strings.Contains(w.Body.String(), `"code":"`+c.code+`"`) || w.Header().Get("Retry-After") != c.retry {
				t.Fatalf("%v stream=%v: %d %s retry=%q", c.err, stream, w.Code, w.Body, w.Header().Get("Retry-After"))
			}
			if strings.Contains(w.Body.String(), "SECRET") || strings.Contains(w.Body.String(), "raw internal") {
				t.Fatal("internal error text leaked")
			}
		}
	}
	w := do(t, Handler(&fakeBackend{err: omnillm.ErrModelNotAllowed}), "POST", "/v1/chat/completions", chatBody, map[string]string{"Accept-Language": "zh-CN"})
	if !strings.Contains(w.Body.String(), "不能用这个模型") {
		t.Fatalf("zh message expected: %s", w.Body)
	}
}

func TestMidStreamErrorEndsCleanly(t *testing.T) {
	w := do(t, Handler(&fakeBackend{mid: &omnillm.Error{Code: omnillm.CodeUpstream, Msg: omnillm.MsgUpstream}}), "POST", "/v1/chat/completions",
		strings.Replace(chatBody, `{"model"`, `{"stream":true,"model"`, 1), nil)
	if !strings.Contains(w.Body.String(), `"code":"upstream_error"`) || !strings.HasSuffix(strings.TrimSpace(w.Body.String()), "data: [DONE]") {
		t.Fatal(w.Body.String())
	}
}

func TestResponseFormatMapping(t *testing.T) {
	b := &fakeBackend{}
	do(t, Handler(b), "POST", "/v1/chat/completions",
		`{"model":"m/x","messages":[{"role":"user","content":"x"}],"response_format":{"type":"json_schema","json_schema":{"name":"v","schema":{"type":"object"}}}}`, nil)
	if string(b.last.JSONSchema) != `{"type":"object"}` || b.last.SchemaName != "v" {
		t.Fatalf("%+v", b.last)
	}
	do(t, Handler(b), "POST", "/v1/chat/completions", `{"model":"m/x","messages":[{"role":"user","content":"x"}],"response_format":{"type":"json_object"}}`, nil)
	if !b.last.JSONObject {
		t.Fatal("json_object")
	}
	w := do(t, Handler(b), "POST", "/v1/chat/completions", `{"model":"m/x","messages":[{"role":"user","content":"x"}],"response_format":{"type":"grammar"}}`, nil)
	if w.Code != 400 || !strings.Contains(w.Body.String(), "unsupported_response_format") {
		t.Fatal(w.Body.String())
	}
}

func TestBadBodies(t *testing.T) {
	h := Handler(&fakeBackend{})
	for _, body := range []string{"", "{", `{"messages":[{"role":"user","content":[{"type":"image_url"}]}]}`, strings.Repeat("x", maxBody+10)} {
		w := do(t, h, "POST", "/v1/chat/completions", body, nil)
		if w.Code != 400 {
			t.Fatalf("body %.30q: %d", body, w.Code)
		}
	}
}

func TestServeRefusesNonLoopback(t *testing.T) {
	if err := Serve(context.Background(), "0.0.0.0:0", Handler(&fakeBackend{})); err == nil {
		t.Fatal("listened on all interfaces")
	}
	if err := Serve(context.Background(), ":0", Handler(&fakeBackend{})); err == nil {
		t.Fatal("listened on all interfaces")
	}
}

func TestAnthropicMessages(t *testing.T) {
	b := &fakeBackend{}
	w := do(t, Handler(b), "POST", "/v1/messages", `{"model":"m/x","system":"be brief","max_tokens":10,"messages":[{"role":"user","content":"hi"}]}`, nil)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"text":"Hello"`) || b.last.Messages[0].Role != "system" {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
}
