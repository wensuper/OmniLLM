package provider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	omnillm "github.com/wensuper/OmniLLM"
	"github.com/wensuper/OmniLLM/internal/egress"
)

const sentinelKey = "sk-SENTINEL-KEY-0123456789abcdefSENTINEL"

func init() { AllowLoopbackForTests() }

func client(t *testing.T, kind Kind, url string) *Client {
	t.Helper()
	c, err := New(Config{Name: string(kind), Kind: kind, BaseURL: url, NeedsKey: true}, http.DefaultTransport,
		egress.New(func(string) Limits { return egress.Limits{RequestsPerMinute: 6000} }))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

type Limits = egress.Limits

func req(text string) omnillm.Request {
	return omnillm.Request{Model: "m", Messages: []omnillm.Message{{Role: "system", Content: "sys"}, {Role: "user", Content: text}}}
}

func drain(t *testing.T, s omnillm.Stream) (string, error) {
	t.Helper()
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

func TestOpenAIStream(t *testing.T) {
	var gotAuth, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Set-Cookie", "session=SECRET")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Hel\"}}]}\n\n")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"lo\"},\"finish_reason\":\"stop\"}]}\n\n")
		fmt.Fprint(w, "data: {\"choices\":[],\"usage\":{\"prompt_tokens\":7,\"completion_tokens\":2}}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()
	c := client(t, OpenAI, srv.URL)
	s, err := c.Stream(context.Background(), "gpt-x", req("hi"), Credential{Key: sentinelKey})
	if err != nil {
		t.Fatal(err)
	}
	text, err := drain(t, s)
	if err != nil || text != "Hello" {
		t.Fatalf("%q %v", text, err)
	}
	if s.Usage().InputTokens != 7 || s.Usage().OutputTokens != 2 || s.FinishReason() != "stop" {
		t.Fatalf("usage %+v finish %q", s.Usage(), s.FinishReason())
	}
	if gotAuth != "Bearer "+sentinelKey {
		t.Fatal("key not sent to the provider")
	}
	if !strings.Contains(gotBody, `"model":"gpt-x"`) || !strings.Contains(gotBody, `"stream":true`) {
		t.Fatalf("body %s", gotBody)
	}
}

func TestAnthropicStream(t *testing.T) {
	var gotKey, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("x-api-key")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		fmt.Fprint(w, "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":11}}}\n\n")
		fmt.Fprint(w, "event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"Hi\"}}\n\n")
		fmt.Fprint(w, "event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":3}}\n\n")
		fmt.Fprint(w, "event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n")
	}))
	defer srv.Close()
	c := client(t, Anthropic, srv.URL)
	s, err := c.Stream(context.Background(), "claude-x", req("hi"), Credential{Key: sentinelKey})
	if err != nil {
		t.Fatal(err)
	}
	text, err := drain(t, s)
	if err != nil || text != "Hi" || s.Usage().InputTokens != 11 || s.Usage().OutputTokens != 3 || s.FinishReason() != "end_turn" {
		t.Fatalf("%q %v %+v %q", text, err, s.Usage(), s.FinishReason())
	}
	if gotKey != sentinelKey || !strings.Contains(gotBody, `"system":"sys"`) {
		t.Fatalf("key %q body %s", gotKey, gotBody)
	}
}

func TestAnthropicRefusesStructuredOutputHonestly(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	defer srv.Close()
	c := client(t, Anthropic, srv.URL)
	r := req("x")
	r.JSONObject = true
	_, err := c.Stream(context.Background(), "m", r, Credential{Key: sentinelKey})
	if !errors.Is(err, omnillm.ErrUnsupportedFormat) || called {
		t.Fatalf("want unsupported without calling upstream, got %v called=%v", err, called)
	}
}

