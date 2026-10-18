package proto

import (
	"errors"
	"strings"
	"testing"
	"time"

	omnillm "github.com/wensuper/OmniLLM"
)

func sample() PairingCode {
	b := func(n byte) []byte {
		x := make([]byte, 32)
		for i := range x {
			x[i] = n
		}
		return x
	}
	return PairingCode{Relay: "wss://relay.example.com", Mailbox: strings.Repeat("ab", 16), Secret: b(1), HubKey: b(2), PSK: b(3),
		Expires: time.Now().Add(5 * time.Minute).Unix()}
}

func TestPairingCodeRoundTrip(t *testing.T) {
	p := sample()
	got, err := ParsePairingCode("  "+p.Encode()+"\n", time.Now())
	if err != nil || got.Relay != p.Relay || string(got.PSK) != string(p.PSK) {
		t.Fatal(got, err)
	}
}

func TestPairingCodeExpiry(t *testing.T) {
	p := sample()
	p.Expires = time.Now().Add(-time.Second).Unix()
	_, err := ParsePairingCode(p.Encode(), time.Now())
	var e *omnillm.Error
	if !errors.As(err, &e) || e.Msg != omnillm.MsgPairingExpired {
		t.Fatalf("want expired, got %v", err)
	}
}

func TestPairingCodeRejects(t *testing.T) {
	p := sample()
	short := p
	short.PSK = short.PSK[:16]
	for _, s := range []string{"", "hello", "omnillm1-", "omnillm1-!!!", "omnillm1-e30", short.Encode(), strings.Repeat("a", 5000)} {
		if _, err := ParsePairingCode(s, time.Now()); err == nil {
			t.Fatalf("accepted %.40q", s)
		}
	}
}

func TestWireErrorDropsDetail(t *testing.T) {
	w := ToWire(&omnillm.Error{Code: omnillm.CodeUpstream, Msg: omnillm.MsgUpstream, Detail: "sk-secret upstream text", RetryAfter: 2 * time.Second})
	e := w.Error()
	if e.Detail != "" || e.RetryAfter != 2*time.Second || e.Code != omnillm.CodeUpstream {
		t.Fatalf("%+v", e)
	}
	// Non-library errors become a plain internal error.
	if w := ToWire(errors.New("raw sk-1234567890abcdef")); w.Code != omnillm.CodeInternal || w.Msg == "" {
		t.Fatalf("%+v", w)
	}
}

func FuzzParsePairingCode(f *testing.F) {
	f.Add(sample().Encode())
	f.Add("omnillm1-e30")
	f.Fuzz(func(t *testing.T, s string) {
		p, err := ParsePairingCode(s, time.Unix(0, 0))
		if err == nil && (len(p.PSK) != 32 || len(p.HubKey) != 32 || len(p.Secret) != 32) {
			t.Fatal("accepted a code with bad key lengths")
		}
	})
}

func FuzzUnmarshal(f *testing.F) {
	f.Add([]byte(`{"t":"chat.start","id":1,"req":{"model":"a/b","messages":[]}}`))
	f.Fuzz(func(t *testing.T, b []byte) {
		e, err := Unmarshal(b)
		if err == nil && (e.T == "" || len(e.T) > 32) {
			t.Fatal("bad type accepted")
		}
	})
}
