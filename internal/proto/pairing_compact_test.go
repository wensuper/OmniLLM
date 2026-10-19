package proto

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func samplePairing() PairingCode {
	return PairingCode{Relay: "wss://relay.ixxzh.com", Mailbox: strings.Repeat("ab", 16), Secret: bytes.Repeat([]byte{1}, 32),
		HubKey: bytes.Repeat([]byte{2}, 32), PSK: bytes.Repeat([]byte{3}, 32), Expires: time.Now().Add(5 * time.Minute).Unix()}
}

func TestCompactPairingCode(t *testing.T) {
	want := samplePairing()
	text := want.Encode()
	if !strings.HasPrefix(text, "omnillm2-") {
		t.Fatalf("not compact: %s", text)
	}
	old, _ := json.Marshal(want)
	oldText := "omnillm1-" + base64.RawURLEncoding.EncodeToString(old)
	if len(text) > len(oldText)*6/10 {
		t.Fatalf("compact %d chars vs %d: not much shorter", len(text), len(oldText))
	}
	for _, in := range []string{text, "https://omni.ixxzh.com/#pair=" + text, oldText} {
		got, err := ParsePairingCode(in, time.Now())
		if err != nil || got.Relay != want.Relay || got.Mailbox != want.Mailbox || !bytes.Equal(got.PSK, want.PSK) ||
			!bytes.Equal(got.HubKey, want.HubKey) || !bytes.Equal(got.Secret, want.Secret) || got.Expires != want.Expires {
			t.Fatalf("%q: %+v %v", in[:20], got, err)
		}
	}
	for _, bad := range []string{text[:40], "omnillm2-" + base64.RawURLEncoding.EncodeToString(make([]byte, 116)), "omnillm2-!!", "omnillm3-abc"} {
		if _, err := ParsePairingCode(bad, time.Now()); err == nil {
			t.Fatalf("accepted %q", bad)
		}
	}
	expired := samplePairing()
	expired.Expires = time.Now().Add(-time.Minute).Unix()
	if _, err := ParsePairingCode(expired.Encode(), time.Now()); err == nil {
		t.Fatal("expired code accepted")
	}
}
