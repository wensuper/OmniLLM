package provider

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// What a subscription needs before the hub can use it, step by step, so a
// page can say the one thing left to do: the official program installed
// (Claude Code, Codex), and logged in where the hub can use the login.
// A desktop app (Claude.app, ChatGPT.app) is noticed only to say so: the
// subscription is the same, the command-line program is what the hub runs.

// desktopApps are the desktop apps that come with each subscription.
var desktopApps = map[Kind][]string{
	ClaudeCode: {"Claude.app"},
	Codex:      {"ChatGPT.app", "Codex.app"},
}

// DesktopApp is the subscription's desktop app installed here ("Claude",
// "ChatGPT", "Codex"; macOS: /Applications or ~/Applications), "" when
// none is.
func DesktopApp(k Kind) string {
	if runtime.GOOS != "darwin" {
		return ""
	}
	home, _ := os.UserHomeDir()
	for _, dir := range []string{"/Applications", filepath.Join(home, "Applications")} {
		for _, app := range desktopApps[k] {
			if st, err := os.Stat(filepath.Join(dir, app)); err == nil && st.IsDir() {
				return strings.TrimSuffix(app, ".app")
			}
		}
	}
	return ""
}

// CodexLoggedInOwn asks the owner's own Codex, with its own settings,
// whether it is logged in. It is for telling apart "not logged in" from
// "logged in, but kept where the hub can't use it" (the system keychain:
// the hub runs Codex on the auth.json login only). It never reads the
// login itself.
func CodexLoggedInOwn(ctx context.Context, path string) bool {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	dir, err := os.MkdirTemp("", "omnillm-cli-")
	if err != nil {
		return false
	}
	defer os.RemoveAll(dir)
	if !TrustedCLI(path) {
		return false
	}
	cmd := exec.CommandContext(ctx, path, "login", "status")
	cmd.Dir, cmd.Env = dir, cliEnv(path, dir)
	if h := os.Getenv("CODEX_HOME"); h != "" {
		cmd.Env = append(cmd.Env, "CODEX_HOME="+h)
	}
	out, err := cmd.CombinedOutput()
	return err == nil && strings.Contains(strings.ToLower(string(out)), "logged in")
}

// CodexFileLogin is the setting the guided Codex login uses, so the login
// lands in auth.json, where the hub's Codex can use it.
const CodexFileLogin = codexFileLogin
