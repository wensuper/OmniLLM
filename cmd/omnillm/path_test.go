package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A new Mac has no ~/.local/bin on PATH: the installer adds one line to the
// shell's startup file, once, and what it prints names the full path.
func TestEnsureOnPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/zsh")
	t.Setenv("PATH", "/usr/bin:/bin")
	if got := runnable("运行 omnillm open"); got != "运行 ~/.local/bin/omnillm open" {
		t.Fatalf("runnable: %q", got)
	}
	if got := runnable("运行 ~/.local/bin/omnillm open 或 omnillm-install.sh"); got != "运行 ~/.local/bin/omnillm open 或 omnillm-install.sh" {
		t.Fatalf("a path is not rewritten again: %q", got)
	}
	ensureOnPath(home)
	ensureOnPath(home)
	b, err := os.ReadFile(filepath.Join(home, ".zshrc"))
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(b), ".local/bin"); n != 1 {
		t.Fatalf("want one PATH line, got %d:\n%s", n, b)
	}
	// Once omnillm is found by name, nothing is rewritten.
	bin := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "omnillm"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+":/usr/bin")
	if got := runnable("运行 omnillm open"); got != "运行 omnillm open" {
		t.Fatalf("on PATH: %q", got)
	}
}

// A linked startup file is never written through.
func TestEnsureOnPathSkipsLinks(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/zsh")
	t.Setenv("PATH", "/usr/bin:/bin")
	target := filepath.Join(t.TempDir(), "elsewhere")
	if err := os.WriteFile(target, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(home, ".zshrc")); err != nil {
		t.Fatal(err)
	}
	ensureOnPath(home)
	if b, _ := os.ReadFile(target); string(b) != "x\n" {
		t.Fatalf("wrote through the link: %q", b)
	}
}
