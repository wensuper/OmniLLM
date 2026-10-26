package main

import (
	"fmt"
	"strings"

	"github.com/wensuper/OmniLLM/internal/provider"
)

// agyLoginScript is loginScript for a Google AI subscription: Google's
// Antigravity CLI (agy), installed with Homebrew when missing, signed in
// in the hub's own home for it (provider.AgyHome), not the owner's: the
// hub runs agy there with no rules and no tools. The owner's own agy
// login, if any, is not touched. agy has no separate login command: it
// signs in when it starts, then opens a chat, which the owner leaves with
// Ctrl+C; the script then asks agy whether it is signed in.
func agyLoginScript(cli string, headless, window bool) string {
	home := provider.AgyHome()
	again := t("回到 OmniLLM 控制台再点一次，或者运行 omnillm login gemini。", "choose it again in the OmniLLM console, or run omnillm login gemini.")
	how := t("登录：选「Google OAuth」，浏览器会打开，用你的 Google 账号确认。看到对话界面后按 Ctrl+C 退出，这里会接着往下走。",
		"Sign in: choose “Google OAuth”; your browser opens; confirm with your Google account. When the chat screen appears, press Ctrl+C to leave it; this goes on by itself.")
	if headless {
		how = t("登录：选「Google OAuth」，下面会显示一个网址，在手机或别的电脑上打开它登录，再把页面给的代码粘回这里。看到对话界面后按 Ctrl+C 退出。",
			"Sign in: choose “Google OAuth”; an address appears below; open it on your phone or another computer, sign in, and paste the code it gives you back here. When the chat screen appears, press Ctrl+C to leave it.")
	}
	var b strings.Builder
	w := func(format string, a ...any) { fmt.Fprintf(&b, format+"\n", a...) }
	w("#!/bin/bash")
	w("# OmniLLM: sign in to Antigravity CLI, Google's official program, for OmniLLM.")
	w("printf '\\033]0;OmniLLM\\007'")
	w("say() { printf '%%s\\n' \"$*\"; }")
	w("say %s", shq(t("OmniLLM：用 Google 官方程序 Antigravity CLI 登录你的 Gemini 订阅。这次登录只给 OmniLLM 用，放在 OmniLLM 自己的文件夹里；登录只留在 Antigravity 里，OmniLLM 看不到。你自己平时用的 agy 不受影响。",
		"OmniLLM: sign in to your Gemini subscription with Antigravity CLI, Google's official program. This sign-in is for OmniLLM only and is kept in OmniLLM's own folder; it stays inside Antigravity, and OmniLLM never sees it. The agy you use yourself isn't affected.")))
	w("say ''")
	w("CLI=%s", shq(cli))
	w(`if [ -z "$CLI" ] || [ ! -x "$CLI" ]; then`)
	w(`  BREW=$(command -v brew || { [ -x /opt/homebrew/bin/brew ] && echo /opt/homebrew/bin/brew; } || { [ -x /usr/local/bin/brew ] && echo /usr/local/bin/brew; })`)
	w(`  if [ -z "$BREW" ]; then say %s; exit 1; fi`, shq(t("要先安装 Antigravity CLI：打开 https://antigravity.google/product/antigravity-cli 按说明装好，然后", "Install Antigravity CLI first: open https://antigravity.google/product/antigravity-cli and follow its steps, then ")+again))
	w("  say %s", shq(t("先用 Homebrew 装上 Antigravity CLI（不用 sudo，约 1 分钟）：", "First, Antigravity CLI is installed with Homebrew (no sudo, about a minute):")))
	w("  say '  brew install --cask antigravity-cli'")
	w(`  HOMEBREW_NO_AUTO_UPDATE=1 "$BREW" install --cask antigravity-cli || { say ''; say %s; exit 1; }`, shq(t("安装没有完成：多半是网络问题。检查网络后，", "The install didn’t finish, most likely the network. Check the network, then ")+again))
	w(`  CLI=$(command -v agy || echo "$(dirname "$BREW")/agy")`)
	w(`  [ -x "$CLI" ] || { say ''; say %s; exit 1; }`, shq(t("安装程序说装好了，但没找到 agy。新开一个终端窗口后，", "The installer finished, but agy wasn’t found. Open a new terminal window, then ")+again))
	w("  say ''")
	w("fi")
	w("AGYHOME=%s", shq(home))
	w(`mkdir -p "$AGYHOME" && chmod 700 "$AGYHOME"`)
	w(`if HOME="$AGYHOME" "$CLI" models >/dev/null 2>&1; then`)
	w("  say %s", shq(t("Antigravity 已经给 OmniLLM 登录好了。", "Antigravity is already signed in for OmniLLM.")))
	w("else")
	w("  say %s", shq(how))
	w(`  HOME="$AGYHOME" "$CLI"`)
	w(`  HOME="$AGYHOME" "$CLI" models >/dev/null 2>&1 || { say ''; say %s; exit 1; }`, shq(t("登录没有完成。再试一次：", "The sign-in didn’t finish. Try again: ")+again))
	w("fi")
	if window {
		if self := installedSelf(); self != "" {
			w("%s login --connect gemini >/dev/null 2>&1", shq(self))
		}
		w("say ''")
		w("say %s", shq(t("好了：Antigravity 已登录。回到浏览器里的 OmniLLM 页面。这个窗口可以关掉。", "Done: Antigravity is signed in. Go back to the OmniLLM page in your browser. You can close this window.")))
	}
	return b.String()
}
