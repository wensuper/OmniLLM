// Package omnillm holds the types shared by the hub, device and local API
// packages: requests, streamed chunks, model descriptions and the typed
// errors every caller can branch on.
//
// OmniLLM is a library first (docs/principles.md, principle 7). Nothing in
// this module reads configuration files, opens ports, writes to stdout or
// keeps global state on its own; the host passes everything in.
package omnillm

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// Caller says who is asking. Background callers (automated programs such as
// NEXA's judging, and every app token) may only use official API keys and
// local models, and queue behind people (principle 1, "自动化调用").
type Caller int

const (
	// Background is the default: an automated program.
	Background Caller = iota
	// Interactive is a person in a conversation right now.
	Interactive
)

func (c Caller) String() string {
	if c == Interactive {
		return "interactive"
	}
	return "background"
}

// Message is one turn of a conversation. A person's message may carry
// images (a photo, a screenshot).
type Message struct {
	Role    string  `json:"role"` // system | user | assistant
	Content string  `json:"content"`
	Images  []Image `json:"images,omitempty"`
}

// Image is one picture in a message: PNG, JPEG, GIF or WebP bytes.
type Image struct {
	MIME string `json:"mime"`
	Data []byte `json:"data"` // base64 in JSON
}

// Request is a model call.
type Request struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature *float64  `json:"temperature,omitempty"`
	// JSONSchema asks for structured output matching this schema.
	JSONSchema json.RawMessage `json:"json_schema,omitempty"`
	SchemaName string          `json:"schema_name,omitempty"`
	// JSONObject asks for any JSON object (ignored when JSONSchema is set).
	JSONObject bool   `json:"json_object,omitempty"`
	Caller     Caller `json:"caller"`
}

// Limits on a request, enforced before anything leaves the hub.
const (
	MaxMessages      = 512
	MaxRequestBytes  = 5 << 20 // 5 MiB of text and images in total (base64 still fits one 8 MiB message)
	MaxImageBytes    = 2 << 20 // one image
	MaxImagesPerMsg  = 4
	MaxImageSide     = 8192     // pixels, either side
	MaxImagePixels   = 40 << 20 // about 40 megapixels
	MaxImages        = 8        // in one request
	MaxOutputTokens  = 64000
	MaxModelIDLength = 200
)

// Validate checks the request shape. It never looks at credentials.
func (r *Request) Validate() error {
	if r.Model == "" || len(r.Model) > MaxModelIDLength {
		return &Error{Code: CodeBadRequest, Msg: msgBadModel}
	}
	if len(r.Messages) == 0 || len(r.Messages) > MaxMessages {
		return &Error{Code: CodeBadRequest, Msg: msgBadMessages}
	}
	total, images := 0, 0
	for _, m := range r.Messages {
		switch m.Role {
		case "system", "user", "assistant":
		default:
			return &Error{Code: CodeBadRequest, Msg: msgBadRole}
		}
		total += len(m.Content)
		if len(m.Images) > 0 && (m.Role != "user" || len(m.Images) > MaxImagesPerMsg) {
			return &Error{Code: CodeBadRequest, Msg: MsgBadImage}
		}
		for _, im := range m.Images {
			if err := im.check(); err != nil {
				return err
			}
			total += len(im.Data)
			images++
		}
	}
	if images > MaxImages {
		return &Error{Code: CodeBadRequest, Msg: MsgBadImage}
	}
	if total > MaxRequestBytes {
		return &Error{Code: CodeBadRequest, Msg: msgTooLarge}
	}
	if r.MaxTokens < 0 || r.MaxTokens > MaxOutputTokens {
		return &Error{Code: CodeBadRequest, Msg: msgBadMaxTokens}
	}
	if r.Temperature != nil && (*r.Temperature < 0 || *r.Temperature > 2) {
		return &Error{Code: CodeBadRequest, Msg: msgBadTemperature}
	}
	if len(r.JSONSchema) > 0 && !json.Valid(r.JSONSchema) {
		return &Error{Code: CodeBadRequest, Msg: msgBadSchema}
	}
	if r.Caller != Background && r.Caller != Interactive {
		return &Error{Code: CodeBadRequest, Msg: msgBadCaller}
	}
	return nil
}

// imageTypes are the formats every vision model takes.
var imageTypes = map[string]bool{"image/png": true, "image/jpeg": true, "image/gif": true, "image/webp": true}

// check accepts an image only when its bytes are what its type says.
func (im Image) check() error {
	if !imageTypes[im.MIME] || len(im.Data) == 0 || len(im.Data) > MaxImageBytes {
		return &Error{Code: CodeBadRequest, Msg: MsgBadImage}
	}
	if sniffImage(im.Data) != im.MIME {
		return &Error{Code: CodeBadRequest, Msg: MsgBadImage}
	}
	// A few KB can claim huge dimensions; whoever decodes it (the hub's
	// Claude Code or Codex, a provider) would spend gigabytes on it.
	w, h, err := imageSize(im.MIME, im.Data)
	if err != nil || w <= 0 || h <= 0 || w > MaxImageSide || h > MaxImageSide || w*h > MaxImagePixels {
		return &Error{Code: CodeBadRequest, Msg: MsgBadImage}
	}
	return nil
}

// HasImages reports whether any message carries an image.
func (r *Request) HasImages() bool {
	for _, m := range r.Messages {
		if len(m.Images) > 0 {
			return true
		}
	}
	return false
}

// Usage is what one call consumed.
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	// CostMicroUSD is an estimate from the configured price table.
	CostMicroUSD int64 `json:"cost_micro_usd"`
}