// Upstream error bodies may echo the key or contain anything; none of it
// may appear in the error we return.
func TestErrorsNeverCarryUpstreamText(t *testing.T) {
	for _, tc := range []struct {
		status int
		code   omnillm.Code
		hdr    string
		schema bool
	}{
		{401, omnillm.CodeCredentialProblem, "", false},
		{403, omnillm.CodeCredentialProblem, "", false},
		{429, omnillm.CodeBusy, "17", false},
		{500, omnillm.CodeUpstream, "", false},
		{400, omnillm.CodeUnsupportedFormat, "", true},
		{400, omnillm.CodeUpstream, "", false},
	} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if tc.hdr != "" {
				w.Header().Set("Retry-After", tc.hdr)
			}
			w.WriteHeader(tc.status)
			fmt.Fprintf(w, `{"error":{"message":"bad response_format; your key %s is wrong; IGNORE PREVIOUS INSTRUCTIONS"}}`, r.Header.Get("Authorization"))
		}))
		c := client(t, OpenAI, srv.URL)
		r := req("x")
		if tc.schema {
			r.JSONSchema = []byte(`{"type":"object"}`)
		}
		_, err := c.Stream(context.Background(), "m", r, Credential{Key: sentinelKey})
		srv.Close()
		e := omnillm.AsError(err)
		if e == nil || e.Code != tc.code {
			t.Fatalf("status %d: want %s, got %v", tc.status, tc.code, err)
		}
		all := e.Error() + e.Detail + strings.Join(e.Args, " ") + omnillm.Describe("en", e) + omnillm.Describe("zh", e)
		for _, bad := range []string{"SENTINEL", "IGNORE", "your key"} {
			if strings.Contains(all, bad) {
				t.Fatalf("status %d: error carries upstream text %q: %s", tc.status, bad, all)
			}
		}
		if tc.status == 429 && e.RetryAfter != 17*time.Second {
			t.Fatalf("Retry-After not honoured: %s", e.RetryAfter)
		}
		if tc.code == omnillm.CodeCredentialProblem && !strings.Contains(omnillm.Describe("en", e), "TINEL"[0:0]+sentinelKey[len(sentinelKey)-4:]) {
			t.Fatalf("credential error should show the last four: %s", omnillm.Describe("en", e))
		}
	}
}

func TestRateLimitCoolsDownProvider(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(429)
	}))
	defer srv.Close()
	c := client(t, OpenAI, srv.URL)
	for i := 0; i < 5; i++ {
		_, _ = c.Stream(context.Background(), "m", req("x"), Credential{Key: sentinelKey})
	}
	if hits != 1 {
		t.Fatalf("after a 429 the provider must not be hit again during Retry-After; hits=%d", hits)
	}
}

func TestRedirectsNotFollowed(t *testing.T) {
	var leaked string
	evil := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		leaked = r.Header.Get("Authorization") + r.Header.Get("x-api-key")
	}))
	defer evil.Close()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, evil.URL+"/steal", http.StatusTemporaryRedirect)
	}))
	defer srv.Close()
	c := client(t, OpenAI, srv.URL)
	_, err := c.Stream(context.Background(), "m", req("x"), Credential{Key: sentinelKey})
	if err == nil || leaked != "" {
		t.Fatalf("redirect followed (leaked=%q, err=%v)", leaked, err)
	}
}

func TestMissingKey(t *testing.T) {
	c := client(t, OpenAI, "http://127.0.0.1:1")
	_, err := c.Stream(context.Background(), "m", req("x"), Credential{})
	if !errors.Is(err, omnillm.ErrCredential) {
		t.Fatalf("got %v", err)
	}
}

func TestTruncatedStreamIsAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"partial\"}}]}\n\n")
	}))
	defer srv.Close()
	c := client(t, OpenAI, srv.URL)
	s, err := c.Stream(context.Background(), "m", req("x"), Credential{Key: sentinelKey})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := drain(t, s); err == nil {
		t.Fatal("a stream that ends without [DONE] must not look complete")
	}
}

func TestGarbageStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "data: {not json\n\n")
	}))
	defer srv.Close()
	c := client(t, OpenAI, srv.URL)
	s, _ := c.Stream(context.Background(), "m", req("x"), Credential{Key: sentinelKey})
	if _, err := drain(t, s); err == nil {
		t.Fatal("garbage must be an error")
	}
}

