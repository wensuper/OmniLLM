package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	omnillm "github.com/wensuper/OmniLLM"
	"github.com/wensuper/OmniLLM/internal/egress"
	"github.com/wensuper/OmniLLM/internal/redact"
)

type parseFn func(s *stream, event, data string) (string, bool, error)

func badStream() error {
	return &omnillm.Error{Code: omnillm.CodeUpstream, Msg: omnillm.MsgUpstream, Detail: "unparseable stream"}
}

func (c *Client) build(ctx context.Context, model string, req omnillm.Request, cred Credential) (*http.Request, parseFn, error) {
	switch c.cfg.Kind {
	case Anthropic:
		return c.buildAnthropic(ctx, model, req, cred)
	default:
		return c.buildOpenAI(ctx, model, req, cred)
	}
}

// ---- OpenAI-compatible (OpenAI, Ollama, others) ----

type oaMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type oaReq struct {
	Model          string          `json:"model"`
	Messages       []oaMsg         `json:"messages"`
	Stream         bool            `json:"stream"`
	StreamOptions  map[string]bool `json:"stream_options,omitempty"`
	MaxTokens      int             `json:"max_tokens,omitempty"`
	Temperature    *float64        `json:"temperature,omitempty"`
	ResponseFormat any             `json:"response_format,omitempty"`
}

func (c *Client) buildOpenAI(ctx context.Context, model string, req omnillm.Request, cred Credential) (*http.Request, parseFn, error) {
	body := oaReq{Model: model, Stream: true, StreamOptions: map[string]bool{"include_usage": true},
		MaxTokens: req.MaxTokens, Temperature: req.Temperature}
	for _, m := range req.Messages {
		body.Messages = append(body.Messages, oaMsg{Role: m.Role, Content: m.Content})
	}
	switch {
	case len(req.JSONSchema) > 0:
		name := req.SchemaName
		if name == "" {
			name = "result"
		}
		body.ResponseFormat = map[string]any{"type": "json_schema", "json_schema": map[string]any{
			"name": name, "schema": json.RawMessage(req.JSONSchema), "strict": true}}
	case req.JSONObject:
		body.ResponseFormat = map[string]string{"type": "json_object"}
	}
	b, err := json.Marshal(body)
	if err != nil {
		return nil, nil, err
	}
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.cfg.BaseURL, "/")+"/chat/completions", bytes.NewReader(b))
	if err != nil {
		return nil, nil, err
	}
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "text/event-stream")
	if cred.Key != "" {
		r.Header.Set("Authorization", "Bearer "+cred.Key)
	}
	return r, parseOpenAI, nil
}

func parseOpenAI(s *stream, _ string, data string) (string, bool, error) {
	if data == "[DONE]" {
		return "", true, nil
	}
	var ev struct {
		Choices []struct {
			Delta struct {
				Content string `json:"content"`
			} `json:"delta"`
			FinishReason *string `json:"finish_reason"`
		} `json:"choices"`
		Usage *struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
		Error any `json:"error"`
	}
	if err := json.Unmarshal([]byte(data), &ev); err != nil {
		return "", false, badStream()
	}
	if ev.Error != nil {
		return "", false, &omnillm.Error{Code: omnillm.CodeUpstream, Msg: omnillm.MsgUpstream, Detail: "error event in stream"}
	}
	if ev.Usage != nil {
		s.setUsage(ev.Usage.PromptTokens, ev.Usage.CompletionTokens)
	}
	text := ""
	for _, ch := range ev.Choices {
		text += ch.Delta.Content
		if ch.FinishReason != nil && *ch.FinishReason != "" {
			s.setFinish(*ch.FinishReason)
		}
	}
	return text, false, nil
}

// ---- Anthropic Messages ----

type anReq struct {
	Model       string   `json:"model"`
	System      string   `json:"system,omitempty"`
	Messages    []oaMsg  `json:"messages"`
	MaxTokens   int      `json:"max_tokens"`
	Temperature *float64 `json:"temperature,omitempty"`
	Stream      bool     `json:"stream"`
}

