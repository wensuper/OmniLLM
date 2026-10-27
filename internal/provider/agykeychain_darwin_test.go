package provider

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMain: no test makes a real keychain; the keychain test below uses a
// stand-in for /usr/bin/security.
func TestMain(m *testing.M) {
	os.Setenv("OMNILLM_AGY_KEYCHAIN", "off")
	os.Exit(m.Run())
}

// The agy home gets its own keychain, made once, set as the default only
// for that home, and unlocked every time; nothing names the owner's home.
func TestAgyKeychain(t *testing.T) {
	t.Setenv("OMNILLM_AGY_KEYCHAIN", "")
	dir := t.TempDir()
	log := filepath.Join(dir, "calls")
	fake := filepath.Join(dir, "security")
	// Records HOME and the arguments; "default-keychain -d user" answers
	// with whatever was set last.
	script := "#!/bin/sh\necho \"$HOME|$*\" >> " + log + "\n" +
		"case \"$1 $#\" in 'create-keychain 4') mkdir -p \"$(dirname \"$4\")\"; : > \"$4\";; esac\n" +
		"if [ \"$1\" = default-keychain ] && [ $# = 3 ]; then cat " + dir + "/default 2>/dev/null; fi\n" +
		"if [ \"$1\" = default-keychain ] && [ $# = 5 ]; then echo \"\\\"$5\\\"\" > " + dir + "/default; fi\n"
	if err := os.WriteFile(fake, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	old := securityTool
	securityTool = fake
	defer func() { securityTool = old }()
	home := filepath.Join(dir, "agy-home")
	os.MkdirAll(home, 0o700)
	kc := filepath.Join(home, "Library", "Keychains", "login.keychain-db")

	agyKeychain(home)
	agyKeychain(home)
	b, _ := os.ReadFile(log)
	calls := strings.Split(strings.TrimSpace(string(b)), "\n")
	count := func(prefix string) (n int) {
		for _, c := range calls {
			if strings.HasPrefix(c, home+"|"+prefix) {
				n++
			}
		}
		return
	}
	for _, c := range calls {
		if !strings.HasPrefix(c, home+"|") {
			t.Fatalf("ran with another HOME: %q", c)
		}
	}
	if count("create-keychain -p  "+kc) != 1 || count("set-keychain-settings "+kc) != 1 {
		t.Fatalf("made more or less than once: %q", calls)
	}
	if count("default-keychain -d user -s "+kc) != 1 || count("list-keychains -d user -s "+kc) != 1 {
		t.Fatalf("default set more or less than once: %q", calls)
	}
	if count("unlock-keychain -p  "+kc) != 2 {
		t.Fatalf("not unlocked every time: %q", calls)
	}
	// Switched off (tests elsewhere): nothing runs.
	os.Remove(log)
	t.Setenv("OMNILLM_AGY_KEYCHAIN", "off")
	agyKeychain(home)
	if _, err := os.Stat(log); err == nil {
		t.Fatal("ran while switched off")
	}
}
