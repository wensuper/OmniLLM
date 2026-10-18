// Package localapi is the OpenAI-compatible HTTP interface on 127.0.0.1 for
// programs that cannot embed the library (design §6). It runs on the hub
// (app tokens) and inside "omnillm connect" on a device (a local token).
//
// Anything on this machine, and any web page in a browser here, can reach a
// loopback port, so: every request needs a token; the Host header must be a
// loopback name (DNS rebinding); requests carrying an Origin header (from a
// browser) are refused and no CORS headers are ever sent; bodies are capped.
package localapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	omnillm "github.com/wensuper/OmniLLM"
)

// Backend serves requests for a bearer token.
type Backend interface {
	Models(ctx context.Context, token string) ([]omnillm.ModelInfo, error)
	Complete(ctx context.Context, token string, req omnillm.Request) (omnillm.Stream, error)
}

const maxBody = 8 << 20

// Handler returns the HTTP handler.
func Handler(b Backend) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/models", func(w http.ResponseWriter, r *http.Request) { models(w, r, b) })
	mux.HandleFunc("POST /v1/chat/completions", func(w http.ResponseWriter, r *http.Request) { chat(w, r, b) })
	mux.HandleFunc("POST /v1/messages", func(w http.ResponseWriter, r *http.Request) { messages(w, r, b) })
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if r.Header.Get("Origin") != "" || r.Header.Get("Sec-Fetch-Site") == "cross-site" {
			writeErr(w, r, &omnillm.Error{Code: omnillm.CodeForbiddenOrigin, Msg: omnillm.MsgBrowserRefused}, http.StatusForbidden)
			return
		}
		if !loopbackHost(r.Host) {
			writeErr(w, r, &omnillm.Error{Code: omnillm.CodeBadRequest, Msg: omnillm.MsgWrongHost}, http.StatusMisdirectedRequest)
			return
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxBody)
		mux.ServeHTTP(w, r)
	})
}

func loopbackHost(hostport string) bool {
	host := hostport
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// Serve listens on a loopback address only and serves until ctx ends.
func Serve(ctx context.Context, addr string, h http.Handler) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	if !loopbackHost(host) {
		return errors.New("localapi: refusing to listen beyond this machine; use 127.0.0.1")
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	srv := &http.Server{Handler: h, ReadHeaderTimeout: 10 * time.Second, ReadTimeout: time.Minute,
		IdleTimeout: 2 * time.Minute, MaxHeaderBytes: 32 << 10}
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		select {
		case <-ctx.Done():
		case <-stop:
			return
		}
		sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(sctx)
	}()
	err = srv.Serve(ln)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func token(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return strings.TrimSpace(r.Header.Get("x-api-key"))
}

func lang(r *http.Request) string {
	if strings.HasPrefix(strings.ToLower(r.Header.Get("Accept-Language")), "zh") {
		return "zh"
	}
	return "en"
}

type apiError struct {
	Error struct {
		Code    omnillm.Code `json:"code"`
		Type    string       `json:"type"`
		Message string       `json:"message"`
	} `json:"error"`
}

func errBody(r *http.Request, e *omnillm.Error) apiError {
	var b apiError
	b.Error.Code = e.Code
	b.Error.Type = string(e.Code)
	b.Error.Message = omnillm.Describe(lang(r), e)
	return b
}

func writeErr(w http.ResponseWriter, r *http.Request, err error, status int) {
	e := omnillm.AsError(err)
	if status == 0 {
		status = omnillm.HTTPStatus(e.Code)
	}
	if e.RetryAfter > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(int((e.RetryAfter+time.Second-1)/time.Second)))
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errBody(r, e))
}

