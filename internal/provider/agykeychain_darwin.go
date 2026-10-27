package provider

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// securityTool is macOS's keychain command; agy calls it by this path to
// keep its sign-in.
var securityTool = "/usr/bin/security"

// agyKeychain gives the hub's agy home a keychain of its own. agy keeps
// its sign-in in the default keychain of $HOME, and the hub's agy home has
// none, so every refresh of the sign-in put up "A keychain cannot be found
// to store “antigravity”" until the person answered it. The keychain lives
// in the agy home (never the owner's), has an empty password and doesn't
// lock by itself: it holds what agy's 0600 token file next to it already
// holds, so it protects no less. Each start unlocks it (after a restart
// macOS has it locked, and a locked keychain asks for its password).
// Failures are left to agy, which falls back to its file.
func agyKeychain(home string) {
	if os.Getenv("OMNILLM_AGY_KEYCHAIN") == "off" { // tests: no keychain is made
		return
	}
	kc := filepath.Join(home, "Library", "Keychains", "login.keychain-db")
	sec := func(args ...string) ([]byte, error) {
		cmd := exec.Command(securityTool, args...)
		cmd.Env = []string{"HOME=" + home, "PATH=/usr/bin:/bin:/usr/sbin:/sbin"}
		cmd.Dir = home
		return cmd.CombinedOutput()
	}
	if _, err := os.Stat(kc); err != nil {
		if err := os.MkdirAll(filepath.Dir(kc), 0o700); err != nil {
			return
		}
		if _, err := sec("create-keychain", "-p", "", kc); err != nil {
			return
		}
		_, _ = sec("set-keychain-settings", kc) // no timeout, no lock on sleep
	}
	// Only the agy home's own list and default, read from its own
	// Library/Preferences: the owner's keychains aren't touched.
	if out, err := sec("default-keychain", "-d", "user"); err != nil || !strings.Contains(string(out), kc) {
		_, _ = sec("list-keychains", "-d", "user", "-s", kc)
		_, _ = sec("default-keychain", "-d", "user", "-s", kc)
	}
	_, _ = sec("unlock-keychain", "-p", "", kc)
}