func (c *Client) buildAnthropic(ctx context.Context, model string, req omnillm.Request, cred Credential) (*http.Request, parseFn, error) {
	// Structured output is not mapped to Anthropic yet: say so instead of
	// pretending, so the caller steps down (design §8.2).
	if len(req.JSONSchema) > 0 || req.JSONObject {
		return nil, nil, &omnillm.Error{Code: omnillm.CodeUnsupportedFormat, Msg: omnillm.MsgUnsupportedFormat}
	}
	body := anReq{Model: model, MaxTokens: req.MaxTokens, Temperature: req.Temperature, Stream: true}
	if body.MaxTokens == 0 {
		body.MaxTokens = 4096
	}
	var sys []string
	for _, m := range req.Messages {
		if m.Role == "system" {
			sys = append(sys, m.Content)
			continue
		}
		body.Messages = append(body.Messages, oaMsg{Role: m.Role, Content: m.Content})
	}
	body.System = strings.Join(sys, "\n\n")
	if len(body.Messages) == 0 {
		return nil, nil, &omnillm.Error{Code: omnillm.CodeBadRequest, Msg: "bad_messages"}
	}
	b, err := json.Marshal(body)
	if err != nil {
		return nil, nil, err
	}
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.cfg.BaseURL, "/")+"/v1/messages", bytes.NewReader(b))
	if err != nil {
		return nil, nil, err
	}
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "text/event-stream")
	r.Header.Set("anthropic-version", "2023-06-01")
	r.Header.Set("x-api-key", cred.Key)
	return r, parseAnthropic, nil
}

func parseAnthropic(s *stream, event, data string) (string, bool, error) {
	var ev struct {
		Type    string `json:"type"`
		Message struct {
			Usage struct {
				InputTokens int `json:"input_tokens"`
			} `json:"usage"`
		} `json:"message"`
		Delta struct {
			Type       string `json:"type"`
			Text       string `json:"text"`
			StopReason string `json:"stop_reason"`
		} `json:"delta"`
		Usage struct {
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal([]byte(data), &ev); err != nil {
		return "", false, badStream()
	}
	switch ev.Type {
	case "message_start":
		s.setUsage(ev.Message.Usage.InputTokens, 0)
	case "content_block_delta":
		if ev.Delta.Type == "text_delta" {
			return ev.Delta.Text, false, nil
		}
	case "message_delta":
		if ev.Delta.StopReason != "" {
			s.setFinish(ev.Delta.StopReason)
		}
		s.setUsage(0, ev.Usage.OutputTokens)
	case "message_stop":
		return "", true, nil
	case "error":
		return "", false, &omnillm.Error{Code: omnillm.CodeUpstream, Msg: omnillm.MsgUpstream, Detail: "error event in stream"}
	}
	return "", false, nil
}

// ListModels asks the provider which models this key can use.
func (c *Client) ListModels(ctx context.Context, cred Credential) ([]string, error) {
	if c.cfg.Kind.Subscription() {
		if !CLILoggedIn(ctx, c.cfg.Kind, c.cfg.Path) {
			return nil, &omnillm.Error{Code: omnillm.CodeCredentialProblem, Msg: omnillm.MsgCLILogin, Args: []string{DisplayName(c.cfg.Kind), cliName(c.cfg.Kind)}}
		}
		return append([]string(nil), cliModels[c.cfg.Kind]...), nil
	}
	if c.cfg.NeedsKey && cred.Key == "" {
		return nil, &omnillm.Error{Code: omnillm.CodeCredentialProblem, Msg: omnillm.MsgCredentialMissing, Args: []string{c.cfg.Name}}
	}
	release, err := c.lim.Acquire(ctx, c.cfg.Name, omnillm.Interactive)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	u := strings.TrimRight(c.cfg.BaseURL, "/") + "/models"
	if c.cfg.Kind == Anthropic {
		u = strings.TrimRight(c.cfg.BaseURL, "/") + "/v1/models?limit=100"
	}
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		release(egress.OK, 0)
		return nil, err
	}
	if c.cfg.Kind == Anthropic {
		r.Header.Set("x-api-key", cred.Key)
		r.Header.Set("anthropic-version", "2023-06-01")
	} else if cred.Key != "" {
		r.Header.Set("Authorization", "Bearer "+cred.Key)
	}
	resp, err := c.http.Do(r)
	if err != nil {
		release(egress.Failed, 0)
		return nil, &omnillm.Error{Code: omnillm.CodeUpstream, Msg: omnillm.MsgUpstream, Detail: redact.String(err.Error())}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		e, o, w := c.classify(resp, cred, omnillm.Request{})
		release(o, w)
		return nil, e
	}
	release(egress.OK, 0)
	var out struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&out); err != nil {
		return nil, badStream()
	}
	var ids []string
	for _, d := range out.Data {
		if d.ID != "" && len(d.ID) <= 150 && !strings.ContainsAny(d.ID, " \t\n") {
			ids = append(ids, d.ID)
		}
	}
	return ids, nil
}
