// Package e2e is the end-to-end encrypted channel between a device and the
// hub (principle 2). The relay in between only ever sees the bytes this
// package produces.
//
// Handshakes use Noise IK (normal sessions) and Noise IKpsk2 (pairing, the
// pairing secret from the on-screen code as the pre-shared key), with
// X25519, ChaCha20-Poly1305 and BLAKE2s. The device is always the initiator
// and must already know the hub's static public key; that key only ever
// comes from the pairing code shown on the hub's own screen.
//
// The first handshake message carries no business data (no 0-RTT): it can be
// replayed, so it only carries the protocol version. Transport messages use
// Noise's strictly increasing nonces, so a replayed, dropped or reordered
// message fails to decrypt and the session is closed.
package e2e

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/flynn/noise"
)

// ProtocolVersion is carried in the handshake payload. Versions only add
// (principle: additive protocol changes).
const ProtocolVersion = 1

var suite = noise.NewCipherSuite(noise.DH25519, noise.CipherChaChaPoly, noise.HashBLAKE2s)

// KeyPair is a long-term X25519 static key.
type KeyPair struct {
	Private []byte
	Public  []byte
}

// GenerateKey makes a new static key pair.
func GenerateKey() (KeyPair, error) {
	k, err := suite.GenerateKeypair(rand.Reader)
	if err != nil {
		return KeyPair{}, err
	}
	return KeyPair{Private: k.Private, Public: k.Public}, nil
}

// Marshal / UnmarshalKey store a key pair as 64 bytes (private||public).
func (k KeyPair) Marshal() []byte { return append(append([]byte{}, k.Private...), k.Public...) }

func UnmarshalKey(b []byte) (KeyPair, error) {
	if len(b) != 64 {
		return KeyPair{}, errors.New("e2e: bad key length")
	}
	return KeyPair{Private: append([]byte{}, b[:32]...), Public: append([]byte{}, b[32:]...)}, nil
}

// MsgConn moves whole messages between the two ends (through the relay).
type MsgConn interface {
	ReadMessage(ctx context.Context) ([]byte, error)
	WriteMessage(ctx context.Context, b []byte) error
	Close() error
}

// Hello is the only handshake payload. It holds nothing secret or
// business-related, because handshake message 1 is replayable.
type Hello struct {
	V    int      `json:"v"`
	Caps []string `json:"caps,omitempty"`
}

const (
	maxHandshakeMsg = 1024
	// Noise caps a message at 65535 bytes; the AEAD tag takes 16.
	maxPlain = 65535 - 16
	// Frame header: 1 byte flags, 4 bytes data length.
	frameHeader = 5
	flagMore    = 1
	// MaxMessage is the largest reassembled application message.
	MaxMessage = 8 << 20
	// Sessions are re-established after this many messages or this long,
	// whichever comes first (rekeying by re-handshake).
	MaxSessionMessages = 1 << 20
	MaxSessionAge      = time.Hour
	// Past the soft limits (Expired) a new session should be opened for new
	// work; streams already running may finish until the hard limits.
	hardMessages     = 2 * MaxSessionMessages
	hardAge          = MaxSessionAge + 15*time.Minute
	handshakeTimeout = 30 * time.Second
)

// Padding buckets for frame plaintexts, so the relay learns less from sizes.
var buckets = []int{1024, 4096, 16384, maxPlain}

func prologue(mailbox []byte) []byte {
	return append([]byte("omnillm/e2e/1\x00"), mailbox...)
}

// SAS is the six-digit code both screens show during pairing. It is derived
// from the handshake hash, so a man in the middle would produce different
// codes on the two ends.
func SAS(channelBinding []byte) string {
	h := sha256.Sum256(append([]byte("omnillm/sas/1\x00"), channelBinding...))
	n := binary.BigEndian.Uint32(h[:4]) % 1_000_000
	return fmt.Sprintf("%06d", n)
}

// Session is an established encrypted channel. Send is safe for concurrent
// use; Recv must be called from one goroutine.
type Session struct {
	conn       MsgConn
	send, recv *noise.CipherState
	peer       []byte
	binding    []byte
	started    time.Time

	sendMu sync.Mutex
	sent   atomic.Uint64
	recvN  atomic.Uint64
	broken atomic.Bool
	closed chan struct{}
	once   sync.Once
}

// PeerStatic is the authenticated static public key of the other end.
func (s *Session) PeerStatic() []byte { return append([]byte{}, s.peer...) }

