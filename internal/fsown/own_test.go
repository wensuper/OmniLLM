package fsown

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMine(t *testing.T) {
	p := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(p, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if !Mine(fi) || !MineOrRoot(fi) {
		t.Fatal("a file this test made is its own")
	}
	if fi, err := os.Stat("/"); err == nil && os.Getuid() != 0 && Mine(fi) {
		t.Fatal("/ belongs to root")
	}
}
