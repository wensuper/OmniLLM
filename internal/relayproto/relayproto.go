// Package relayproto is the framing between the hub and the relay. The relay
// only ever routes opaque bytes by mailbox ID; everything inside is
// end-to-end ciphertext.
package relayproto

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

// Frame types on the hub link: [1 type][16 mailbox][payload].
const (
	Claim     byte = 1 // hub→relay: payload = sha256(mailbox secret)
	Release   byte = 2 // hub→relay
	Data      byte = 3 // both ways: one end-to-end message
	Kick      byte = 4 // hub→relay: close the device connection
	Open      byte = 5 // relay→hub: a device connected
	Closed    byte = 6 // relay→hub: the device disconnected
	ClaimFail byte = 7 // relay→hub
	// LinkDown is never on the wire: the hub's link client emits it when
	// its connection to the relay drops.
	LinkDown byte = 0xF0
)

const (
	IDLen = 16
	// MaxMessage bounds one relayed message (a Noise message is ≤ 65535).
	MaxMessage = 65535 + 1 + IDLen + 64
	// MaxClaimsPerHub bounds how many mailboxes one hub may hold.
	MaxClaimsPerHub = 1024
	// Close codes a device sees. Unknown mailbox, wrong secret and hub
	// offline all look the same, so the relay is not an oracle.
	CloseUnavailable = 4404
	CloseReplaced    = 4409
	CloseKicked      = 4403
)

// ID is a mailbox identifier.
type ID [IDLen]byte

func (id ID) String() string { return hex.EncodeToString(id[:]) }

// ParseID decodes a hex mailbox ID.
func ParseID(s string) (ID, error) {
	var id ID
	b, err := hex.DecodeString(s)
	if err != nil || len(b) != IDLen {
		return id, errors.New("relayproto: bad mailbox id")
	}
	copy(id[:], b)
	return id, nil
}

// SecretHash is what the hub sends in Claim and the relay compares against.
func SecretHash(secret []byte) [32]byte {
	return sha256.Sum256(append([]byte("omnillm/mailbox/1\x00"), secret...))
}

// Encode builds a hub-link frame.
func Encode(t byte, id ID, payload []byte) []byte {
	out := make([]byte, 1+IDLen+len(payload))
	out[0] = t
	copy(out[1:], id[:])
	copy(out[1+IDLen:], payload)
	return out
}

// Decode splits a hub-link frame.
func Decode(b []byte) (byte, ID, []byte, error) {
	var id ID
	if len(b) < 1+IDLen || len(b) > MaxMessage {
		return 0, id, nil, errors.New("relayproto: bad frame")
	}
	copy(id[:], b[1:1+IDLen])
	return b[0], id, b[1+IDLen:], nil
}