func models(w http.ResponseWriter, r *http.Request, b Backend) {
	ms, err := b.Models(r.Context(), token(r))
	if err != nil {
		writeErr(w, r, err, 0)
		return
	}
	type xo struct {
		Locality    omnillm.Locality `json:"locality"`
		Provider    string           `json:"provider"`
		DisplayName string           `json:"display_name"`
	}
	type item struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		OwnedBy string `json:"owned_by"`
		X       xo     `json:"x_omnillm"`
	}
	out := struct {
		Object string `json:"object"`
		Data   []item `json:"data"`
	}{Object: "list", Data: []item{}}
	for _, m := range ms {
		out.Data = append(out.Data, item{ID: m.ID, Object: "model", OwnedBy: m.Provider, X: xo{m.Locality, m.Provider, m.DisplayName}})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

// content accepts a string or an array of text parts.
func content(raw json.RawMessage) (string, error) {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s, nil
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &parts); err != nil {
		return "", errors.New("content")
	}
	var b strings.Builder
	for _, p := range parts {
		if p.Type != "text" {
			return "", errors.New("only text content is supported")
		}
		b.WriteString(p.Text)
	}
	return b.String(), nil
}

type chatReq struct {
	Model    string `json:"model"`
	Messages []struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	} `json:"messages"`
	MaxTokens           int      `json:"max_tokens"`
	MaxCompletionTokens int      `json:"max_completion_tokens"`
	Temperature         *float64 `json:"temperature"`
	Stream              bool     `json:"stream"`
	ResponseFormat      *struct {
		Type       string `json:"type"`
		JSONSchema *struct {
			Name   string          `json:"name"`
			Schema json.RawMessage `json:"schema"`
		} `json:"json_schema"`
	} `json:"response_format"`
}

func badRequest() error { return &omnillm.Error{Code: omnillm.CodeBadRequest, Msg: "bad_messages"} }

func decode(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(v); err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			return &omnillm.Error{Code: omnillm.CodeBadRequest, Msg: "too_large"}
		}
		return badRequest()
	}
	return nil
}

func chat(w http.ResponseWriter, r *http.Request, b Backend) {
	var cr chatReq
	if err := decode(r, &cr); err != nil {
		writeErr(w, r, err, 0)
		return
	}
	req := omnillm.Request{Model: cr.Model, MaxTokens: cr.MaxTokens, Temperature: cr.Temperature, Caller: omnillm.Background}
	if req.MaxTokens == 0 {
		req.MaxTokens = cr.MaxCompletionTokens
	}
	for _, m := range cr.Messages {
		role := m.Role
		if role == "developer" {
			role = "system"
		}
		c, err := content(m.Content)
		if err != nil {
			writeErr(w, r, badRequest(), 0)
			return
		}
		req.Messages = append(req.Messages, omnillm.Message{Role: role, Content: c})
	}
	if rf := cr.ResponseFormat; rf != nil {
		switch rf.Type {
		case "json_schema":
			if rf.JSONSchema == nil || len(rf.JSONSchema.Schema) == 0 {
				writeErr(w, r, badRequest(), 0)
				return
			}
			req.JSONSchema, req.SchemaName = rf.JSONSchema.Schema, rf.JSONSchema.Name
		case "json_object":
			req.JSONObject = true
		case "text", "":
		default:
			writeErr(w, r, &omnillm.Error{Code: omnillm.CodeUnsupportedFormat, Msg: omnillm.MsgUnsupportedFormat}, 0)
			return
		}
	}
	s, err := b.Complete(r.Context(), token(r), req)
	if err != nil {
		writeErr(w, r, err, 0)
		return
	}
	defer s.Close()
	id := fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano())
	created := time.Now().Unix()
	if !cr.Stream {
		var text strings.Builder
		for {
			c, err := s.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				writeErr(w, r, err, 0)
				return
			}
			text.WriteString(c.Text)
		}
		u := s.Usage()
		finish := s.FinishReason()
		if finish == "" || finish == "end_turn" {
			finish = "stop"
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": id, "object": "chat.completion", "created": created, "model": req.Model,
			"choices": []any{map[string]any{"index": 0, "message": map[string]string{"role": "assistant", "content": text.String()}, "finish_reason": finish}},
			"usage":   map[string]int{"prompt_tokens": u.InputTokens, "completion_tokens": u.OutputTokens, "total_tokens": u.InputTokens + u.OutputTokens},
		})
		return
	}
	// Read the first chunk before committing to 200, so an immediate error
	// still gets its proper status code.
	first, ferr := s.Recv()
	if ferr != nil && !errors.Is(ferr, io.EOF) {
		writeErr(w, r, ferr, 0)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	fl, _ := w.(http.Flusher)
	send := func(v any) {
		bb, _ := json.Marshal(v)
		fmt.Fprintf(w, "data: %s\n\n", bb)
		if fl != nil {
			fl.Flush()
		}
	}
	chunk := func(text string, finish any) map[string]any {
		return map[string]any{"id": id, "object": "chat.completion.chunk", "created": created, "model": req.Model,
			"choices": []any{map[string]any{"index": 0, "delta": map[string]string{"content": text}, "finish_reason": finish}}}
	}
	if first.Text != "" {
		send(chunk(first.Text, nil))
	}
	for ferr == nil {
		c, err := s.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			send(errBody(r, omnillm.AsError(err)))
			fmt.Fprint(w, "data: [DONE]\n\n")
			return
		}
		send(chunk(c.Text, nil))
	}
	u := s.Usage()
	end := chunk("", "stop")
	end["usage"] = map[string]int{"prompt_tokens": u.InputTokens, "completion_tokens": u.OutputTokens, "total_tokens": u.InputTokens + u.OutputTokens}
	send(end)
	fmt.Fprint(w, "data: [DONE]\n\n")
	if fl != nil {
		fl.Flush()
	}
}