// ChannelBinding is the handshake hash (for SAS).
func (s *Session) ChannelBinding() []byte { return append([]byte{}, s.binding...) }

// Expired reports whether the session should be replaced by a new handshake.
func (s *Session) Expired() bool {
	return s.broken.Load() || s.sent.Load() >= MaxSessionMessages || s.recvN.Load() >= MaxSessionMessages || time.Since(s.started) > MaxSessionAge
}

func (s *Session) pastHard() bool {
	return s.sent.Load() >= hardMessages || s.recvN.Load() >= hardMessages || time.Since(s.started) > hardAge
}

// ErrExpired is returned once a session must be replaced.
var ErrExpired = errors.New("e2e: session expired, re-handshake required")

// Close closes the underlying connection.
func (s *Session) Close() error {
	var err error
	s.once.Do(func() {
		close(s.closed)
		err = s.conn.Close()
	})
	return err
}

// Send encrypts and sends one application message, fragmenting and padding.
func (s *Session) Send(ctx context.Context, msg []byte) error {
	if len(msg) > MaxMessage {
		return errors.New("e2e: message too large")
	}
	s.sendMu.Lock()
	defer s.sendMu.Unlock()
	if s.broken.Load() {
		return errors.New("e2e: session broken")
	}
	if s.pastHard() {
		return ErrExpired
	}
	maxData := maxPlain - frameHeader
	for first := true; first || len(msg) > 0; first = false {
		n := len(msg)
		flags := byte(0)
		if n > maxData {
			n = maxData
			flags = flagMore
		}
		plain := frame(flags, msg[:n])
		msg = msg[n:]
		ct, err := s.send.Encrypt(nil, nil, plain)
		if err != nil {
			s.broken.Store(true)
			return err
		}
		if err := s.conn.WriteMessage(ctx, ct); err != nil {
			// The nonce has advanced; the peer can never decrypt what
			// follows, so this session is over.
			s.broken.Store(true)
			s.conn.Close()
			return err
		}
		s.sent.Add(1)
	}
	return nil
}

func frame(flags byte, data []byte) []byte {
	need := frameHeader + len(data)
	size := buckets[len(buckets)-1]
	for _, b := range buckets {
		if need <= b {
			size = b
			break
		}
	}
	out := make([]byte, size)
	out[0] = flags
	binary.BigEndian.PutUint32(out[1:5], uint32(len(data)))
	copy(out[frameHeader:], data)
	return out
}

// Recv returns the next whole application message.
func (s *Session) Recv(ctx context.Context) ([]byte, error) {
	var buf []byte
	for {
		if s.pastHard() {
			return nil, ErrExpired
		}
		ct, err := s.conn.ReadMessage(ctx)
		if err != nil {
			return nil, err
		}
		if len(ct) > maxPlain+16 {
			return nil, errors.New("e2e: oversized frame")
		}
		plain, err := s.recv.Decrypt(nil, nil, ct)
		if err != nil {
			// Tampered, replayed, reordered or dropped: the session is over.
			return nil, fmt.Errorf("e2e: decrypt: %w", err)
		}
		s.recvN.Add(1)
		if len(plain) < frameHeader {
			return nil, errors.New("e2e: short frame")
		}
		flags := plain[0]
		n := int(binary.BigEndian.Uint32(plain[1:5]))
		if n > len(plain)-frameHeader || flags&^flagMore != 0 {
			return nil, errors.New("e2e: bad frame")
		}
		if len(buf)+n > MaxMessage {
			return nil, errors.New("e2e: message too large")
		}
		buf = append(buf, plain[frameHeader:frameHeader+n]...)
		if flags&flagMore == 0 {
			return buf, nil
		}
	}
}

func helloPayload() []byte {
	b, _ := json.Marshal(Hello{V: ProtocolVersion})
	return b
}

func parseHello(b []byte) (Hello, error) {
	var h Hello
	if len(b) == 0 || len(b) > 512 {
		return h, errors.New("e2e: bad hello")
	}
	if err := json.Unmarshal(b, &h); err != nil {
		return h, errors.New("e2e: bad hello")
	}
	if h.V < 1 {
		return h, errors.New("e2e: bad protocol version")
	}
	return h, nil
}