// Chunk is one piece of a streamed reply.
type Chunk struct {
	Text string `json:"text"`
	// Model, when set, is the exact model that is answering as the
	// provider reports it (e.g. "claude-opus-5-5" behind the alias
	// "opus"). It comes at most once, possibly with no text.
	Model string `json:"model,omitempty"`
}

// Stream is a reply in progress. Recv returns io.EOF after the last chunk;
// Usage and FinishReason are valid after that. Close must be called.
type Stream interface {
	Recv() (Chunk, error)
	Usage() Usage
	FinishReason() string
	Close() error
}

// Locality says where data goes when this model is used, as seen from the
// caller. A caller must treat anything it cannot confirm as Remote.
type Locality string

const (
	// Local: the data never leaves this machine.
	Local Locality = "local"
	// Remote: the data reaches a cloud provider (possibly via the hub).
	Remote Locality = "remote"
)

// ModelInfo describes a model the caller may use.
type ModelInfo struct {
	ID          string   `json:"id"`           // provider/model
	DisplayName string   `json:"display_name"` // for people
	Provider    string   `json:"provider"`
	Locality    Locality `json:"locality"`
	// HumanOnly models may not be used by Background callers.
	HumanOnly bool `json:"human_only,omitempty"`
	// OnHost: the model runs on the hub's own machine (a local model).
	// For a device it is still Remote (the data leaves the device); this
	// only lets a screen say "on your Mac".
	OnHost bool `json:"on_host,omitempty"`
}

// Code is a stable error code. Local API status codes map 1:1 (design §6).
type Code string

const (
	CodeBadRequest        Code = "bad_request"                 // 400
	CodeUnsupportedFormat Code = "unsupported_response_format" // 400
	CodeInvalidToken      Code = "invalid_token"               // 401
	CodeModelNotAllowed   Code = "model_not_allowed"           // 403
	CodeBusy              Code = "busy"                        // 429 + Retry-After
	CodeBudgetExceeded    Code = "budget_exceeded"             // 429 + Retry-After
	CodeHubOffline        Code = "hub_offline"                 // 503
	CodeUpstream          Code = "upstream_error"              // 502
	CodeCredentialProblem Code = "credential_problem"          // 502
	CodeRegionNotAllowed  Code = "region_not_allowed"          // 503
	CodeInternal          Code = "internal"                    // 500
	CodeNotFound          Code = "not_found"                   // 404
	CodePairingFailed     Code = "pairing_failed"
	CodePairingDenied     Code = "pairing_denied"
	CodeToolNotAllowed    Code = "tool_not_allowed"
	CodeForbiddenOrigin   Code = "forbidden_origin" // 403
	CodeDevicePaused      Code = "device_paused"    // 403: the owner paused this device or app
)

// Error is the only error type that crosses the library boundary toward
// people or devices. Msg is a key into the message table, never upstream
// text; Args fill its placeholders (provider names or kinds, last four
// characters). Describe renders them in the reader's language: a kind
// becomes the name people know ("Claude 订阅" / "your Claude subscription")
// and a wait comes from RetryAfter, so no Args hold language-specific text.
// Detail is for the host's logs only, never sent to a device, and is
// redacted before it is set.
type Error struct {
	Code       Code
	Msg        string
	Args       []string
	RetryAfter time.Duration
	Detail     string
}

func (e *Error) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("omnillm: %s: %s", e.Code, e.Detail)
	}
	return "omnillm: " + string(e.Code)
}

// Is lets errors.Is match on the code alone.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	return ok && t.Code == e.Code && t.Msg == "" && t.Detail == "" && t.Args == nil
}

// Sentinel values for errors.Is.
var (
	ErrBadRequest        = &Error{Code: CodeBadRequest}
	ErrUnsupportedFormat = &Error{Code: CodeUnsupportedFormat}
	ErrInvalidToken      = &Error{Code: CodeInvalidToken}
	ErrModelNotAllowed   = &Error{Code: CodeModelNotAllowed}
	ErrBusy              = &Error{Code: CodeBusy}
	ErrBudgetExceeded    = &Error{Code: CodeBudgetExceeded}
	ErrHubOffline        = &Error{Code: CodeHubOffline}
	ErrUpstream          = &Error{Code: CodeUpstream}
	ErrCredential        = &Error{Code: CodeCredentialProblem}
	ErrRegionNotAllowed  = &Error{Code: CodeRegionNotAllowed}
	ErrNotFound          = &Error{Code: CodeNotFound}
	ErrPairingFailed     = &Error{Code: CodePairingFailed}
	ErrPairingDenied     = &Error{Code: CodePairingDenied}
	ErrToolNotAllowed    = &Error{Code: CodeToolNotAllowed}
	ErrDevicePaused      = &Error{Code: CodeDevicePaused}
)

// AsError returns err as *Error, wrapping anything else as internal.
func AsError(err error) *Error {
	if err == nil {
		return nil
	}
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return &Error{Code: CodeInternal, Msg: msgInternal}
}

// HTTPStatus maps a code to the local API status (design §6).
func HTTPStatus(c Code) int {
	switch c {
	case CodeBadRequest, CodeUnsupportedFormat:
		return 400
	case CodeInvalidToken:
		return 401
	case CodeModelNotAllowed, CodeToolNotAllowed, CodeForbiddenOrigin, CodeDevicePaused:
		return 403
	case CodeNotFound:
		return 404
	case CodeBusy, CodeBudgetExceeded:
		return 429
	case CodeUpstream, CodeCredentialProblem:
		return 502
	case CodeHubOffline, CodeRegionNotAllowed:
		return 503
	default:
		return 500
	}
}
