package store

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestFileStore(t *testing.T) {
	p := filepath.Join(t.TempDir(), "d", "state.json")
	f, err := OpenFile(p)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := PutJSON(ctx, f, "device/a", map[string]int{"x": 1}); err != nil {
		t.Fatal(err)
	}
	_ = f.Put(ctx, "device/b", []byte(`1`))
	_ = f.Put(ctx, "app/c", []byte(`2`))
	st, _ := os.Stat(p)
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("mode %v", st.Mode().Perm())
	}
	g, err := OpenFile(p)
	if err != nil {
		t.Fatal(err)
	}
	keys, _ := g.List(ctx, "device/")
	if len(keys) != 2 || keys[0] != "device/a" {
		t.Fatal(keys)
	}
	var m map[string]int
	if err := GetJSON(ctx, g, "device/a", &m); err != nil || m["x"] != 1 {
		t.Fatal(m, err)
	}
	_ = g.Delete(ctx, "device/a")
	if _, err := g.Get(ctx, "device/a"); !errors.Is(err, ErrNotFound) {
		t.Fatal("delete")
	}
}

func TestCorruptFileRefused(t *testing.T) {
	p := filepath.Join(t.TempDir(), "state.json")
	_ = os.WriteFile(p, []byte("{not json"), 0o600)
	if _, err := OpenFile(p); err == nil {
		t.Fatal("corrupt file opened (and would be overwritten)")
	}
}