// Dial runs the initiator (device) side. psk is the pairing secret for a
// pairing session, nil for a normal one.
func Dial(ctx context.Context, conn MsgConn, mailbox []byte, local KeyPair, hubStatic, psk []byte) (*Session, error) {
	if len(hubStatic) != 32 {
		return nil, errors.New("e2e: bad hub key")
	}
	ctx, cancel := context.WithTimeout(ctx, handshakeTimeout)
	defer cancel()
	cfg := noise.Config{
		CipherSuite:   suite,
		Random:        rand.Reader,
		Pattern:       noise.HandshakeIK,
		Initiator:     true,
		Prologue:      prologue(mailbox),
		StaticKeypair: noise.DHKey{Private: local.Private, Public: local.Public},
		PeerStatic:    hubStatic,
	}
	if psk != nil {
		if len(psk) != 32 {
			return nil, errors.New("e2e: bad psk")
		}
		cfg.PresharedKey = psk
		cfg.PresharedKeyPlacement = 2
	}
	hs, err := noise.NewHandshakeState(cfg)
	if err != nil {
		return nil, err
	}
	msg1, _, _, err := hs.WriteMessage(nil, helloPayload())
	if err != nil {
		return nil, err
	}
	if err := conn.WriteMessage(ctx, msg1); err != nil {
		return nil, err
	}
	msg2, err := conn.ReadMessage(ctx)
	if err != nil {
		return nil, err
	}
	if len(msg2) > maxHandshakeMsg {
		return nil, errors.New("e2e: oversized handshake")
	}
	payload, send, recv, err := hs.ReadMessage(nil, msg2)
	if err != nil {
		return nil, fmt.Errorf("e2e: handshake: %w", err)
	}
	if send == nil || recv == nil {
		return nil, errors.New("e2e: handshake incomplete")
	}
	if _, err := parseHello(payload); err != nil {
		return nil, err
	}
	return &Session{conn: conn, send: send, recv: recv, peer: append([]byte{}, hubStatic...),
		binding: hs.ChannelBinding(), started: time.Now(), closed: make(chan struct{})}, nil
}

// Authorizer decides, after message 1 is decrypted, whether this device
// static key may open a session on this mailbox.
type Authorizer func(deviceStatic []byte) error

// Accept runs the responder (hub) side. psk is set for a pairing mailbox,
// in which case any device key is accepted by the handshake and the caller
// must confirm the device separately.
func Accept(ctx context.Context, conn MsgConn, mailbox []byte, local KeyPair, psk []byte, authorize Authorizer) (*Session, error) {
	ctx, cancel := context.WithTimeout(ctx, handshakeTimeout)
	defer cancel()
	cfg := noise.Config{
		CipherSuite:   suite,
		Random:        rand.Reader,
		Pattern:       noise.HandshakeIK,
		Initiator:     false,
		Prologue:      prologue(mailbox),
		StaticKeypair: noise.DHKey{Private: local.Private, Public: local.Public},
	}
	if psk != nil {
		if len(psk) != 32 {
			return nil, errors.New("e2e: bad psk")
		}
		cfg.PresharedKey = psk
		cfg.PresharedKeyPlacement = 2
	}
	hs, err := noise.NewHandshakeState(cfg)
	if err != nil {
		return nil, err
	}
	msg1, err := conn.ReadMessage(ctx)
	if err != nil {
		return nil, err
	}
	if len(msg1) > maxHandshakeMsg {
		return nil, errors.New("e2e: oversized handshake")
	}
	payload, _, _, err := hs.ReadMessage(nil, msg1)
	if err != nil {
		return nil, fmt.Errorf("e2e: handshake: %w", err)
	}
	if _, err := parseHello(payload); err != nil {
		return nil, err
	}
	peer := hs.PeerStatic()
	if len(peer) != 32 {
		return nil, errors.New("e2e: no peer key")
	}
	if authorize != nil {
		if err := authorize(peer); err != nil {
			return nil, err
		}
	}
	msg2, recv, send, err := hs.WriteMessage(nil, helloPayload())
	if err != nil {
		return nil, err
	}
	if err := conn.WriteMessage(ctx, msg2); err != nil {
		return nil, err
	}
	return &Session{conn: conn, send: send, recv: recv, peer: append([]byte{}, peer...),
		binding: hs.ChannelBinding(), started: time.Now(), closed: make(chan struct{})}, nil
}

// KeyEqual compares two public keys in constant time.
func KeyEqual(a, b []byte) bool {
	return len(a) == len(b) && subtle.ConstantTimeCompare(a, b) == 1
}

// RandomBytes returns n bytes from crypto/rand.
func RandomBytes(n int) []byte {
	b := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		panic("omnillm: crypto/rand failed: " + err.Error())
	}
	return b
}
