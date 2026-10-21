package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wensuper/OmniLLM/internal/provider"
)

// fakeHome gives the login script a home whose PATH holds only the fake
// programs in bin (and the system's tools).
func fakeHome(t *testing.T) (home, bin string) {
	home = t.TempDir()
	bin = filepath.Join(home, "fakebin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("PATH", bin+":/usr/bin:/bin")
	return home, bin
}

func writeExec(t *testing.T, p, body string) {
	t.Helper()
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestLoginScriptsAreValidBash(t *testing.T) {
	_, bin := fakeHome(t)
	// A path with a quote and a space must stay one word.
	odd := filepath.Join(bin, "it's here")
	if err := os.MkdirAll(odd, 0o755); err != nil {
		t.Fatal(err)
	}
	writeExec(t, filepath.Join(odd, "claude"), "exit 0\n")
	t.Setenv("PATH", odd+":/usr/bin:/bin")
	for _, l := range []string{"zh", "en"} {
		t.Setenv("OMNILLM_LANG", l)
		for _, k := range []provider.Kind{provider.ClaudeCode, provider.Codex} {
			for _, headless := range []bool{false, true} {
				s := loginScript(k, filepath.Join(odd, "claude"), headless, !headless)
				if out, err := exec.Command("bash", "-n", "-c", s).CombinedOutput(); err != nil {
					t.Fatalf("%s %s: %v %s\n%s", l, k, err, out, s)
				}
			}
		}
	}
}

// Installed but signed out: the script runs the program's own login (for
// Codex, kept in auth.json) and installs nothing.
func TestLoginScriptRunsTheOfficialLogin(t *testing.T) {
	home, bin := fakeHome(t)
	log := filepath.Join(home, "calls")
	writeExec(t, filepath.Join(bin, "claude"), `echo "claude $*" >> `+shq(log)+"\n")
	writeExec(t, filepath.Join(bin, "codex"), `echo "codex $*" >> `+shq(log)+"\n")
	writeExec(t, filepath.Join(bin, "curl"), `echo "curl $*" >> `+shq(log)+"\nexit 1\n")
	for k, cli := range map[provider.Kind]string{provider.ClaudeCode: filepath.Join(bin, "claude"), provider.Codex: filepath.Join(bin, "codex")} {
		if out, err := exec.Command("bash", "-c", loginScript(k, cli, false, true)).CombinedOutput(); err != nil {
			t.Fatalf("%s: %v %s", k, err, out)
		}
	}
	b, _ := os.ReadFile(log)
	got := string(b)
	if !strings.Contains(got, "claude auth login --claudeai") || !strings.Contains(got, `codex -c cli_auth_credentials_store="file" login`) || strings.Contains(got, "curl") {
		t.Fatalf("calls: %q", got)
	}
}

// Not installed (only a desktop app, say): the official installer runs
// first, then the login of what it installed in ~/.local/bin.
func TestLoginScriptInstallsFirst(t *testing.T) {
	home, bin := fakeHome(t)
	log := filepath.Join(home, "calls")
	local := filepath.Join(home, ".local", "bin")
	// The fake installer is "curl … | bash": curl prints a script that
	// puts a claude into ~/.local/bin.
	writeExec(t, filepath.Join(bin, "curl"), `echo "curl $*" >> `+shq(log)+`
cat <<'EOS'
mkdir -p "$HOME/.local/bin"
printf '#!/bin/sh\necho "installed-claude $*" >> `+log+`\n' > "$HOME/.local/bin/claude"
chmod +x "$HOME/.local/bin/claude"
EOS
`)
	if out, err := exec.Command("bash", "-c", loginScript(provider.ClaudeCode, "", false, true)).CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	b, _ := os.ReadFile(log)
	got := string(b)
	if !strings.Contains(got, "curl -fsSL https://claude.ai/install.sh") || !strings.Contains(got, "installed-claude auth login --claudeai") {
		t.Fatalf("calls: %q", got)
	}
	if _, err := os.Stat(filepath.Join(local, "claude")); err != nil {
		t.Fatal(err)
	}
}

// A failed install stops with a sentence, and nothing else runs.
func TestLoginScriptStopsWhenTheInstallFails(t *testing.T) {
	t.Setenv("OMNILLM_LANG", "en")
	_, bin := fakeHome(t)
	writeExec(t, filepath.Join(bin, "curl"), "exit 6\n")
	out, err := exec.Command("bash", "-c", loginScript(provider.Codex, "", false, true)).CombinedOutput()
	if err == nil || !strings.Contains(string(out), "The install didn’t finish") {
		t.Fatalf("%v %s", err, out)
	}
}

func TestLoginKindWords(t *testing.T) {
	for w, k := range map[string]provider.Kind{"claude": provider.ClaudeCode, "Claude": provider.ClaudeCode, "chatgpt": provider.Codex, "codex": provider.Codex, "gemini": ""} {
		if got := loginKind(w); got != k {
			t.Errorf("%s: %q", w, got)
		}
	}
}
