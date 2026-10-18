// Package proto is the application protocol inside the end-to-end session,
// and the pairing code format. Every message is JSON with a type field.
package proto

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	omnillm "github.com/wensuper/OmniLLM"
)

// Message types. New types may be added; unknown types are ignored by both
// ends (additive protocol).
const (
	TPairHello  = "pair.hello"  // device→hub: {name}
	TPairOK     = "pair.ok"     // hub→device: {device_id}
	TPairDenied = "pair.denied" // hub→device
	TModelsList = "models.list" // device→hub
	TModels     = "models"      // hub→device: {models}
	TChatStart  = "chat.start"  // device→hub: {id, req}
	TChatAck    = "chat.ack"    // hub→device: {id}, sent on receipt
	TChatDelta  = "chat.delta"  // hub→device: {id, text}
	TChatEnd    = "chat.end"    // hub→device: {id, usage, finish}
	TChatError  = "chat.error"  // hub→device: {id, error}
	TChatCancel = "chat.cancel" // device→hub: {id}
	TToolCall   = "tool.call"   // hub→device: {id, tool, args}
	TToolResult = "tool.result" // device→hub: {id, result | error}
	TPing       = "ping"        // either way
	TPong       = "pong"
)

// Envelope is the outer shape of every message.
type Envelope struct {
	T  string `json:"t"`
	ID uint64 `json:"id,omitempty"`

	Name     string              `json:"name,omitempty"`
	DeviceID string              `json:"device_id,omitempty"`
	Models   []omnillm.ModelInfo `json:"models,omitempty"`
	Req      *omnillm.Request    `json:"req,omitempty"`
	Text     string              `json:"text,omitempty"`
	Usage    *omnillm.Usage      `json:"usage,omitempty"`
	Finish   string              `json:"finish,omitempty"`
	Err      *WireError          `json:"error,omitempty"`
	Tool     string              `json:"tool,omitempty"`
	Args     json.RawMessage     `json:"args,omitempty"`
	Result   json.RawMessage     `json:"result,omitempty"`
	// Secret is the replacement mailbox secret in pair.ok. It is only ever
	// marshalled inside the end-to-end session. #nosec G117
	Secret []byte `json:"secret,omitempty"`
}

// WireError is what crosses to a device: a code, the message key and its
// arguments, and a retry hint. Never upstream text, never Detail.
type WireError struct {
	Code       omnillm.Code `json:"code"`
	Msg        string       `json:"msg,omitempty"`
	Args       []string     `json:"args,omitempty"`
	RetryAfter int64        `json:"retry_after_ms,omitempty"`
}

// ToWire strips an error down to what a device may see.
func ToWire(err error) *WireError {
	e := omnillm.AsError(err)
	if e == nil {
		return nil
	}
	return &WireError{Code: e.Code, Msg: e.Msg, Args: e.Args, RetryAfter: e.RetryAfter.Milliseconds()}
}

// FromWire rebuilds an *omnillm.Error on the device.
func (w *WireError) Error() *omnillm.Error {
	if w == nil {
		return nil
	}
	return &omnillm.Error{Code: w.Code, Msg: w.Msg, Args: w.Args, RetryAfter: time.Duration(w.RetryAfter) * time.Millisecond}
}

// Marshal / Unmarshal envelopes.
func Marshal(e Envelope) ([]byte, error) { return json.Marshal(e) }

func Unmarshal(b []byte) (Envelope, error) {
	var e Envelope
	if err := json.Unmarshal(b, &e); err != nil {
		return e, errors.New("proto: bad message")
	}
	if e.T == "" || len(e.T) > 32 {
		return e, errors.New("proto: bad message type")
	}
	return e, nil
}

// PairingCode is everything a device needs to pair. It is shown only on the
// hub's own screen (or the owner's SSH terminal) and never passes through
// the relay.
type PairingCode struct {
	Relay   string `json:"r"`
	Mailbox string `json:"m"` // hex
	Secret  []byte `json:"s"` // mailbox secret (relay auth)
	HubKey  []byte `json:"k"` // hub static public key
	PSK     []byte `json:"p"` // one-time pairing secret
	Expires int64  `json:"e"` // unix seconds
}

const pairingPrefix = "omnillm1-"

// Encode renders the pairing text.
func (p PairingCode) Encode() string {
	b, _ := json.Marshal(p)
	return pairingPrefix + base64.RawURLEncoding.EncodeToString(b)
}

// ParsePairingCode reads a pairing text, checking shape and expiry.
func ParsePairingCode(s string, now time.Time) (PairingCode, error) {
	var p PairingCode
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, pairingPrefix) || len(s) > 2048 {
		return p, errors.New("proto: not a pairing code")
	}
	b, err := base64.RawURLEncoding.DecodeString(s[len(pairingPrefix):])
	if err != nil {
		return p, errors.New("proto: pairing code is damaged; copy it again")
	}
	if err := json.Unmarshal(b, &p); err != nil {
		return p, errors.New("proto: pairing code is damaged; copy it again")
	}
	if len(p.Secret) != 32 || len(p.HubKey) != 32 || len(p.PSK) != 32 || len(p.Mailbox) != 32 || p.Relay == "" {
		return p, errors.New("proto: pairing code is incomplete; copy it again")
	}
	if now.Unix() > p.Expires {
		return p, &omnillm.Error{Code: omnillm.CodePairingFailed, Msg: omnillm.MsgPairingExpired}
	}
	return p, nil
}
