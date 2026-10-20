// Package proto is the application protocol inside the end-to-end session,
// and the pairing code format. Every message is JSON with a type field.
package proto

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

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

// Pairing text formats, all still read:
//
//   - "omnillm1-" + base64url(JSON) (v0.2.0-alpha.3 and earlier);
//
//   - "omnillm2-" + base64url(binary) (alpha.4), about half as long, so the
//     QR code is far less dense and phones scan it fast;
//
//   - "omnillm3-" (written now): the same binary after a 4-byte checksum,
//     so a pasted text that lost or changed a character is refused as
//     damaged at once instead of failing later in a confusing way.
//
//     [sha256(rest)[:4] |] expires uint32 | mailbox 16 | secret 32 | hub key 32 | psk 32 | relay URL
const (
	pairingPrefix  = "omnillm1-"
	pairingPrefix2 = "omnillm2-"
	pairingPrefix3 = "omnillm3-"
	compactFixed   = 4 + 16 + 32 + 32 + 32
)

// Encode renders the pairing text (compact form).
func (p PairingCode) Encode() string {
	mb, err := hex.DecodeString(p.Mailbox)
	if err != nil || len(mb) != 16 || len(p.Secret) != 32 || len(p.HubKey) != 32 || len(p.PSK) != 32 {
		b, _ := json.Marshal(p) // not a complete code: keep it readable
		return pairingPrefix + base64.RawURLEncoding.EncodeToString(b)
	}
	b := make([]byte, compactFixed, compactFixed+len(p.Relay))
	binary.BigEndian.PutUint32(b[0:], uint32(p.Expires))
	copy(b[4:], mb)
	copy(b[20:], p.Secret)
	copy(b[52:], p.HubKey)
	copy(b[84:], p.PSK)
	b = append(b, p.Relay...)
	sum := sha256.Sum256(b)
	return pairingPrefix3 + base64.RawURLEncoding.EncodeToString(append(sum[:4], b...))
}

// Link is the pairing link for the web client at webApp (what the QR code
// holds): scanning it with a phone's camera opens the web client, which
// reads the code from the fragment.
func (p PairingCode) Link(webApp string) string {
	return webApp + "#pair=" + p.Encode()
}

// ParsePairingCode reads a pairing text, checking shape and expiry.
func ParsePairingCode(s string, now time.Time) (PairingCode, error) {
	var p PairingCode
	s = strings.TrimSpace(s)
	// A pairing link (what the QR code holds) carries the text after
	// "#pair=": the fragment, which a browser never sends to the web host.
	if i := strings.Index(s, "#pair="); i >= 0 {
		s = s[i+len("#pair="):]
	}
	// People see these, so they carry a message that says what to do.
	damaged := &omnillm.Error{Code: omnillm.CodeBadRequest, Msg: omnillm.MsgPairingDamaged, Detail: "pairing code is damaged"}
	notCode := &omnillm.Error{Code: omnillm.CodeBadRequest, Msg: omnillm.MsgNotPairing, Detail: "not a pairing code"}
	switch {
	case len(s) > 2048:
		return p, notCode
	case strings.HasPrefix(s, pairingPrefix2), strings.HasPrefix(s, pairingPrefix3):
		b, err := base64.RawURLEncoding.DecodeString(s[len(pairingPrefix2):])
		if err == nil && strings.HasPrefix(s, pairingPrefix3) {
			if len(b) < 4 {
				return p, damaged
			}
			sum := sha256.Sum256(b[4:])
			if subtle.ConstantTimeCompare(sum[:4], b[:4]) != 1 {
				return p, damaged
			}
			b = b[4:]
		}
		if err != nil || len(b) <= compactFixed || len(b) > compactFixed+512 || !utf8.Valid(b[compactFixed:]) {
			return p, damaged
		}
		p = PairingCode{Expires: int64(binary.BigEndian.Uint32(b[0:])), Mailbox: hex.EncodeToString(b[4:20]),
			Secret: append([]byte{}, b[20:52]...), HubKey: append([]byte{}, b[52:84]...), PSK: append([]byte{}, b[84:116]...),
			Relay: string(b[compactFixed:])}
	case strings.HasPrefix(s, pairingPrefix):
		b, err := base64.RawURLEncoding.DecodeString(s[len(pairingPrefix):])
		if err != nil {
			return p, damaged
		}
		if err := json.Unmarshal(b, &p); err != nil {
			return p, damaged
		}
	default:
		return p, notCode
	}
	if len(p.Secret) != 32 || len(p.HubKey) != 32 || len(p.PSK) != 32 || len(p.Mailbox) != 32 || p.Relay == "" {
		return p, damaged
	}
	if now.Unix() > p.Expires {
		return p, &omnillm.Error{Code: omnillm.CodePairingFailed, Msg: omnillm.MsgPairingExpired}
	}
	return p, nil
}
