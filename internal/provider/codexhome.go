package provider

// Codex reads the owner's instruction files (~/.codex/AGENTS.md) and
// settings from its home, whatever flags it gets, so a device could ask it
// to quote them. The hub therefore runs Codex in a home of its own that
// holds nothing of the owner's except the login: auth.json there is a
// symbolic link to the owner's auth.json. Nothing is copied, so there is
// one login and one refresh token and the two never log each other out
// (Codex rewrites auth.json in place when it refreshes, keeping the link).

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/wensuper/OmniLLM/internal/fsown"
)

// codexSourceHome is the owner's Codex home.
func codexSourceHome() string {
	if h := os.Getenv("CODEX_HOME"); h != "" && filepath.IsAbs(h) {
		return h
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".codex")
}

// codexHubHome is the hub's own Codex home.
func codexHubHome() string {
	if d := os.Getenv("OMNILLM_HOME"); d != "" {
		return filepath.Join(d, "codex-home")
	}
	base, err := os.UserConfigDir()
	if err != nil {
		base = os.TempDir()
	}
	return filepath.Join(base, "OmniLLM", "codex-home")
}

// errCodexLogin: the owner's Codex login is not in a file the hub can link
// (a keychain login, or none).
var errCodexLogin = errors.New("codex: no auth.json login to use")

// prepareCodexHome makes the hub's Codex home: private, an empty config,
// no instructions, and auth.json linking to the owner's. It repairs a link
// that points elsewhere and removes anything else the owner's files could
// have put there.
func prepareCodexHome() (string, error) {
	src := filepath.Join(codexSourceHome(), "auth.json")
	if st, err := os.Stat(src); err != nil || !st.Mode().IsRegular() {
		return "", errCodexLogin
	}
	home := codexHubHome()
	if err := os.MkdirAll(home, 0o700); err != nil {
		return "", err
	}
	// Ours alone: a real folder of this user's (not a link, not one
	// someone else made), private, or Codex would read their files.
	if fi, err := os.Lstat(home); err != nil || !fi.IsDir() || !fsown.Mine(fi) {
		return "", errors.New("codex: its folder isn't this user's own")
	}
	if err := os.Chmod(home, 0o700); err != nil {
		return "", err
	}
	link := filepath.Join(home, "auth.json")
	if cur, err := os.Readlink(link); err != nil || cur != src {
		_ = os.Remove(link)
		if err := os.Symlink(src, link); err != nil {
			return "", err
		}
	}
	for _, f := range []string{"AGENTS.md", "AGENTS.override.md", "config.toml", "instructions.md"} {
		_ = os.Remove(filepath.Join(home, f))
	}
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte("# OmniLLM hub: nothing of the owner's settings\n"), 0o600); err != nil {
		return "", err
	}
	return home, nil
}