// messages is a minimal Anthropic-compatible endpoint (text only).
func messages(w http.ResponseWriter, r *http.Request, b Backend) {
	var mr struct {
		Model       string          `json:"model"`
		System      json.RawMessage `json:"system"`
		MaxTokens   int             `json:"max_tokens"`
		Temperature *float64        `json:"temperature"`
		Stream      bool            `json:"stream"`
		Messages    []struct {
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
		} `json:"messages"`
	}
	if err := decode(r, &mr); err != nil {
		writeErr(w, r, err, 0)
		return
	}
	if mr.Stream {
		writeErr(w, r, &omnillm.Error{Code: omnillm.CodeUnsupportedFormat, Msg: omnillm.MsgUnsupportedFormat}, 0)
		return
	}
	req := omnillm.Request{Model: mr.Model, MaxTokens: mr.MaxTokens, Temperature: mr.Temperature, Caller: omnillm.Background}
	if len(mr.System) > 0 {
		s, err := content(mr.System)
		if err != nil {
			writeErr(w, r, badRequest(), 0)
			return
		}
		req.Messages = append(req.Messages, omnillm.Message{Role: "system", Content: s})
	}
	for _, m := range mr.Messages {
		c, err := content(m.Content)
		if err != nil {
			writeErr(w, r, badRequest(), 0)
			return
		}
		req.Messages = append(req.Messages, omnillm.Message{Role: m.Role, Content: c})
	}
	s, err := b.Complete(r.Context(), token(r), req)
	if err != nil {
		writeErr(w, r, err, 0)
		return
	}
	defer s.Close()
	var text strings.Builder
	for {
		c, err := s.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			writeErr(w, r, err, 0)
			return
		}
		text.WriteString(c.Text)
	}
	u := s.Usage()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id": fmt.Sprintf("msg_%d", time.Now().UnixNano()), "type": "message", "role": "assistant", "model": req.Model,
		"content":     []any{map[string]string{"type": "text", "text": text.String()}},
		"stop_reason": "end_turn",
		"usage":       map[string]int{"input_tokens": u.InputTokens, "output_tokens": u.OutputTokens},
	})
}
