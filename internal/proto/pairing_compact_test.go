package proto

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	omnillm "github.com/wensuper/OmniLLM"
)

func samplePairing() PairingCode {
	return PairingCode{Relay: "wss://relay.ixxzh.com", Mailbox: strings.Repeat("ab", 16), Secret: bytes.Repeat([]byte{1}, 32),
		HubKey: bytes.Repeat([]byte{2}, 32), PSK: bytes.Repeat([]byte{3}, 32), Expires: time.Now().Add(5 * time.Minute).Unix()}
}

func TestCompactPairingCode(t *testing.T) {
	want := samplePairing()
	text := want.Encode()
	if !strings.HasPrefix(text, "omnillm3-") {
		t.Fatalf("not compact: %s", text)
	}
	old, _ := json.Marshal(want)
	oldText := "omnillm1-" + base64.RawURLEncoding.EncodeToString(old)
	if len(text) > len(oldText)*65/100 {
		t.Fatalf("compact %d chars vs %d: not much shorter", len(text), len(oldText))
	}
	for _, in := range []string{text, "https://omni.ixxzh.com/#pair=" + text, oldText} {
		got, err := ParsePairingCode(in, time.Now())
		if err != nil || got.Relay != want.Relay || got.Mailbox != want.Mailbox || !bytes.Equal(got.PSK, want.PSK) ||
			!bytes.Equal(got.HubKey, want.HubKey) || !bytes.Equal(got.Secret, want.Secret) || got.Expires != want.Expires {
			t.Fatalf("%q: %+v %v", in[:20], got, err)
		}
	}
	// One changed character anywhere is caught by the checksum.
	for i := 9; i < len(text); i += 7 {
		c := byte('A')
		if text[i] == 'A' {
			c = 'B'
		}
		bad := text[:i] + string(c) + text[i+1:]
		if _, err := ParsePairingCode(bad, time.Now()); err == nil {
			t.Fatalf("damaged at %d accepted", i)
		}
	}
	// The alpha.4 form (no checksum) is still read.
	raw, _ := base64.RawURLEncoding.DecodeString(text[9:])
	if got, err := ParsePairingCode("omnillm2-"+base64.RawURLEncoding.EncodeToString(raw[4:]), time.Now()); err != nil || got.Relay != want.Relay {
		t.Fatalf("omnillm2 form: %v", err)
	}
	for _, bad := range []string{text[:40], "omnillm2-" + base64.RawURLEncoding.EncodeToString(make([]byte, 116)), "omnillm2-!!", "omnillm3-abc", "omnillm4-abc"} {
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

// People pasting a broken or wrong text get a message that says what to do,
// not "something went wrong inside".
func TestPairingTextErrorsSayWhatToDo(t *testing.T) {
	good := samplePairing().Encode()
	cases := map[string]string{
		good[:len(good)-6]:          omnillm.MsgPairingDamaged,
		good[:20] + "x" + good[21:]: omnillm.MsgPairingDamaged,
		"omnillm1-" + base64.RawURLEncoding.EncodeToString([]byte(`{"relay":"wss://x"}`)): omnillm.MsgPairingDamaged,
		"hello":                   omnillm.MsgNotPairing,
		strings.Repeat("a", 3000): omnillm.MsgNotPairing,
	}
	for text, want := range cases {
		_, err := ParsePairingCode(text, time.Now())
		e := omnillm.AsError(err)
		if e == nil || e.Msg != want || e.Code == omnillm.CodeInternal {
			t.Errorf("%.30q: got %v, want message %s", text, err, want)
		}
	}
}
