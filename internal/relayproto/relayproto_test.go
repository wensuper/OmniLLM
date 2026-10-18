package relayproto

import "testing"

func FuzzDecode(f *testing.F) {
	var id ID
	f.Add(Encode(Data, id, []byte("x")))
	f.Add([]byte{1})
	f.Fuzz(func(t *testing.T, b []byte) {
		typ, gid, p, err := Decode(b)
		if err != nil {
			return
		}
		if len(b) < 1+IDLen || len(b) > MaxMessage {
			t.Fatal("accepted a bad length")
		}
		if string(Encode(typ, gid, p)) != string(b) {
			t.Fatal("round trip differs")
		}
	})
}

func TestParseID(t *testing.T) {
	for _, s := range []string{"", "zz", "00", "0123456789abcdef0123456789abcdeff"} {
		if _, err := ParseID(s); err == nil {
			t.Fatalf("accepted %q", s)
		}
	}
}
