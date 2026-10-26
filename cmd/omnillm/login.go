package main

// Guided subscription login (design §10.8). A Claude or ChatGPT
// subscription is used through its official program, Claude Code or
// Codex, logged in by the owner. Many people have only the desktop app, or
// the program without a login. "omnillm login claude" (and the console's
// button, which opens the same steps in a Terminal window) installs the
// program with its official installer when it is missing, runs its own
// login (the browser opens; the owner signs in there), and connects it.
// OmniLLM never sees the login: the program keeps it.

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/wensuper/OmniLLM/internal/control"
	"github.com/wensuper/OmniLLM/internal/provider"
)

// loginKind turns what a person types (claude, chatgpt, codex) into the
// provider kind; "" when it is neither.
func loginKind(s string) provider.Kind {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "claude", "claude-code", "anthropic":
		return provider.ClaudeCode
	case "chatgpt", "codex", "openai":
		return provider.Codex
	case "gemini", "google", "antigravity", "agy":
		return provider.Antigravity
	}
	return ""
}

// shq quotes s for a POSIX shell.
func shq(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

// loginScript is the bash script that installs (when missing) and logs in
// the official program of kind. Every command it runs is printed first, so
// the owner sees what happens in their Terminal. headless: no browser on
// this machine (Codex then uses its device code). cli is the program when
// it is installed ("" when not: the script installs it first). window: it
// runs in a window opened for it (the console's button), which the owner
// may close at the end; otherwise omnillm login says what follows.
func loginScript(k provider.Kind, cli string, headless, window bool) string {
	home, _ := os.UserHomeDir()
	var name, sub, word, install, bin, login, how, failed string
	if k == provider.Antigravity {
		return agyLoginScript(cli, headless, window)
	}
	if k == provider.ClaudeCode {
		name, sub, word = "Claude Code", t("Claude 订阅", "Claude subscription"), "claude"
		install = "curl -fsSL https://claude.ai/install.sh | bash"
		bin = filepath.Join(home, ".local", "bin", "claude")
		login = `"$CLI" auth login --claudeai`
		how = t("登录：浏览器会打开，用你的 Claude 账号确认就行。", "Sign in: your browser opens; confirm with your Claude account.")
		if headless {
			how = t("登录：下面会显示一个网址，在手机或别的电脑上打开它登录，再把页面给的代码粘回这里。", "Sign in: an address appears below; open it on your phone or another computer, sign in, and paste the code it gives you back here.")
		}
		failed = t("登录没有完成。如果你用的是免费版 Claude：要 Pro 或 Max 才能用 Claude Code。否则再试一次：", "The sign-in didn’t finish. On the free Claude plan, Claude Code needs Pro or Max. Otherwise try again: ")
	} else {
		name, sub, word = "Codex", t("ChatGPT 订阅", "ChatGPT subscription"), "chatgpt"
		// The installer would otherwise ask "Start Codex now?" and open
		// Codex itself instead of going on to sign in.
		install = "curl -fsSL https://chatgpt.com/codex/install.sh | CODEX_NON_INTERACTIVE=1 sh"
		bin = filepath.Join(home, ".local", "bin", "codex")
		// The login goes to auth.json, where the hub's Codex can use it.
		login = `"$CLI" -c ` + shq(provider.CodexFileLogin) + ` login`
		how = t("登录：浏览器会打开，用你的 ChatGPT 账号确认就行。", "Sign in: your browser opens; confirm with your ChatGPT account.")
		if headless {
			login += " --device-auth"
			how = t("登录：下面会显示一个网址和一次性代码，在手机或别的电脑上打开网址、输入代码。要先在 ChatGPT 的“安全”设置里打开“设备码登录”。", "Sign in: an address and a one-time code appear below; open the address on your phone or another computer and enter the code. Turn on device code login in ChatGPT’s security settings first.")
		}
		failed = t("登录没有完成。再试一次：", "The sign-in didn’t finish. Try again: ")
	}
	again := t("回到 OmniLLM 控制台再点一次，或者运行 omnillm login "+word+"。", "choose it again in the OmniLLM console, or run omnillm login "+word+".")
	var b strings.Builder
	w := func(format string, a ...any) { fmt.Fprintf(&b, format+"\n", a...) }
	w("#!/bin/bash")
	// A failed download must fail "curl … | bash" too.
	w("set -o pipefail")
	w("# OmniLLM: sign in with %s, the official program.", name)
	w("printf '\\033]0;OmniLLM\\007'")
	w("say() { printf '%%s\\n' \"$*\"; }")
	w("say %s", shq(fmt.Sprintf(t("OmniLLM：用官方程序 %s 登录你的 %s。登录只留在 %[1]s 里，OmniLLM 看不到。", "OmniLLM: sign in to your %[2]s with %[1]s, the official program. The login stays in %[1]s; OmniLLM never sees it."), name, sub)))
	w("say ''")
	w("CLI=%s", shq(cli))
	w(`if [ -z "$CLI" ] || [ ! -x "$CLI" ]; then`)
	w("  say %s", shq(fmt.Sprintf(t("先用官方安装程序装上 %s（不用 sudo，约 1 分钟）。下面是它的输出，里面让你做的事都不用管，这里会接着往下走：", "First, %s is installed with its official installer (no sudo, about a minute). Its output follows; skip anything it asks you to do, this goes on by itself:"), name)))
	w("  say %s", shq("  "+install))
	w("  %s || { say ''; say %s; exit 1; }", install, shq(t("安装没有完成：多半是网络问题。检查网络（或者换个网络）后，", "The install didn’t finish, most likely the network. Check the network (or try another one), then ")+again))
	w("  [ -x %s ] && CLI=%s || CLI=$(command -v %s)", shq(bin), shq(bin), cliWord(k))
	w(`  [ -n "$CLI" ] && [ -x "$CLI" ] || { say ''; say %s; exit 1; }`, shq(fmt.Sprintf(t("安装程序说装好了，但没找到 %s。新开一个终端窗口后，", "The installer finished, but %s wasn’t found. Open a new terminal window, then "), name)+again))
	w("  say ''")
	w("fi")
	w("say %s", shq(how))
	w("%s || { say ''; say %s; exit 1; }", login, shq(failed+again))
	if window {
		// Connect it now: no page may be open to notice the sign-in.
		if self := installedSelf(); self != "" {
			w("%s login --connect %s >/dev/null 2>&1", shq(self), word)
		}
		w("say ''")
		w("say %s", shq(fmt.Sprintf(t("好了：%s 已登录。回到浏览器里的 OmniLLM 页面。这个窗口可以关掉。", "Done: %s is signed in. Go back to the OmniLLM page in your browser. You can close this window."), name)))
	}
	return b.String()
}

// phoneNext is the sign-in window's closing line when a phone is still to
// be added (no device yet, remote access on): the page goes on to its QR
// code. "" otherwise.
func phoneNext(k provider.Kind) string {
	st, ok := queryStatus()
	if !ok || st.Devices > 0 || !st.RelayConfigured {
		return ""
	}
	return fmt.Sprintf("say %s\n", shq(t("接着在那个页面上用手机扫二维码，把手机加进来。", "Then scan the QR code on that page with your phone to add it.")))
}

// cliWord is the program's command name.
func cliWord(k provider.Kind) string {
	switch k {
	case provider.ClaudeCode:
		return "claude"
	case provider.Antigravity:
		return "agy"
	}
	return "codex"
}

// openLoginWindow opens the guided login for kind ("claude-code", "codex") in a
// Terminal window (macOS), for the console's button.
func openLoginWindow(kind string) error {
	k := provider.Kind(kind)
	if !k.Subscription() {
		return errors.New("login: not a subscription")
	}
	if runtime.GOOS != "darwin" {
		return say(runnable(fmt.Sprintf(t("在终端里运行：omnillm login %s", "In a terminal, run: omnillm login %s"), loginWord(k))))
	}
	// A .command file opens in Terminal by itself (no permission to
	// control Terminal needed); it is ours, in the hub's folder, 0700.
	p := filepath.Join(dataDir("hub"), "login-"+kind+".command")
	_ = os.Remove(p) // a fresh file, never one a link points elsewhere from
	if err := os.WriteFile(p, []byte(loginScript(k, provider.FindCLI(k), false, true)+phoneNext(k)), 0o700); err != nil {
		return err
	}
	_ = os.Chmod(p, 0o700)
	return exec.Command("open", "-a", "Terminal", p).Run()
}

// loginWord is the word omnillm login takes for kind.
func loginWord(k provider.Kind) string {
	switch k {
	case provider.ClaudeCode:
		return "claude"
	case provider.Antigravity:
		return "gemini"
	}
	return "chatgpt"
}

// loginCmd is "omnillm login claude|chatgpt": the guided login in this
// terminal, then the hub connects it.
func loginCmd(args []string) {
	// --connect: only the last step, for the sign-in window's script.
	if len(args) == 2 && args[0] == "--connect" && loginKind(args[1]) != "" {
		sock := filepath.Join(dataDir("hub"), "control.sock")
		_ = ctl(sock, control.Request{Op: "subscription.connect", Args: map[string]string{"kind": string(loginKind(args[1]))}, Lang: lang()},
			func(control.Reply) (bool, bool) { return false, false })
		return
	}
	if len(args) != 1 || loginKind(args[0]) == "" {
		// The one line that says it all; the whole command list would bury it.
		fmt.Fprintln(os.Stderr, runnable(t("用法：omnillm login claude（Claude 订阅）、omnillm login chatgpt（ChatGPT 订阅）或 omnillm login gemini（Gemini 订阅，用 Antigravity）。",
			"Usage: omnillm login claude (Claude subscription), omnillm login chatgpt (ChatGPT subscription) or omnillm login gemini (Google AI subscription, through Antigravity).")))
		os.Exit(2)
	}
	k := loginKind(args[0])
	if role() == "device" {
		die(say(t("这台电脑只当设备用，用的是你另一台电脑上的订阅：订阅要在那台电脑上登录（在那台电脑上运行 omnillm login）。",
			"This computer is set up as a device and uses the subscription on your other computer: sign in there (run omnillm login on that computer).")))
	}
	// Signing in is the person's step: an assistant gets it in words.
	if notAPerson() {
		fmt.Println(runnable(fmt.Sprintf(t("登录是你自己的一步：在这台机器上你自己的终端里运行 omnillm login %s。", "Signing in is your own step: in your own terminal on this machine, run omnillm login %s."), loginWord(k))))
		return
	}
	cmd := exec.Command("/bin/bash", "-c", loginScript(k, provider.FindCLI(k), headless(), false))
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		os.Exit(1)
	}
	// Connect it now (the hub would also find it by itself).
	if _, up := queryStatus(); !up {
		fmt.Println(runnable(t("OmniLLM 现在没在运行；运行 omnillm install 启动它，它会自己接上。", "OmniLLM isn’t running now; run omnillm install to start it, and it connects this by itself.")))
		return
	}
	sock := filepath.Join(dataDir("hub"), "control.sock")
	var msg string
	err := ctl(sock, control.Request{Op: "subscription.connect", Args: map[string]string{"kind": string(k)}, Lang: lang()}, func(r control.Reply) (bool, bool) {
		msg = r.Text
		return false, false
	})
	if err != nil {
		// The hub answered: signed in, but it could not be used (say why).
		die(err)
	}
	fmt.Println(msg)
	afterModel()
}

// afterModel is what a terminal says once a model is connected: on a
// machine without a screen, straight on to the phone's QR code; elsewhere,
// the next step while there is no device yet.
func afterModel() {
	st, ok := queryStatus()
	if !ok {
		return
	}
	st.Running, st.Headless = true, headless()
	if pairsHere(st) {
		pairHere()
		return
	}
	if st.Devices == 0 && st.RelayConfigured {
		fmt.Println(runnable(nextStep(st)))
	}
}

// installedSelf is the installed omnillm (~/.local/bin/omnillm), for a
// script to call back; "" when it isn't there.
func installedSelf() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	p := filepath.Join(home, ".local", "bin", "omnillm")
	if st, err := os.Stat(p); err != nil || st.IsDir() || st.Mode()&0o111 == 0 {
		return ""
	}
	return p
}
