package provider

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wensuper/OmniLLM/internal/fsown"
)

// A program is used only when no one else could have put it (or a link to
// it) where it is found.
func TestTrustedCLI(t *testing.T) {
	d := t.TempDir()
	bin := filepath.Join(d, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(bin, "claude")
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !TrustedCLI(p) {
		t.Fatal("this user's own program is trusted")
	}
	if TrustedCLI("bin/claude") {
		t.Fatal("a relative path is not")
	}
	if err := os.Chmod(p, 0o775); err != nil {
		t.Fatal(err)
	}
	// Group-writable is trusted only when the group is the user's own
	// private one (as on many Linux systems, CI included).
	if fi, _ := os.Stat(p); TrustedCLI(p) != fsown.PrivateGroup(fi) {
		t.Fatalf("group-writable: trusted=%v, private group=%v", TrustedCLI(p), fsown.PrivateGroup(fi))
	}
	_ = os.Chmod(p, 0o755)
	// A link from a folder anyone can write to, even to a trusted file.
	open := filepath.Join(d, "open")
	if err := os.MkdirAll(open, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.Chmod(open, 0o777)
	link := filepath.Join(open, "claude")
	if err := os.Symlink(p, link); err != nil {
		t.Fatal(err)
	}
	if TrustedCLI(link) {
		t.Fatal("a link in a folder others can write to is not")
	}
	// The same file in a folder others can write to.
	_ = os.Chmod(bin, 0o777)
	if TrustedCLI(p) {
		t.Fatal("a program in a folder others can write to is not")
	}
}

func TestNodeNewer(t *testing.T) {
	a, b := "/h/.nvm/versions/node/v22.3.0/bin", "/h/.nvm/versions/node/v9.11.2/bin"
	if !nodeNewer(a, b) || nodeNewer(b, a) {
		t.Fatal("v22 is newer than v9")
	}
}