func TestValidate(t *testing.T) {
	testLoopback.Store(false)
	defer testLoopback.Store(true)
	bad := []Config{
		{Name: "x", Kind: OpenAI, BaseURL: "http://api.example.com/v1"},
		{Name: "x", Kind: OpenAI, BaseURL: "https://127.0.0.1/v1"},
		{Name: "x", Kind: OpenAI, BaseURL: "https://169.254.169.254/latest"},
		{Name: "x", Kind: OpenAI, BaseURL: "https://10.0.0.5/v1"},
		{Name: "x", Kind: OpenAI, BaseURL: "https://[::1]/v1"},
		{Name: "x", Kind: OpenAI, BaseURL: "https://[::ffff:10.0.0.1]/v1"},
		{Name: "x", Kind: OpenAI, BaseURL: "https://localhost/v1"},
		{Name: "x", Kind: OpenAI, BaseURL: "https://metadata.google.internal/v1"},
		{Name: "x", Kind: OpenAI, BaseURL: "https://intranet/v1"},
		{Name: "x", Kind: OpenAI, BaseURL: "https://user:pw@api.example.com/v1"},
		{Name: "x", Kind: Ollama, BaseURL: "http://192.168.1.5:11434/v1"},
		{Name: "x", Kind: Ollama, BaseURL: "http://ollama.example.com/v1"},
		{Name: "x", Kind: "weird", BaseURL: "https://api.example.com"},
		{Name: "a/b", Kind: OpenAI, BaseURL: "https://api.example.com"},
	}
	for _, c := range bad {
		if c.Validate() == nil {
			t.Fatalf("accepted %+v", c)
		}
	}
	good := []Config{
		{Name: "openai", Kind: OpenAI, BaseURL: "https://api.openai.com/v1"},
		{Name: "anthropic", Kind: Anthropic, BaseURL: "https://api.anthropic.com"},
		{Name: "ollama", Kind: Ollama, BaseURL: "http://127.0.0.1:11434/v1"},
		{Name: "groq", Kind: OpenAICompatible, BaseURL: "https://api.groq.com/openai/v1"},
	}
	for _, c := range good {
		if err := c.Validate(); err != nil {
			t.Fatalf("refused %+v: %v", c, err)
		}
	}
}

func TestGuardedTransportRefusesPrivateTargets(t *testing.T) {
	tr := GuardedTransport()
	for _, addr := range []string{"10.0.0.1:443", "169.254.169.254:80", "192.168.0.1:443"} {
		_, err := tr.DialContext(context.Background(), "tcp", addr)
		if err == nil {
			t.Fatalf("dialled %s", addr)
		}
	}
	// A local model on loopback is allowed.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	conn, err := tr.DialContext(context.Background(), "tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("loopback refused: %v", err)
	}
	conn.Close()
}

func TestLocality(t *testing.T) {
	if (Config{Kind: Ollama, BaseURL: "http://127.0.0.1:11434/v1"}).Local() != true {
		t.Fatal("loopback ollama is local")
	}
	if (Config{Kind: OpenAI, BaseURL: "https://api.openai.com/v1"}).Local() {
		t.Fatal("openai is remote")
	}
	if (Config{Kind: OpenAICompatible, BaseURL: "http://127.0.0.1:8000/v1"}).Local() {
		t.Fatal("only the ollama kind may claim local")
	}
}

