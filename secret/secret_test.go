package secret

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestFileRoundTripAndPermissions(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "sub", "secrets.enc")
	key := bytes.Repeat([]byte{1}, 32)
	f, err := NewFile(p, key)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := f.Set(ctx, "provider/x/key", []byte("sk-PLAINTEXT-VALUE")); err != nil {
		t.Fatal(err)
	}
	st, _ := os.Stat(p)
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("file mode %v", st.Mode().Perm())
	}
	dst, _ := os.Stat(filepath.Dir(p))
	if dst.Mode().Perm() != 0o700 {
		t.Fatalf("dir mode %v", dst.Mode().Perm())
	}
	raw, _ := os.ReadFile(p)
	if bytes.Contains(raw, []byte("PLAINTEXT")) || bytes.Contains(raw, []byte("provider/x")) {
		t.Fatal("secret file is not encrypted")
	}
	g, _ := NewFile(p, key)
	v, err := g.Get(ctx, "provider/x/key")
	if err != nil || string(v) != "sk-PLAINTEXT-VALUE" {
		t.Fatal(v, err)
	}
	names, _ := g.List(ctx, "provider/")
	if len(names) != 1 {
		t.Fatal(names)
	}
	_ = g.Delete(ctx, "provider/x/key")
	if _, err := g.Get(ctx, "provider/x/key"); !errors.Is(err, ErrNotFound) {
		t.Fatal("delete")
	}
}

func TestFileWrongKeyAndTamper(t *testing.T) {
	p := filepath.Join(t.TempDir(), "s.enc")
	f, _ := NewFile(p, bytes.Repeat([]byte{1}, 32))
	ctx := context.Background()
	_ = f.Set(ctx, "a", []byte("b"))
	wrong, _ := NewFile(p, bytes.Repeat([]byte{2}, 32))
	if _, err := wrong.Get(ctx, "a"); err == nil {
		t.Fatal("wrong master key opened the file")
	}
	raw, _ := os.ReadFile(p)
	raw[len(raw)-1] ^= 1
	_ = os.WriteFile(p, raw, 0o600)
	if _, err := f.Get(ctx, "a"); err == nil {
		t.Fatal("tampered file accepted")
	}
	// A failed load must not be "fixed" by a write that loses everything.
	if err := f.Set(ctx, "c", []byte("d")); err == nil {
		t.Fatal("wrote over a file it could not read")
	}
}

func TestMasterKeyParsing(t *testing.T) {
	for _, s := range []string{"", "abc", "AAAA", "!!!!"} {
		if _, err := ParseMasterKey(s); err == nil {
			t.Fatalf("accepted %q", s)
		}
	}
	if _, err := NewFile("x", []byte("short")); err == nil {
		t.Fatal("short key accepted")
	}
	if _, err := ParseMasterKey("AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE="); err != nil {
		t.Fatal(err)
	}
}

func TestNames(t *testing.T) {
	for _, n := range []string{"", "../x", "/abs", "a b", "a\x00b", "a\nb", string(make([]byte, 201))} {
		if ValidName(n) == nil {
			t.Fatalf("accepted %q", n)
		}
	}
	m := NewMemory()
	if err := m.Set(context.Background(), "../escape", []byte("x")); err == nil {
		t.Fatal("memory store accepted a bad name")
	}
}

func TestMemoryCopies(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()
	v := []byte("abc")
	_ = m.Set(ctx, "k", v)
	v[0] = 'X'
	got, _ := m.Get(ctx, "k")
	got[1] = 'Y'
	again, _ := m.Get(ctx, "k")
	if string(again) != "abc" {
		t.Fatal("memory store aliases caller slices")
	}
}
