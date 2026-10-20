package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// ~/.local/bin is not on a new Mac's PATH, nor on a new Linux account's
// until it exists at login. Every next step we print says "omnillm …", so
// the installer puts the folder on PATH for new terminals, and until then
// what we print names the program by its full path.

// cmdName is how a person can run omnillm from this terminal right now.
func cmdName() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "omnillm"
	}
	want := filepath.Join(home, ".local", "bin", "omnillm")
	if got, err := exec.LookPath("omnillm"); err == nil {
		if g, err := filepath.EvalSymlinks(got); err == nil && g == want || got == want {
			return "omnillm"
		}
	}
	return "~/.local/bin/omnillm"
}

// runnable rewrites "omnillm …" in a sentence for people into a command
// this terminal can run.
func runnable(s string) string {
	n := cmdName()
	if n == "omnillm" {
		return s
	}
	return bareCmd.ReplaceAllString(s, "${1}"+n+" ")
}

// bareCmd is "omnillm " not already part of a path or a word.
var bareCmd = regexp.MustCompile(`(^|[^/\w.-])omnillm `)

// rcFile is the startup file of the person's shell.
func rcFile(home string) (path, line string) {
	sh := filepath.Base(os.Getenv("SHELL"))
	switch {
	case sh == "zsh":
		return filepath.Join(home, ".zshrc"), `export PATH="$HOME/.local/bin:$PATH"`
	case sh == "bash" && runtime.GOOS == "darwin":
		return filepath.Join(home, ".bash_profile"), `export PATH="$HOME/.local/bin:$PATH"`
	case sh == "bash":
		return filepath.Join(home, ".bashrc"), `export PATH="$HOME/.local/bin:$PATH"`
	case sh == "fish":
		return filepath.Join(home, ".config", "fish", "config.fish"), `fish_add_path -g $HOME/.local/bin`
	}
	return filepath.Join(home, ".profile"), `export PATH="$HOME/.local/bin:$PATH"`
}

// ensureOnPath adds ~/.local/bin to the shell's startup file when this
// terminal can't find omnillm by name. It never rewrites what is there:
// one appended line, once.
func ensureOnPath(home string) {
	if cmdName() == "omnillm" {
		return
	}
	rc, line := rcFile(home)
	if b, err := os.ReadFile(rc); err == nil && strings.Contains(string(b), ".local/bin") {
		fmt.Println(t("新开一个终端窗口后，就可以直接输入 omnillm。", "Open a new terminal window and omnillm works by name."))
		return
	}
	// Only a plain file of the person's own: never write through a link.
	if fi, err := os.Lstat(rc); err == nil && !fi.Mode().IsRegular() {
		fmt.Println(t("提示：~/.local/bin 不在 PATH 里，请用完整路径 ~/.local/bin/omnillm 运行。", "Note: ~/.local/bin is not on PATH; run omnillm as ~/.local/bin/omnillm."))
		return
	}
	if err := os.MkdirAll(filepath.Dir(rc), 0o755); err != nil {
		return
	}
	f, err := os.OpenFile(rc, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Println(t("提示：~/.local/bin 不在 PATH 里，请用完整路径 ~/.local/bin/omnillm 运行。", "Note: ~/.local/bin is not on PATH; run omnillm as ~/.local/bin/omnillm."))
		return
	}
	defer f.Close()
	if _, err := fmt.Fprintf(f, "\n# Added by the OmniLLM installer\n%s\n", line); err != nil {
		return
	}
	short := strings.Replace(rc, home, "~", 1)
	fmt.Printf(t("已把 ~/.local/bin 加到 PATH（写在 %s）。新开一个终端窗口后，就可以直接输入 omnillm。\n",
		"Added ~/.local/bin to PATH (in %s). Open a new terminal window and omnillm works by name.\n"), short)
}