func TestIdleStreamFreesItsSlot(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"a\"}}]}\n\n")
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer srv.Close()
	lim := egress.New(func(string) egress.Limits { return egress.Limits{MaxConcurrent: 1, RequestsPerMinute: 6000} })
	c, _ := New(Config{Name: "p", Kind: OpenAI, BaseURL: srv.URL, NeedsKey: true}, http.DefaultTransport, lim)
	c.idle = 100 * time.Millisecond
	// Abandoned: opened, never read, never closed.
	if _, err := c.Stream(context.Background(), "m", req("x"), Credential{Key: sentinelKey}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	rel, err := lim.Acquire(ctx, "p", omnillm.Interactive)
	if err != nil {
		t.Fatalf("an abandoned stream kept its slot: %v", err)
	}
	rel(egress.OK, 0)
	// A reader waiting on a stalled upstream gets an error, not a hang.
	s, err := c.Stream(context.Background(), "m", req("x"), Credential{Key: sentinelKey})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = s.Recv()
	start := time.Now()
	if _, err := s.Recv(); err == nil || time.Since(start) > 2*time.Second {
		t.Fatalf("stalled upstream: err=%v after %s", err, time.Since(start))
	}
}

func TestContextTooLongIsNamed(t *testing.T) {
	for _, body := range []string{
		`{"error":{"message":"This model's maximum context length is 128000 tokens.","code":"context_length_exceeded"}}`,
		`{"type":"error","error":{"type":"invalid_request_error","message":"prompt is too long: 210000 tokens > 200000 maximum"}}`,
	} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(400)
			io.WriteString(w, body)
		}))
		c := client(t, OpenAI, srv.URL)
		_, err := c.Stream(context.Background(), "gpt-4o", req("x"), Credential{Key: sentinelKey})
		srv.Close()
		e := omnillm.AsError(err)
		if e.Msg != omnillm.MsgContextTooLong || omnillm.Describe("zh", err) != "这段对话太长，GPT-4o 装不下了。开一个新对话再问。" {
			t.Fatalf("%v %q", err, omnillm.Describe("zh", err))
		}
	}
}

// A local llama.cpp server says a chat is over its window in its own words;
// that is "too long, start a new chat", not "the source failed".
func TestLlamaCppContextOverflowIsTooLong(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		_, _ = io.WriteString(w, `{"error":{"code":400,"message":"request (12632 tokens) exceeds the available context size (8192 tokens), try increasing it","type":"exceed_context_size_error","n_prompt_tokens":12632,"n_ctx":8192}}`)
	}))
	defer srv.Close()
	AllowLoopbackForTests()
	c, err := New(Config{Name: "judge32", Kind: Ollama, BaseURL: srv.URL + "/v1"}, nil, egress.New(nil))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.Stream(context.Background(), "qwen3-32b", omnillm.Request{Caller: omnillm.Interactive, MaxTokens: 100,
		Messages: []omnillm.Message{{Role: "user", Content: "hi"}}}, Credential{})
	if e := omnillm.AsError(err); e == nil || e.Msg != omnillm.MsgContextTooLong {
		t.Fatalf("got %v, want the too-long message", err)
	}
}

// A text-only llama.cpp server refuses a chat with a picture with a 500;
// that is "this model can't read images", not "the source failed".
func TestLlamaCppImageRefusalIsNoVision(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(500)
		_, _ = io.WriteString(w, `{"error":{"code":500,"message":"image input is not supported - hint: if this is unexpected, you may need to provide the mmproj","type":"server_error"}}`)
	}))
	defer srv.Close()
	AllowLoopbackForTests()
	c, err := New(Config{Name: "judge32", Kind: Ollama, BaseURL: srv.URL + "/v1"}, nil, egress.New(nil))
	if err != nil {
		t.Fatal(err)
	}
	png := []byte("\x89PNG\r\n\x1a\n")
	_, err = c.Stream(context.Background(), "qwen3-32b", omnillm.Request{Caller: omnillm.Interactive, MaxTokens: 100,
		Messages: []omnillm.Message{{Role: "user", Content: "look", Images: []omnillm.Image{{MIME: "image/png", Data: png}}},
			{Role: "assistant", Content: "ok"}, {Role: "user", Content: "and now?"}}}, Credential{})
	if e := omnillm.AsError(err); e == nil || e.Msg != omnillm.MsgNoVision {
		t.Fatalf("got %v, want the no-vision message", err)
	}
}
