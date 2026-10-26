// Command omnillm is the standalone program: a thin shell around the
// library (principle 7). It adds a data directory, a control socket for the
// owner, and terminal prompts; all behaviour lives in the library.
package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/term"

	omnillm "github.com/wensuper/OmniLLM"
	"github.com/wensuper/OmniLLM/console"
	"github.com/wensuper/OmniLLM/device"
	"github.com/wensuper/OmniLLM/hub"
	"github.com/wensuper/OmniLLM/internal/control"
	"github.com/wensuper/OmniLLM/internal/e2e"
	"github.com/wensuper/OmniLLM/internal/fsown"
	"github.com/wensuper/OmniLLM/internal/provider"
	"github.com/wensuper/OmniLLM/internal/qrcode"
	"github.com/wensuper/OmniLLM/localapi"
	"github.com/wensuper/OmniLLM/relay"
	"github.com/wensuper/OmniLLM/secret"
	"github.com/wensuper/OmniLLM/store"
)

func zh() bool {
	for _, k := range []string{"OMNILLM_LANG", "LC_ALL", "LANG"} {
		if v := os.Getenv(k); v != "" {
			return strings.HasPrefix(strings.ToLower(v), "zh")
		}
	}
	return false
}

func lang() string {
	if zh() {
		return "zh"
	}
	return "en"
}

// t picks the Chinese or English text, in the terminal's language.
func t(z, e string) string { return tl(lang(), z, e) }

// tl picks the text in language l: what the hub answers is in the
// caller's language (control.Request.Lang), not its own.
func tl(l, z, e string) string {
	if l == "zh" {
		return z
	}
	return e
}

// callerLang is the language a control request asks for; an older CLI
// sends none and gets the hub's own.
func callerLang(req control.Request) string {
	if req.Lang == "zh" || req.Lang == "en" {
		return req.Lang
	}
	return lang()
}

// ctl is control.Call in the terminal's language: the hub writes its
// replies (sentences, questions, errors) in it.
func ctl(sock string, req control.Request, onReply func(control.Reply) (bool, bool)) error {
	if req.Lang == "" {
		req.Lang = lang()
	}
	err := control.Call(sock, req, onReply)
	var re *control.ReplyError
	if errors.As(err, &re) {
		return say(stripControls(re.Text))
	}
	var fs *control.ForeignSocketError
	if errors.As(err, &fs) {
		return sayf(t("警告：%s 属于另一个用户，OmniLLM 不会连它。检查 OMNILLM_HOME 指向的是不是你自己的文件夹。", "Warning: %s belongs to another user, so OmniLLM won’t use it. Check that OMNILLM_HOME points at a folder of your own."), fs.Path)
	}
	if errors.Is(err, control.ErrNotRunning) {
		switch {
		case role() == "device":
			return say(runnable(deviceHubHint()))
		case role() == "" && !exists(dataDir("hub")):
			return say(t("这台电脑上还没有安装 OmniLLM。安装方法见 https://github.com/wensuper/OmniLLM/blob/main/README.zh-CN.md#安装",
				"OmniLLM isn’t installed on this computer yet. See https://github.com/wensuper/OmniLLM#installation"))
		}
		return say(runnable(notRunning()))
	}
	return err
}

func die(err error) {
	if errors.Is(err, context.Canceled) { // Ctrl-C
		fmt.Fprintln(os.Stderr)
		os.Exit(130)
	}
	fmt.Fprintln(os.Stderr, describe(err))
	os.Exit(1)
}

// sayErr is a sentence already written for people (in their language).
type sayErr string

func (e sayErr) Error() string { return string(e) }

// usageErr is a command typed in a form it doesn't have: the person gets
// every command's form (subUsage); the text is the one line about it.
type usageErr string

func (e usageErr) Error() string { return string(e) }

// dieArgs is die for what a command's arguments gave, exit 2: a form it
// doesn't have prints the one-line reason first, then the full help (-h
// alone: the help, exit 0).
func dieArgs(err error) {
	var u usageErr
	if errors.As(err, &u) {
		if errors.Is(err, errHelp) {
			if role() == "device" {
				helpCmd(nil) // the commands this computer has
			} else {
				helpCmd([]string{"--all"})
			}
			os.Exit(0)
		}
		// The reason stays on screen; the full list is one command away.
		fmt.Fprintln(os.Stderr, string(u))
		fmt.Fprintln(os.Stderr, t("全部命令：omnillm help --all", "Every command: omnillm help --all"))
		os.Exit(2)
	}
	// A value that isn't right: the sentence says which, no help needed.
	fmt.Fprintln(os.Stderr, describe(err))
	os.Exit(2)
}

// isHelpWord is a word asking for the help: -h, --help, help.
func isHelpWord(w string) bool { return w == "-h" || w == "-help" || w == "--help" || w == "help" }

// errHelp is -h or --help after a command: the person asked for the help.
var errHelp = usageErr("help")

// badCommand is a hub or connect command that isn't one.
func badCommand(words ...string) error {
	return usageErr(fmt.Sprintf(t("不认识命令 omnillm %s。", "Unknown command omnillm %s."), strings.Join(words, " ")))
}

// flagWord finds the flag "-x" from Go's flag error as the person typed
// it ("--x", "-x", without "=value").
func flagWord(args []string, name string) string {
	bare := strings.TrimLeft(name, "-")
	for _, a := range args {
		w, _, _ := strings.Cut(a, "=")
		if strings.HasPrefix(w, "-") && strings.TrimLeft(w, "-") == bare {
			return w
		}
	}
	return "--" + bare
}

// flagMistake matches the flag Go's flag package names in its errors.
var flagMistake = regexp.MustCompile(`(?:for flag|for|argument:|defined:) (-\S+?):?(?:\s|$)`)

// parseFlags parses a command's options with fs, never letting the flag
// package print or exit: a mistake comes back as a usageErr whose text is
// the one line saying what was wrong. Words left over are refused too.
func parseFlags(fs *flag.FlagSet, args []string) error {
	if err := parseFlagsRest(fs, args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return usageErr(fmt.Sprintf(t("多了“%s”。", "Unexpected “%s”."), fs.Arg(0)))
	}
	return nil
}

// parseFlagsRest is parseFlags for a command that takes words after its
// options (connect ask's question).
func parseFlagsRest(fs *flag.FlagSet, args []string) error {
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}
	err := fs.Parse(args)
	switch {
	case errors.Is(err, flag.ErrHelp):
		return errHelp
	case err != nil:
		msg, name := err.Error(), ""
		if m := flagMistake.FindStringSubmatch(msg); m != nil {
			name = flagWord(args, m[1])
		}
		switch {
		case strings.HasPrefix(msg, "flag provided but not defined"):
			return usageErr(fmt.Sprintf(t("不认识选项 %s。", "Unknown option %s."), name))
		case strings.HasPrefix(msg, "flag needs an argument"):
			return usageErr(fmt.Sprintf(t("选项 %s 后面要写一个值。", "Option %s needs a value."), name))
		case name != "":
			return usageErr(fmt.Sprintf(t("选项 %s 的值不对。", "The value of option %s isn’t valid."), name))
		}
		return usageErr(t("选项写得不对。", "Those options aren’t valid."))
	}
	return nil
}

func say(s string) error                 { return sayErr(s) }
func sayf(format string, a ...any) error { return sayErr(fmt.Sprintf(format, a...)) }

// hubSay is what the hub answers a CLI over the control socket: the
// message for people only; an unexpected cause goes to the hub's own log.
func hubSay(l string, err error) error {
	if e := omnillm.AsError(err); e.Code == omnillm.CodeInternal {
		log.Printf("hub: control: %v", err)
		return say(omnillm.ForHub(l, runtime.GOOS, omnillm.Describe(l, err)) + tl(l, "（原因写在 OmniLLM 的日志里）", " (the cause is in OmniLLM’s log)"))
	}
	return say(omnillm.ForHub(l, runtime.GOOS, omnillm.Describe(l, err)))
}

// deviceHubOS is the hub's system as a device command learned it (from
// the models reply); "" before, or on a hub. Device messages say "your
// Mac" only when it is one.
var deviceHubOS string

// onDevice: this command speaks as a device (connect …).
var onDevice bool

// describe is what a person sees for err: our own sentence as it is, the
// library's message, and for anything unexpected the generic message plus
// the underlying cause, so it can be reported.
func describe(err error) string {
	var s sayErr
	if errors.As(err, &s) {
		return string(s)
	}
	var u usageErr
	if errors.As(err, &u) {
		return string(u)
	}
	msg := omnillm.Describe(lang(), err)
	if onDevice {
		msg = omnillm.ForHub(lang(), deviceHubOS, msg)
		// Removed on the other side: this side still holds the pairing.
		if omnillm.AsError(err).Msg == omnillm.MsgDeviceRevoked {
			msg += runnable(t("在这台电脑上先运行 omnillm connect forget，再运行 omnillm connect pair。", " On this computer, run omnillm connect forget, then omnillm connect pair."))
		}
	}
	if e := omnillm.AsError(err); e.Code == omnillm.CodeInternal {
		msg += "\n  (" + err.Error() + ")"
	}
	return msg
}

// usageText is the short list of everyday commands, or with all every
// command (hub, devices, relay admin).
func usageText(all bool) string {
	if all {
		daily := onMachine(t("%s 每天的总额度，例如 omnillm hub daily-total 20", "daily total for %s, e.g. omnillm hub daily-total 20"))
		return strings.Replace(t(`用法：
  omnillm install                    一步装好：主机作为后台服务运行，打开控制台
  omnillm open                       打开本机控制台
  omnillm open --add                 打开控制台，直接到「添加设备」
  omnillm pair                       在终端里添加手机：显示二维码，手机扫码后核对 6 位数（同 omnillm hub pair）
  omnillm verify web [网页地址]      核对网页版和 GitHub 上签名发布的是否一致
  omnillm status                     查看主机状态
  omnillm update [--check] [--auto on|off]
                                     更新到最新版本（主机默认会在空闲时自己更新）；--auto 开关自动更新
  omnillm login claude|chatgpt|gemini
                                     用 Claude、ChatGPT 或 Gemini 订阅：没装官方程序会先装上，再在浏览器里登录
  omnillm uninstall [--purge]        停止并移除后台服务（--purge 连数据和 API Key 一起删）
  omnillm tray                       把菜单栏图标找回来
  omnillm version                    显示版本
  omnillm hub run                    启动主机（前台运行）
  omnillm hub provider add <openai|anthropic|ollama|openai-compatible> [--name N] [--base-url U]
                                     接入模型来源（API Key 从标准输入读，不写在命令行里）
  omnillm hub provider rm <来源>     移除模型来源
  omnillm hub provider off|on <来源> 关闭或打开模型来源（登录和 Key 保留）
  omnillm hub providers | models | devices | apps | usage [--json]
                                     列出来（--json 输出原始数据）
  omnillm hub usage [--day today|yesterday|7d]
                                     用量；--day 看哪一天：今天、昨天或最近 7 天
  omnillm hub audit                  今天每次调用的记录（JSON，给脚本用）
  omnillm hub pair [--name N] [--models 名字或ID,…] [--budget-usd 5 | --no-limit] [--background-only]
  omnillm hub revoke <名字或ID>      立即移除设备（名字里有空格或括号时加引号：omnillm hub revoke "iPhone (2)"）
  omnillm hub device set <名字或ID> [--name N] [--paused true|false] [--budget-usd X | --no-limit] [--models 名字或ID,… | --all-models]
                                     改设备的名字、开启或暂停、每天额度、能用的模型
  omnillm hub app add <名字>         给不能嵌入库的程序发一枚应用令牌
  omnillm hub app set <名字或ID> [--name N] [--paused true|false] [--budget-usd X | --no-limit] [--models 名字或ID,… | --all-models]
                                     改应用的名字、开启或暂停、每天额度、能用的模型
  omnillm hub app rm <名字或ID>      移除应用，它的令牌立即失效
  omnillm hub model set <名字或ID> [--name 新名字] [--shown true|false] [--human-only true|false] [--auto true|false] [--price-in X --price-out Y]
                                     改名（--name "" 恢复原名）、是否在设备上显示、只能手动对话、Auto 可选、价格
  omnillm hub daily-total <美元> | --no-limit
                                     {daily}
  omnillm hub region confirm         确认主机所在地区变化并恢复调用
  omnillm connect pair               在这台设备上输入主机给的配对文字
  omnillm connect run [--listen 127.0.0.1:8766]
  omnillm connect chat [--model ID]  在终端里和主机上的模型对话
  omnillm connect ask [--model ID] [--image 图片文件] "问题"
  omnillm connect models             列出能用的模型
  omnillm connect forget             移除这台设备（主机上也一并移除）
  omnillm relay --listen :443 --autocert relay.example.com --hub-token-hash HEX
  omnillm relay token                生成一枚主机登记令牌和它的哈希
`, `Usage:
  omnillm install                    one step: run the hub as a background service, open the console
  omnillm open                       open the local console
  omnillm open --add                 open the console on Add device
  omnillm pair                       add a phone in the terminal: a QR code to scan, then check the 6 digits (same as omnillm hub pair)
  omnillm verify web [address]       check the web app against the signed GitHub release
  omnillm status                     show the hub’s status
  omnillm update [--check] [--auto on|off]
                                     update to the newest version (the hub also updates itself when idle); --auto turns that on or off
  omnillm login claude|chatgpt|gemini
                                     use your Claude, ChatGPT or Gemini subscription: installs the official program if needed, signs in in your browser
  omnillm uninstall [--purge]        stop and remove the background service (--purge also deletes data and API keys)
  omnillm tray                       bring back the menu bar icon
  omnillm version                    show the version
  omnillm hub run                    run the hub (foreground)
  omnillm hub provider add <openai|anthropic|ollama|openai-compatible> [--name N] [--base-url U]
                                     connect a model source (the API key is read from stdin, never argv)
  omnillm hub provider rm <source>   remove a model source
  omnillm hub provider off|on <source>  switch a model source off or on (its sign-in or key is kept)
  omnillm hub providers | models | devices | apps | usage [--json]
                                     list them (--json: the raw data)
  omnillm hub usage [--day today|yesterday|7d]
                                     usage; --day: which day: today, yesterday or the last 7 days
  omnillm hub audit                  today’s calls, one record each, as JSON (for scripts)
  omnillm hub pair [--name N] [--models name-or-id,…] [--budget-usd 5 | --no-limit] [--background-only]
  omnillm hub revoke <name-or-id>    remove a device now (quote a name with spaces or brackets: omnillm hub revoke "iPhone (2)")
  omnillm hub device set <name-or-id> [--name N] [--paused true|false] [--budget-usd X | --no-limit] [--models name-or-id,… | --all-models]
                                     rename a device, pause or resume it, change its daily limit or models
  omnillm hub app add <name>         token for a program that can’t embed the library
  omnillm hub app set <name-or-id> [--name N] [--paused true|false] [--budget-usd X | --no-limit] [--models name-or-id,… | --all-models]
                                     rename an app, pause or resume it, change its daily limit or models
  omnillm hub app rm <name-or-id>    remove an app; its token stops working at once
  omnillm hub model set <name-or-id> [--name new-name] [--shown true|false] [--human-only true|false] [--auto true|false] [--price-in X --price-out Y]
                                     rename (--name "" restores the original), show on devices, chat only, in Auto, prices
  omnillm hub daily-total <usd> | --no-limit
                                     {daily}
  omnillm hub region confirm         accept the hub’s new region and resume calls
  omnillm connect pair               enter the hub’s pairing text on this device
  omnillm connect run [--listen 127.0.0.1:8766]
  omnillm connect chat [--model ID]  chat with the hub’s models in this terminal
  omnillm connect ask [--model ID] [--image FILE] "question"
  omnillm connect models             list the models you can use
  omnillm connect forget             remove this device (from the hub too)
  omnillm relay --listen :443 --autocert relay.example.com --hub-token-hash HEX
  omnillm relay token                make a hub registration token and its hash
`), "{daily}", daily, 1)
	}
	if role() == "device" {
		return t(`用法（这台电脑只当设备用）：
  omnillm connect chat               在终端里和另一台电脑上的模型对话
  omnillm connect ask "问题"         问一句
  omnillm connect models             能用哪些模型
  omnillm connect run                给本机程序开一个本机接口
  omnillm connect pair               在这台电脑上输入配对文字
  omnillm connect forget             删除这里的配对
  omnillm update                     更新
  omnillm help --all                 全部命令
`, `Usage (this computer is set up as a device):
  omnillm connect chat               chat in this terminal with the models on your other computer
  omnillm connect ask "question"     one question
  omnillm connect models             which models it can use
  omnillm connect run                a local API for programs on this computer
  omnillm connect pair               enter a pairing text on this computer
  omnillm connect forget             delete the pairing here
  omnillm update                     update
  omnillm help --all                 every command
`)
	}
	return t(`用法：
  omnillm install                    一步装好：主机作为后台服务运行，打开控制台
  omnillm open                       打开本机控制台
  omnillm open --add                 打开控制台，直接到「添加设备」
  omnillm pair                       在终端里添加手机：显示二维码，手机扫码后核对 6 位数
  omnillm verify web [网页地址]      核对网页版和 GitHub 上签名发布的是否一致
  omnillm status                     查看主机状态
  omnillm update [--check] [--auto on|off]
                                     更新到最新版本（主机默认会在空闲时自己更新）；--auto 开关自动更新
  omnillm login claude|chatgpt|gemini
                                     用 Claude、ChatGPT 或 Gemini 订阅：没装官方程序会先装上，再在浏览器里登录
  omnillm uninstall [--purge]        停止并移除后台服务（--purge 连数据和 API Key 一起删）
  omnillm tray                       把菜单栏图标找回来
  omnillm version                    显示版本
  omnillm help --all                 所有命令（主机、设备、远程连接管理）
`, `Usage:
  omnillm install                    one step: run the hub as a background service, open the console
  omnillm open                       open the local console
  omnillm open --add                 open the console on Add device
  omnillm pair                       add a phone in the terminal: a QR code to scan, then check the 6 digits
  omnillm verify web [address]       check the web app against the signed GitHub release
  omnillm status                     show the hub’s status
  omnillm update [--check] [--auto on|off]
                                     update to the newest version (the hub also updates itself when idle); --auto turns that on or off
  omnillm login claude|chatgpt|gemini
                                     use your Claude, ChatGPT or Gemini subscription: installs the official program if needed, signs in in your browser
  omnillm uninstall [--purge]        stop and remove the background service (--purge also deletes data and API keys)
  omnillm tray                       bring back the menu bar icon
  omnillm version                    show the version
  omnillm help --all                 all commands (hub, devices, relay admin)
`)
}

// usage is for a command that isn't one: the short list, exit 2.
func usage() {
	fmt.Fprint(os.Stderr, usageText(false))
	os.Exit(2)
}

// subUsage is for a hub or connect command used wrongly: every command,
// so the right form is there to copy, on stderr, exit 2.
func subUsage() {
	fmt.Fprint(os.Stderr, usageText(true))
	os.Exit(2)
}

// helpCmd is `omnillm help [--all]` (and omnillm alone): the list, exit 0.
// guideCmd is a bare "omnillm": what is set up, the one thing to do next,
// and the few commands people look for, by what they want to do. The full
// list is omnillm help.
func guideCmd() {
	switch {
	case role() == "device" && !device.Paired(context.Background(), secrets(dataDir("device"), "omnillm-device")):
		fmt.Println(runnable(deviceNext())) // not paired, or the pairing can't be read
		return
	case role() == "device":
		fmt.Println(runnable(t("这台电脑用的是你另一台电脑上的 OmniLLM。\n\n想做什么：\n"+
			"  对话                 omnillm connect chat\n"+
			"  问一句               omnillm connect ask \"问题\"\n"+
			"  换一台电脑配对       omnillm connect forget，再 omnillm connect pair\n"+
			"  更新                 omnillm update\n"+
			"  全部命令             omnillm help --all",
			"This computer uses OmniLLM on another computer of yours.\n\nWhat do you want to do?\n"+
				"  Chat                        omnillm connect chat\n"+
				"  Ask one question            omnillm connect ask \"question\"\n"+
				"  Pair with another computer  omnillm connect forget, then omnillm connect pair\n"+
				"  Update                      omnillm update\n"+
				"  Every command               omnillm help --all")))
		return
	case role() == "" && !exists(dataDir("hub")):
		fmt.Println(t("OmniLLM 还没有装好。安装：curl -fsSL https://raw.githubusercontent.com/wensuper/OmniLLM/main/install.sh | sh\n全部命令：omnillm help",
			"OmniLLM isn’t set up yet. To install: curl -fsSL https://raw.githubusercontent.com/wensuper/OmniLLM/main/install.sh | sh\nEvery command: omnillm help"))
		return
	}
	st, ok := queryStatus()
	st.Running = ok
	st.Headless = headless()
	if ok {
		fmt.Println(runningLine(st))
	}
	fmt.Println(runnable(nextStep(st)))
	fmt.Println()
	console, connect := "", ""
	if !st.Headless {
		console = t("  打开控制台           omnillm open\n", "  Open the console         omnillm open\n")
	}
	// With no model yet, next already lists how to connect one.
	if st.Providers > 0 {
		connect = t("  接上订阅             omnillm login claude（或 chatgpt、gemini）\n"+
			"  接上 API Key         omnillm hub provider add anthropic（OpenAI 的 Key 换成 openai）\n",
			"  Connect a subscription   omnillm login claude (or chatgpt, gemini)\n"+
				"  Connect an API key       omnillm hub provider add anthropic (openai for an OpenAI key)\n")
	}
	fmt.Println(runnable(t("想做什么：\n"+
		"  加手机或另一台电脑   omnillm pair\n"+connect+
		"  看现在怎么样         omnillm status\n"+console+
		"  更新                 omnillm update\n"+
		"  全部命令             omnillm help --all",
		"What do you want to do?\n"+
			"  Add a phone or computer  omnillm pair\n"+connect+
			"  See how things are       omnillm status\n"+console+
			"  Update                   omnillm update\n"+
			"  Every command            omnillm help --all")))
}

// pairedNote is what a terminal says once a device is added: it is ready,
// and the three things worth remembering.
func pairedNote() string {
	return onMachine(t("新加的设备已经能用%s 上的模型了，在它上面直接对话就行。\n以后：再加手机 omnillm pair · 看状态 omnillm status · 忘了怎么用，输入 omnillm",
		"The new device can now use the models on %s; just chat on it.\nLater: add another phone with omnillm pair · see how things are with omnillm status · forgot how? type omnillm"))
}

func helpCmd(args []string) {
	all := len(args) > 0 && (args[0] == "--all" || args[0] == "-a" || args[0] == "all")
	fmt.Print(usageText(all))
}

func main() {
	log.SetFlags(0)
	adoptHome()
	// A data folder set with OMNILLM_HOME is checked for every command
	// (device ones too), not only at install and hub start.
	// uninstall and status still work with one that fails it (it may have
	// been accepted by an earlier version): they say so instead, and
	// uninstall then leaves that folder's contents alone.
	switch a := append(os.Args, "")[1]; a {
	case "help", "version", "--version", "-h", "--help":
	default:
		if _, err := privateHome(); err != nil {
			if a != "uninstall" && a != "status" {
				die(err)
			}
			fmt.Fprintln(os.Stderr, err)
			homeRejected = true
		}
	}
	if len(os.Args) < 2 {
		guideCmd()
		return
	}
	switch os.Args[1] {
	case "help", "-h", "--help":
		helpCmd(os.Args[2:])
	case "pair":
		// omnillm pair is omnillm hub pair: adding a phone is common enough
		// to deserve the short command.
		hubCmd(append([]string{"pair"}, os.Args[2:]...))
	case "hub":
		hubCmd(os.Args[2:])
	case "connect":
		connectCmd(os.Args[2:])
	case "relay":
		relayCmd(os.Args[2:])
	case "open":
		openCmd(os.Args[2:])
	case "install":
		installCmd(os.Args[2:])
	case "uninstall":
		uninstallCmd(os.Args[2:])
	case "status":
		statusCmd(os.Args[2:])
	case "update", "upgrade":
		updateCmd(os.Args[2:])
	case "login":
		loginCmd(os.Args[2:])
	case "verify":
		verifyCmd(os.Args[2:])
	case "tray":
		trayCmd()
	case "version":
		fmt.Println("omnillm " + version)
	default:
		fmt.Fprintf(os.Stderr, t("不认识命令 omnillm %s。\n\n", "Unknown command omnillm %s.\n\n"), os.Args[1])
		usage()
	}
}

// version is set at build time (-ldflags "-X main.version=…").
var version = "0.1.0-dev"

// openCmd is "omnillm open [--add]": the console in the browser; --add
// opens it on the QR code for adding a phone or another computer.
func openCmd(args []string) {
	fs := flag.NewFlagSet("open", flag.ContinueOnError)
	add := fs.Bool("add", false, "open on “Add device”")
	if err := parseFlags(fs, args); err != nil {
		dieArgs(err)
	}
	sock := filepath.Join(dataDir("hub"), "control.sock")
	var url string
	err := ctl(sock, control.Request{Op: "console.login"}, func(r control.Reply) (bool, bool) {
		url = r.Text
		return false, false
	})
	if err != nil {
		die(err)
	}
	if !strings.HasPrefix(url, "http://127.0.0.1:") {
		fmt.Fprintln(os.Stderr, notRunning())
		os.Exit(1)
	}
	if *add {
		url += "&add"
	}
	if err := openLogin(url); err != nil {
		// No screen here: what works on this machine instead of a link no
		// browser can open.
		if headless() {
			fmt.Println(runnable(t("这台机器没有屏幕，控制台在这里打不开。在它的终端里亲自运行：\n· 加手机或电脑：omnillm pair\n· 接上模型：omnillm login claude（或 chatgpt），或 omnillm hub provider add anthropic\n· 看状态：omnillm status",
				"This machine has no screen, so the console can’t open here. In its terminal, run yourself:\n· Add a phone or computer: omnillm pair\n· Connect a model: omnillm login claude (or chatgpt), or omnillm hub provider add anthropic\n· See how things are: omnillm status")))
			return
		}
		// The link is a console session: only for a person at this
		// terminal, never into an assistant's transcript.
		if notAPerson() || !term.IsTerminal(int(os.Stdout.Fd())) {
			fmt.Println(runnable(onMachine(t("这里打不开浏览器。请在%s 的终端里亲自运行 omnillm open。", "No browser opened here. In a terminal on %s, run omnillm open yourself."))))
			return
		}
		fmt.Print(onMachine(t("在 %s 的浏览器里打开（2 分钟内有效，只能用一次）：\n", "Open this in a browser on %s (valid 2 minutes, single use):\n")))
		fmt.Println(url)
		return
	}
	fmt.Println(t("已在浏览器里打开控制台。", "Opened the console in your browser."))
}

// openLogin opens the one-time login link without putting it on a command
// line (other users can read command lines on a shared machine): the link
// goes into a page readable only by this user, and the browser opens that.
func openLogin(url string) error {
	if os.Getenv("OMNILLM_NO_BROWSER") != "" {
		return errors.New("browser disabled")
	}
	page := filepath.Join(dataDir("hub"), "open.html")
	body := `<!doctype html><meta charset="utf-8"><meta name="referrer" content="no-referrer">` +
		`<link rel="icon" href="data:image/svg+xml;base64,` + base64.StdEncoding.EncodeToString(console.Icon()) + `">` +
		`<meta http-equiv="refresh" content="0;url=` + html.EscapeString(url) + `"><title>OmniLLM</title>`
	if err := secret.WriteFileAtomic(page, []byte(body)); err != nil {
		return err
	}
	return openBrowser("file://" + page)
}

func openBrowser(url string) error {
	if os.Getenv("OMNILLM_NO_BROWSER") != "" {
		return errors.New("browser disabled")
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
			return errors.New("no display")
		}
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

func dataDir(role string) string {
	if d := os.Getenv("OMNILLM_HOME"); d != "" {
		return filepath.Join(d, role)
	}
	base, err := os.UserConfigDir()
	if err != nil {
		die(err)
	}
	return filepath.Join(base, "OmniLLM", role)
}

// secrets: the system keychain, or an encrypted file when OMNILLM_MASTER_KEY
// is set (servers without a keychain).
func secrets(dir, service string) secret.Store {
	mk := os.Getenv("OMNILLM_MASTER_KEY")
	if f := os.Getenv("OMNILLM_MASTER_KEY_FILE"); mk == "" && f != "" {
		st, err := os.Stat(f)
		if err != nil {
			die(err)
		}
		if st.Mode().Perm()&0o077 != 0 || !fsown.Mine(st) {
			die(say(t("主密钥文件必须是你自己的，并且只有你能读（0600）：", "The master key file must be yours and readable only by you (0600): ") + f))
		}
		b, err := os.ReadFile(f)
		if err != nil {
			die(err)
		}
		mk = strings.TrimSpace(string(b))
	}
	if mk != "" {
		k, err := secret.ParseMasterKey(mk)
		if err != nil {
			die(err)
		}
		f, err := secret.NewFile(filepath.Join(dir, "secrets.enc"), k)
		if err != nil {
			die(err)
		}
		return f
	}
	kr := secret.NewKeyring(service)
	if runtime.GOOS == "linux" {
		// A server has no unlocked keyring. Use it when it answers;
		// otherwise the private key file beside the data, as the installed
		// hub does on Linux (secretEnv; as strong as the account's file
		// permissions, INSTALL.md says so).
		kf := filepath.Join(dir, "master.key")
		_, statErr := os.Stat(kf)
		if _, err := kr.Get(context.Background(), "omnillm/probe"); statErr != nil && errors.Is(err, secret.ErrNotFound) {
			return kr
		}
		if err := ensureMasterKey(kf); err != nil {
			die(err)
		}
		b, err := os.ReadFile(kf)
		if err != nil {
			die(err)
		}
		k, err := secret.ParseMasterKey(strings.TrimSpace(string(b)))
		if err != nil {
			die(err)
		}
		f, err := secret.NewFile(filepath.Join(dir, "secrets.enc"), k)
		if err != nil {
			die(err)
		}
		return f
	}
	return kr
}

func signalCtx() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}

// ---------------- hub ----------------

func hubCmd(args []string) {
	if len(args) == 0 {
		subUsage()
	}
	if isHelpWord(args[0]) {
		dieArgs(errHelp)
	}
	// A device-only machine has no hub: say so before anything else (an
	// assistant would otherwise relay "run omnillm pair here").
	// (hub run from a terminal too; only a service starts it otherwise.)
	if role() == "device" && (args[0] != "run" || os.Getenv("OMNILLM_DETACHED") != "1") {
		die(say(runnable(deviceHubHint())))
	}
	// -h anywhere after the command is the help, never a name or a value.
	if slices.ContainsFunc(args[1:], func(w string) bool { return w == "-h" || w == "-help" || w == "--help" }) {
		dieArgs(errHelp)
	}
	dir := dataDir("hub")
	sock := filepath.Join(dir, "control.sock")
	switch args[0] {
	case "run":
		hubRun(dir, sock, args[1:])
	case "provider":
		if len(args) == 2 && (args[1] == "rm" || args[1] == "off" || args[1] == "on") {
			dieArgs(usageErr(t("请写出模型来源的名字（omnillm hub providers 里能看到）。", "Name the model source (see omnillm hub providers).")))
		}
		if len(args) >= 3 && args[1] == "rm" {
			call(sock, control.Request{Op: "provider.rm", Args: map[string]string{"name": strings.Join(args[2:], " ")}})
			return
		}
		if len(args) >= 3 && (args[1] == "off" || args[1] == "on") {
			call(sock, control.Request{Op: "provider.off", Args: map[string]string{"name": strings.Join(args[2:], " "), "off": fmt.Sprint(args[1] == "off")}})
			return
		}
		if len(args) < 3 || args[1] != "add" || strings.HasPrefix(args[2], "-") {
			dieArgs(usageErr(t("用法：omnillm hub provider add <openai|anthropic|ollama|openai-compatible> | rm 名字 | off 名字 | on 名字（列出来源：omnillm hub providers）",
				"Usage: omnillm hub provider add <openai|anthropic|ollama|openai-compatible> | rm NAME | off NAME | on NAME (to list them: omnillm hub providers)")))
		}
		fs := flag.NewFlagSet("provider add", flag.ContinueOnError)
		name := fs.String("name", "", "")
		base := fs.String("base-url", "", "")
		if err := parseFlags(fs, args[3:]); err != nil {
			dieArgs(err)
		}
		kind := args[2]
		switch kind {
		case "openai", "anthropic", "ollama", "openai-compatible":
		case "claude", "claude-code", "chatgpt", "codex", "gemini", "google", "antigravity", "agy":
			dieArgs(say(runnable(t("订阅用 omnillm login claude、chatgpt 或 gemini 接入。", "Subscriptions connect with omnillm login claude, chatgpt or gemini."))))
		default:
			dieArgs(usageErr(t("用法：omnillm hub provider add <openai|anthropic|ollama|openai-compatible> [--name N] [--base-url U]",
				"Usage: omnillm hub provider add <openai|anthropic|ollama|openai-compatible> [--name N] [--base-url U]")))
		}
		if kind == "openai-compatible" && *base != "" {
			// The address alone (as https, with a fixed name): a server on
			// this computer or its network is refused before any key.
			probe := *base
			if u, err := url.Parse(probe); err == nil && u.Scheme == "http" {
				u.Scheme = "https"
				probe = u.String()
			}
			if err := (provider.Config{Kind: provider.OpenAICompatible, Name: "x", BaseURL: probe, NeedsKey: true}).Validate(); err != nil && strings.Contains(err.Error(), "not public") {
				dieArgs(say(runnable(t("openai-compatible 的来源必须在互联网上（https、公开地址）；这台电脑或局域网里的模型服务不支持。本机模型可以用 Ollama：omnillm hub provider add ollama。",
					"openai-compatible sources must be on the internet (https, a public address); a model server on this computer or your network isn't supported. Ollama is: omnillm hub provider add ollama."))))
			}
		}
		if kind == "openai-compatible" && (*name == "" || *base == "") {
			dieArgs(usageErr(t("用法：omnillm hub provider add openai-compatible --name 名字 --base-url https://…", "Usage: omnillm hub provider add openai-compatible --name NAME --base-url https://…")))
		}
		// The name and address as the hub will check them, before the key.
		if pre, err := provider.Default(provider.Kind(kind)); err == nil || kind == "openai-compatible" {
			if kind == "openai-compatible" {
				pre = provider.Config{Kind: provider.OpenAICompatible, NeedsKey: true}
			}
			if *name != "" {
				pre.Name = *name
			}
			if *base != "" {
				pre.BaseURL = *base
			}
			if err := pre.Validate(); err != nil {
				switch m := err.Error(); {
				case strings.Contains(m, "bad name"):
					dieArgs(usageErr(t("名字只能用英文字母、数字和 -（不要空格）。", "Use letters, digits and - for the name (no spaces).")))
				case strings.Contains(m, "https") || strings.Contains(m, "base URL") || strings.Contains(m, "scheme"):
					if kind != "ollama" {
						dieArgs(usageErr(t("地址要以 https:// 开头，例如 https://api.example.com/v1。", "The address starts with https://, e.g. https://api.example.com/v1.")))
					}
				}
			}
		}
		// Not running: say so before anyone pastes a key.
		if err := ctl(sock, control.Request{Op: "status"}, func(control.Reply) (bool, bool) { return false, false }); err != nil {
			die(err)
		}
		key := ""
		if kind != "ollama" {
			// A key reaches an assistant's command only through its chat:
			// it is typed in the person's own terminal. (A script piping
			// a key in, with no assistant around, is fine.)
			if installingAssistant() != "" || otherAssistant() {
				cmd := "omnillm hub provider add " + kind
				if kind == "openai-compatible" {
					name, base := *name, *base
					if name == "" {
						name = "NAME"
					}
					if base == "" {
						base = "https://…"
					}
					cmd += " --name " + name + " --base-url " + base
				}
				fmt.Println(runnable(t("API Key 由你自己输入：在这台机器上你自己的终端里运行 "+cmd+"，在提示时粘贴；不要贴进聊天。",
					"You enter the API key yourself: in your own terminal on this machine, run "+cmd+" and paste it when asked; don’t paste it into a chat.")))
				return
			}
			key = readSecret(t("粘贴 API Key（不会显示）：", "Paste the API key (not shown): "))
		}
		call(sock, control.Request{Op: "provider.add", Args: map[string]string{"kind": kind, "name": *name, "base_url": *base}, Secret: key})
		afterModel()
	case "devices", "apps", "models", "providers", "usage":
		// A table for people; --json, the hub's data as it is.
		fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
		raw := fs.Bool("json", false, "")
		var day *string
		if args[0] == "usage" {
			day = fs.String("day", "today", "")
		}
		if err := parseFlags(fs, args[1:]); err != nil {
			dieArgs(err)
		}
		req := control.Request{Op: args[0]}
		var render func([]byte) (string, error)
		switch args[0] {
		case "devices":
			render = func(b []byte) (string, error) { return devicesTable(b, time.Now()) }
		case "apps":
			render = appsTable
		case "models":
			render = modelsTable
		case "providers":
			render = func(b []byte) (string, error) { return providersTable(b, time.Now()) }
		case "usage":
			if !validDay(*day) {
				dieArgs(say(t("--day 只能是 today（今天）、yesterday（昨天）或 7d（最近 7 天）。", "--day must be today, yesterday or 7d (the last 7 days).")))
			}
			req.Args = map[string]string{"day": *day}
			render = func(b []byte) (string, error) { return usageTable(b, *day) }
		}
		if *raw {
			render = indentJSON
		}
		printTable(sock, req, render)
	case "audit":
		if len(args) > 1 {
			dieArgs(usageErr(fmt.Sprintf(t("多了“%s”。", "Unexpected “%s”."), args[1])))
		}
		empty := "" // a script gets the JSON as it is ([] when empty)
		if term.IsTerminal(int(os.Stdout.Fd())) {
			empty = t("今天还没有调用。", "No calls yet today.")
		}
		callOr(sock, control.Request{Op: args[0]}, empty)
	case "pair":
		// A hub that answers first: "your own step" would point at nothing.
		if err := ctl(sock, control.Request{Op: "status"}, func(control.Reply) (bool, bool) { return false, false }); err != nil {
			die(err)
		}
		a, err := pairArgs(args[1:])
		if err != nil {
			dieArgs(err)
		}
		if st, ok := queryStatus(); ok && !st.RelayConfigured {
			die(say(runnable(t("这台主机的远程连接关着（OMNILLM_RELAY=off），加不了设备。", "Remote access is off on this hub (OMNILLM_RELAY=off), so devices can't be added.") + sentenceGap() + relayOnText(lang()))))
		}
		if notAPerson() {
			cmd := strings.Join(append([]string{"omnillm pair"}, shellWords(args[1:])...), " ")
			fmt.Println(runnable(fmt.Sprintf(t("添加设备是你自己的一步：在这台机器上你自己的终端里运行 %s，扫码、核对 6 位数都在那里做。",
				"Adding a device is your own step: in your own terminal on this machine, run %s; the scan and the 6-digit check happen there."), cmd)))
			return
		}
		callOr(sock, control.Request{Op: "pair", Args: a}, "")
		if st, ok := queryStatus(); ok && (st.Providers == 0 || (st.ProvidersOn != nil && *st.ProvidersOn == 0)) {
			st.Running = true
			fmt.Println(runnable(nextStep(st)))
		} else {
			fmt.Println(runnable(pairedNote()))
		}
	case "revoke":
		// The name may be typed unquoted: "hub revoke iPhone 2".
		if len(args) < 2 {
			dieArgs(usageErr(t("请写出要移除的设备的名字或 ID。", "Name the device to remove (its name or ID).")))
		}
		call(sock, control.Request{Op: "revoke", Args: map[string]string{"id": strings.Join(args[1:], " ")}})
	case "device":
		if len(args) >= 2 && (args[1] == "rm" || args[1] == "remove") {
			dieArgs(usageErr(t("移除设备：omnillm hub revoke 名字", "To remove a device: omnillm hub revoke NAME")))
		}
		if len(args) < 2 || args[1] != "set" {
			dieArgs(usageErr(t("用法：omnillm hub device set 名字 [--name N] [--paused true|false] [--budget-usd X] [--models …]（列出设备：omnillm hub devices）",
				"Usage: omnillm hub device set NAME [--name N] [--paused true|false] [--budget-usd X] [--models …] (to list them: omnillm hub devices)")))
		}
		a, err := deviceSetArgs(splitTarget(args[2:]))
		if err != nil {
			dieArgs(err)
		}
		call(sock, control.Request{Op: "device.set", Args: a})
	case "app":
		if len(args) < 2 {
			dieArgs(usageErr(appUsage()))
		}
		if (args[1] == "add" || args[1] == "rm") && len(args) < 3 {
			dieArgs(usageErr(t("请写出应用的名字。", "Name the app.")))
		}
		switch args[1] {
		case "add":
			for _, w := range args[2:] {
				if strings.HasPrefix(w, "-") {
					dieArgs(usageErr(t("hub app add 只接名字；额度之后用 omnillm hub app set 名字 --budget-usd X 改。", "hub app add takes only a name; set its limit afterwards with omnillm hub app set NAME --budget-usd X.")))
				}
			}
			call(sock, control.Request{Op: "app.add", Args: map[string]string{"name": strings.Join(args[2:], " ")}})
		case "rm":
			call(sock, control.Request{Op: "app.rm", Args: map[string]string{"id": strings.Join(args[2:], " ")}})
		case "set":
			a, err := appSetArgs(splitTarget(args[2:]))
			if err != nil {
				dieArgs(err)
			}
			call(sock, control.Request{Op: "app.set", Args: a})
		default:
			dieArgs(usageErr(appUsage()))
		}
	case "model":
		if len(args) < 2 || args[1] != "set" {
			dieArgs(usageErr(t("用法：omnillm hub model set 名字 [--name 新名字] [--shown true|false] [--human-only true|false] [--auto true|false] [--price-in 输入价 --price-out 输出价]（列出模型：omnillm hub models）",
				"Usage: omnillm hub model set NAME [--name NEW-NAME] [--shown true|false] [--human-only true|false] [--auto true|false] [--price-in IN --price-out OUT] (to list them: omnillm hub models)")))
		}
		a, err := modelSetArgs(splitTarget(args[2:]))
		if err != nil {
			dieArgs(err)
		}
		call(sock, control.Request{Op: "model.set", Args: a})
	case "daily-total":
		v, err := dailyTotalArgs(args[1:])
		if err != nil {
			dieArgs(err)
		}
		call(sock, control.Request{Op: "daily_total.set", Args: map[string]string{"micro_usd": strconv.FormatInt(v, 10)}})
	case "region":
		if len(args) == 1 {
			dieArgs(usageErr(t("用法：omnillm hub region confirm（地区变了、模型暂停时，确认是你本人换了地方）", "Usage: omnillm hub region confirm (after the region changed and models paused: confirm you moved)")))
		}
		if len(args) != 2 || args[1] != "confirm" {
			dieArgs(badCommand(append([]string{"hub"}, args...)...))
		}
		if err := ctl(sock, control.Request{Op: "status"}, func(control.Reply) (bool, bool) { return false, false }); err != nil {
			die(err)
		}
		if st, ok := queryStatus(); ok && !st.RegionPaused {
			fmt.Println(t("地区没有变，不用确认。", "The region hasn’t changed; there’s nothing to confirm."))
			return
		}
		if notAPerson() {
			fmt.Println(runnable(t("确认换了地区是你自己的决定：在这台机器上你自己的终端里运行 omnillm hub region confirm。",
				"Confirming the new region is your own decision: in your own terminal on this machine, run omnillm hub region confirm.")))
			return
		}
		call(sock, control.Request{Op: "region.confirm"})
	default:
		dieArgs(badCommand("hub", args[0]))
	}
}

// validDay is a --day hub usage takes.
func validDay(d string) bool { return d == "today" || d == "yesterday" || d == "7d" }

// pairArgs reads `hub pair [--name N] [--models a,b] [--budget-usd 5]
// [--background-only]` as the pair request's arguments.
func pairArgs(args []string) (map[string]string, error) {
	fs := flag.NewFlagSet("pair", flag.ContinueOnError)
	name := fs.String("name", "", "")
	models := fs.String("models", "", "")
	budget := fs.String("budget-usd", "", "")
	bg := fs.Bool("background-only", false, "for servers and scripts: calls never count as a person waiting")
	noLimit := fs.Bool("no-limit", false, "")
	if err := parseFlags(fs, args); err != nil {
		return nil, err
	}
	a := map[string]string{"name": *name, "models": *models, "budget_usd": "0", "background_only": fmt.Sprint(*bg)}
	if *noLimit && *budget != "" {
		return nil, usageErr(t("--budget-usd 和 --no-limit 只能用一个。", "Use either --budget-usd or --no-limit, not both."))
	}
	if *noLimit {
		a["budget_usd"] = "nolimit"
	} else if *budget != "" {
		micro, err := dollars(lang(), *budget, false)
		if err != nil {
			return nil, err
		}
		a["budget_usd"] = strconv.FormatFloat(float64(micro)/1e6, 'f', -1, 64)
	}
	return a, nil
}

// indentJSON is --json: the hub's data, indented.
func indentJSON(b []byte) (string, error) {
	var out bytes.Buffer
	if err := json.Indent(&out, b, "", "  "); err != nil {
		return string(b), nil
	}
	return out.String(), nil
}

func readSecret(prompt string) string {
	fmt.Fprint(os.Stderr, prompt)
	if term.IsTerminal(int(os.Stdin.Fd())) {
		b, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			die(err)
		}
		return strings.TrimSpace(string(b))
	}
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		die(err)
	}
	return strings.TrimSpace(line)
}

func readLine(prompt string) string {
	fmt.Fprint(os.Stderr, prompt)
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(line)
}

func call(sock string, req control.Request) { callOr(sock, req, "") }

// printTable asks the hub for a list and prints it rendered for people.
func printTable(sock string, req control.Request, render func([]byte) (string, error)) {
	var data []byte
	err := ctl(sock, req, func(r control.Reply) (bool, bool) {
		if r.Event == "ok" {
			data = r.Data
		}
		return false, false
	})
	if err != nil {
		die(err)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		data = []byte("null")
	}
	out, err := render(data)
	if err != nil {
		die(err)
	}
	fmt.Println(out)
}

// callOr is call, printing empty instead of an empty list.
func callOr(sock string, req control.Request, empty string) {
	if err := stream(sock, req, empty); err != nil {
		die(err)
	}
}

// stream runs a hub command, printing what it says (texts, the QR code on
// a terminal, the owner's yes or no), and returns its error.
func stream(sock string, req control.Request, empty string) error {
	return ctl(sock, req, func(r control.Reply) (bool, bool) {
		switch r.Event {
		case "info", "ok", "done":
			if r.Text != "" {
				fmt.Println(runnable(stripControls(r.Text))) // its commands as this terminal can run them
			}
			if d := string(bytes.TrimSpace(r.Data)); empty != "" && (d == "null" || d == "[]") {
				fmt.Println(empty)
			} else if len(r.Data) > 0 {
				var out bytes.Buffer
				if json.Indent(&out, r.Data, "", "  ") == nil {
					fmt.Println(out.String())
				} else {
					fmt.Println(string(r.Data))
				}
			}
		case "qr":
			if term.IsTerminal(int(os.Stdout.Fd())) {
				if q, err := qrcode.Terminal(r.Text); err == nil {
					fmt.Print(q)
				}
			}
		case "confirm":
			ans := readLine(r.Text + " [y/N] ")
			return strings.EqualFold(ans, "y") || strings.EqualFold(ans, "yes") || ans == "是", true
		}
		return false, false
	})
}

func hubRun(dir, sock string, args []string) {
	log.SetFlags(log.LstdFlags) // a service log needs times
	fs := flag.NewFlagSet("hub run", flag.ExitOnError)
	relayURL := fs.String("relay", os.Getenv("OMNILLM_RELAY"), "relay URL (wss://…; default the project’s; off: none)")
	api := fs.String("api", envOr("OMNILLM_API", defaultAPI), "serve the local API for app tokens on this loopback address (off: none)")
	consoleAddr := fs.String("console", "127.0.0.1:8770", "serve the local console on this loopback address (empty: off)")
	webApp := fs.String("web-app", "", "web client the pairing QR code opens (default: the project’s)")
	_ = fs.Parse(args)
	apiSetting = *api
	// The data folder is checked here too, not only at install: the hub
	// keeps its secrets and state there.
	if _, err := privateHome(); err != nil {
		log.Fatal(err)
	}
	// One hub per data folder: a second one (say, run to watch the log)
	// would fight the service for the relay and take its socket.
	if c, err := net.DialTimeout("unix", sock, time.Second); err == nil {
		c.Close()
		if serviceInstalled() {
			stop := "launchctl bootout gui/" + strconv.Itoa(os.Getuid()) + "/" + launchdLabel
			if runtime.GOOS != "darwin" {
				stop = "systemctl --user stop " + systemdUnit
			}
			die(sayf(t("OmniLLM 已经在运行（后台服务）。它的日志在 %s；要在前台运行，先停掉后台服务：%s", "OmniLLM is already running (the background service). Its log is at %s; to run it in the foreground, stop the service first: %s"), filepath.Join(dir, "hub.log"), stop))
		}
		die(sayf(t("另一个 omnillm hub run 已经在用这个数据文件夹（%s），先停掉它。", "Another omnillm hub run is already using this data folder (%s); stop it first."), dir))
	}
	switch *relayURL {
	case "":
		*relayURL = hub.DefaultRelayURL
	case "off":
		*relayURL = ""
	}
	if *webApp == "" {
		*webApp = webAppFromEnv()
	} else {
		os.Setenv("OMNILLM_WEB_APP", *webApp)
		*webApp = webAppFromEnv()
	}
	ctx, stop := signalCtx()
	defer stop()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		die(err)
	}
	sec := secrets(dir, "omnillm-hub")
	st, err := store.OpenFile(filepath.Join(dir, "state.json"))
	if err != nil {
		die(err)
	}
	token := ""
	if *relayURL != "" {
		token = os.Getenv("OMNILLM_RELAY_TOKEN")
		if token == "" {
			b, err := sec.Get(ctx, "hub/relay_token")
			if errors.Is(err, secret.ErrNotFound) {
				b = []byte(hex.EncodeToString(e2e.RandomBytes(32)))
				if err := sec.Set(ctx, "hub/relay_token", b); err != nil {
					die(err)
				}
			} else if err != nil {
				die(err)
			}
			token = string(b)
		}
	}
	h, err := hub.New(ctx, hub.Options{Secrets: sec, Store: st, RelayURL: *relayURL, RelayToken: token, WebAppURL: *webApp, Lang: lang(),
		Logf: func(f string, a ...any) { log.Printf(f, a...) }})
	if err != nil {
		die(err)
	}
	if err := h.Start(ctx); err != nil {
		die(err)
	}
	ln, err := control.Listen(sock)
	var tooLong *control.PathTooLongError
	if errors.As(err, &tooLong) {
		fmt.Fprintln(os.Stderr, t("数据文件夹的路径太长，系统建不了控制通道。请把 OMNILLM_HOME 设成短一点的文件夹，比如 ~/.omnillm。",
			"The data folder path is too long for the control channel. Set OMNILLM_HOME to a shorter folder, such as ~/.omnillm."))
		os.Exit(1)
	}
	if err != nil {
		die(err)
	}
	// Who answers here: an install without a service manager stops this
	// process before it starts the new one (startDetached).
	pidFile := filepath.Join(dir, "hub.pid")
	_ = os.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())), 0o600)
	defer os.Remove(pidFile)
	var con *console.Server
	if *consoleAddr != "" {
		// Bind first: the console exists (and login links are issued) only
		// once its port is really ours.
		cln, err := console.Listen(*consoleAddr)
		if err != nil {
			log.Printf(t("控制台没能启动：%v", "Console didn’t start: %v"), err)
		} else {
			con, err = console.New(h, cln.Addr().String())
			if err != nil {
				die(err)
			}
			if runtime.GOOS == "darwin" {
				con.OpenLogin = openLoginWindow // elsewhere the page shows the command
			}
			log.Printf(t("控制台：http://%s/console/（用 omnillm open 打开）", "Console: http://%s/console/ (open it with omnillm open)"), cln.Addr())
			go func() {
				if err := con.ServeListener(ctx, cln); err != nil {
					log.Printf(t("控制台停止了：%v", "Console stopped: %v"), err)
				}
			}()
		}
	}
	go control.Serve(ctx, ln, hubHandler(h, *relayURL, con))
	// Subscriptions logged in on this machine since the last start
	// (Claude Code, Codex) connect by themselves.
	go func() {
		for _, cfg := range h.DetectSubscriptions(ctx) {
			if _, err := h.AddProvider(ctx, cfg, ""); err != nil {
				log.Printf("hub: %s not connected: %s", cfg.Name, omnillm.Describe("en", err))
			}
		}
	}()
	// The local API for programs on this Mac (the console's Apps): on by
	// default, loopback only, every request with an app token. Taken here so
	// the console shows the address only once it is really there.
	if con != nil {
		con.APIOff = *api == "" || *api == "off" || !validAPI(*api) // by its setting, not a busy port
	}
	if *api != "" && *api != "off" {
		if ln, err := localapi.Listen(*api); err != nil {
			if !validAPI(*api) {
				log.Printf(t("给应用用的本机接口的地址要写成 127.0.0.1:端口（端口 1–65535），%s 不行；应用暂时连不上。", "The local API for apps needs an address like 127.0.0.1:PORT (port 1–65535), not %s; apps can’t connect for now."), *api)
			} else {
				log.Printf(t("给应用用的本机接口没能启动（%s 多半被占了）；应用暂时连不上。", "The local API for apps did not start (%s is probably in use); apps can’t connect for now."), *api)
			}
		} else {
			if con != nil {
				con.APIURL = "http://" + ln.Addr().String() + "/v1"
			}
			go func() {
				if err := localapi.ServeListener(ctx, ln, localapi.Handler(localapi.HubBackend{Hub: h})); err != nil {
					log.Printf("local API: %v", err)
				}
			}()
		}
	}
	go autoUpdate(ctx, h, dir)
	// The page phones open is checked against the release that built it
	// (verifyweb.go): whoever hosts it can't change it unnoticed.
	if con != nil {
		con.WebCheck = func() any { return lastWebCheck(dir) }
	}
	go watchWeb(ctx, dir, *webApp)
	log.Print(t("OmniLLM 已启动。", "OmniLLM is running."))
	if *relayURL == "" {
		log.Print(t("远程连接关着（OMNILLM_RELAY=off），其他设备连不进来。", "Remote access is off (OMNILLM_RELAY=off), so other devices can’t connect.") + sentenceGap() + relayOnText(lang()))
	} else {
		sum := sha256.Sum256([]byte(token))
		log.Printf(t("远程连接登记令牌的哈希（自己部署 relay 时填进 --hub-token-hash）：%x", "Remote access registration token hash (for your own relay’s --hub-token-hash): %x"), sum)
	}
	<-ctx.Done()
	sctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = h.Stop(sctx)
	_ = os.Remove(sock)
}

func jsonData(v any) json.RawMessage {
	// An empty list is [] for scripts, never null.
	if rv := reflect.ValueOf(v); rv.Kind() == reflect.Slice && rv.IsNil() {
		return json.RawMessage("[]")
	}
	b, _ := json.MarshalIndent(v, "", "  ")
	return b
}

func hubHandler(h *hub.Hub, relayURL string, con *console.Server) control.Handler {
	return func(ctx context.Context, c *control.Conn, req control.Request) error {
		a := req.Args
		// Every sentence back is in the caller's language.
		l := callerLang(req)
		switch req.Op {
		case "ollama.auto", "providers.auto":
			// Subscriptions the owner is already logged in to on this
			// machine (Claude Code, Codex) connect with no key at all.
			for _, cfg := range h.DetectSubscriptions(ctx) {
				if _, err := h.AddProvider(ctx, cfg, ""); err != nil {
					log.Printf("hub: %s not connected: %s", cfg.Name, omnillm.Describe("en", err))
				}
			}
			ps, _ := h.Providers(ctx)
			if len(ps) > 0 {
				return c.Send(control.Reply{Event: "ok"})
			}
			if ids, err := h.DetectOllama(ctx); err == nil && len(hub.ChatModels(ids)) > 0 {
				cfg, _ := provider.Default(provider.Ollama)
				if _, err := h.AddProvider(ctx, cfg, ""); err != nil {
					return hubSay(l, err)
				}
			}
			return c.Send(control.Reply{Event: "ok"})
		case "subscription.state":
			// One subscription's state, for the installer: is the
			// assistant running it already usable here?
			for _, s := range h.SubscriptionStates(ctx) {
				if s.Kind == a["kind"] {
					return c.Send(control.Reply{Event: "ok", Data: jsonData(s)})
				}
			}
			return say(tl(l, "不认识这种订阅。", "Unknown subscription."))
		case "subscription.connect":
			// After omnillm login: connect it now, even if it was
			// removed before (logging in again is asking for it).
			if err := h.ConnectSubscription(ctx, a["kind"]); err != nil {
				return hubSay(l, err)
			}
			name := tl(l, "Claude 订阅", "Claude subscription")
			switch provider.Kind(a["kind"]) {
			case provider.Codex:
				name = tl(l, "ChatGPT 订阅", "ChatGPT subscription")
			case provider.Antigravity:
				name = tl(l, "Gemini 订阅（Antigravity）", "Gemini subscription (Antigravity)")
			}
			return c.Send(control.Reply{Event: "ok", Text: fmt.Sprintf(tl(l, "已接入 %s：手机和其他设备现在就能用。", "Connected your %s: your phone and other devices can use it now."), name)})
		case "console.login":
			if con == nil {
				return say(tl(l, "控制台没有开启。", "The console is off."))
			}
			return c.Send(control.Reply{Event: "ok", Text: con.LoginURL()})
		case "status":
			// With nothing connected, look for logged-in subscriptions, at
			// most once a minute: it runs the programs, and status is polled.
			if ps, _ := h.Providers(ctx); len(ps) == 0 && time.Since(time.Unix(0, lastDetect.Load())) > time.Minute {
				lastDetect.Store(time.Now().UnixNano())
				for _, cfg := range h.DetectSubscriptions(ctx) {
					_, _ = h.AddProvider(ctx, cfg, "")
				}
			}
			ps, _ := h.Providers(ctx)
			ds, _ := h.Devices(ctx)
			online := 0
			for _, d := range ds {
				if d.Online {
					online++
				}
			}
			var problems []SourceProblem
			for _, p := range ps {
				if p.Problem != "" && !p.Off {
					problems = append(problems, SourceProblem{Name: p.Name, Kind: p.Kind, DisplayName: p.DisplayName, Problem: p.Problem, ResetsAt: p.ResetsAt})
				}
			}
			_, _, paused := h.RegionStatus()
			on := 0
			for _, p := range ps {
				if !p.Off {
					on++
				}
			}
			st := map[string]any{"running": true, "version": version, "providers": len(ps), "providers_on": on, "devices": len(ds), "online": online,
				"relay_configured": relayURL != "", "relay_up": h.RelayUp(), "console": con != nil, "os": runtime.GOOS,
				"region_paused": paused, "problems": problems, "lang": h.UILang(ctx), "web_check": lastWebCheck(dataDir("hub"))}
			return c.Send(control.Reply{Event: "ok", Data: jsonData(st)})
		case "provider.add":
			cfg, err := provider.Default(provider.Kind(a["kind"]))
			if a["kind"] == string(provider.OpenAICompatible) {
				cfg, err = provider.Config{Kind: provider.OpenAICompatible, NeedsKey: true}, nil
			}
			if err != nil {
				return say(tl(l, "不认识这种模型来源。可以接入：openai、anthropic、ollama、openai-compatible。",
					"Unknown model source kind. You can connect: openai, anthropic, ollama, openai-compatible."))
			}
			if a["name"] != "" {
				cfg.Name = a["name"]
			}
			if a["base_url"] != "" {
				cfg.BaseURL = a["base_url"]
				// Ollama's own docs give the address without /v1.
				if cfg.Kind == provider.Ollama {
					if u, err := url.Parse(cfg.BaseURL); err == nil && (u.Path == "" || u.Path == "/") {
						cfg.BaseURL = strings.TrimSuffix(cfg.BaseURL, "/") + "/v1"
					}
				}
			}
			ms, err := h.AddProvider(ctx, cfg, req.Secret)
			if err != nil && cfg.Kind != provider.Ollama && omnillm.AsError(err).Msg == omnillm.MsgUpstream && nothingAnswers(err) {
				return say(strings.Replace(tl(l, "没有服务在 URL 应答；检查地址后再运行一次。", "Nothing answers at URL; check the address and run the command again."), "URL", cfg.BaseURL, 1))
			}
			if err != nil {
				// The key was just typed: check it, rather than "replace it".
				if cfg.Kind == provider.Ollama && omnillm.AsError(err).Msg == omnillm.MsgBadSource {
					if strings.Contains(omnillm.AsError(err).Detail, "this machine") {
						return say(tl(l, "OmniLLM 只用这台电脑上的 Ollama（http://127.0.0.1:11434）；别的电脑上的 Ollama 不支持。", "OmniLLM only uses an Ollama on this computer (http://127.0.0.1:11434); an Ollama on another computer isn't supported."))
					}
					return say(tl(l, "地址以 http:// 开头，例如 http://127.0.0.1:11434。", "The address starts with http://, e.g. http://127.0.0.1:11434."))
				}
				if cfg.Kind == provider.Ollama && omnillm.AsError(err).Msg == omnillm.MsgUpstream && !nothingAnswers(err) {
					if strings.HasSuffix(strings.TrimSuffix(cfg.BaseURL, "/"), "/v1") {
						return say(strings.Replace(tl(l, "URL 上应答的不是 Ollama；检查端口（Ollama 在 http://127.0.0.1:11434）。", "Something else answers at URL, not Ollama; check the port (Ollama listens on http://127.0.0.1:11434)."), "URL", cfg.BaseURL, 1))
					}
					return say(tl(l, "那个地址有东西应答，但不是 Ollama 的接口；地址要以 /v1 结尾（默认 http://127.0.0.1:11434/v1）。", "Something answers at "+cfg.BaseURL+", but not Ollama’s API; the address ends in /v1 (default http://127.0.0.1:11434/v1)."))
				}
				if cfg.Kind == provider.Ollama && omnillm.AsError(err).Msg == omnillm.MsgUpstream {
					return say(strings.Replace(onMachineIn(l, tl(l, "%s 上的 Ollama 没有在运行（URL 没有应答）。先打开 Ollama，再运行一次这条命令。", "Ollama isn’t running on %s (nothing answers at URL). Start Ollama, then run the command again.")), "URL", cfg.BaseURL, 1))
				}
				if omnillm.AsError(err).Msg == omnillm.MsgCredentialBad {
					return say(tl(l, "这个 Key 用不了（可能输错了、已作废或余额不足）。到服务商网站核对后再运行一次。",
						"That key didn’t work (mistyped, revoked or out of credit). Check it on the provider’s site and run the command again."))
				}
				return hubSay(l, err)
			}
			return c.Send(control.Reply{Event: "ok", Text: connectedText(l, cfg.Name, len(ms))})
		case "providers":
			ps, err := h.Providers(ctx)
			if err != nil {
				return hubSay(l, err)
			}
			return c.Send(control.Reply{Event: "ok", Data: jsonData(ps)})
		case "models":
			// With the owner's names and choices, as the console shows them.
			ms, err := h.OwnerModels(ctx)
			if err != nil {
				return hubSay(l, err)
			}
			return c.Send(control.Reply{Event: "ok", Data: jsonData(ms)})
		case "devices":
			ds, err := h.Devices(ctx)
			if err != nil {
				return hubSay(l, err)
			}
			ms, _ := h.OwnerModels(ctx)
			spent := spentToday(ctx, h)
			var rows []deviceRow
			for _, d := range ds {
				b, on, sp := d.Budget, d.Online, spent["device/"+d.ID]
				n := usableModels(ms, d.Models, true, d.BackgroundOnly)
				rows = append(rows, deviceRow{ID: d.ID, Name: d.Name, Budget: limitTextIn(l, d.Budget), BudgetMicroUSD: &b, Paused: d.Paused,
					Created: d.Created, LastSeen: d.LastSeen, Models: d.Models, Usable: &n, Online: &on, SpentMicroUSD: &sp})
			}
			return c.Send(control.Reply{Event: "ok", Data: jsonData(rows)})
		case "apps":
			as, err := h.Apps(ctx)
			if err != nil {
				return hubSay(l, err)
			}
			ms, _ := h.OwnerModels(ctx)
			spent := spentToday(ctx, h)
			var rows []appRow
			for _, x := range as {
				b, sp := x.Budget, spent["app/"+x.ID]
				n := usableModels(ms, x.Models, false, true)
				rows = append(rows, appRow{ID: x.ID, Name: x.Name, Budget: limitTextIn(l, x.Budget), BudgetMicroUSD: &b, Paused: x.Paused,
					Models: x.Models, Usable: &n, SpentMicroUSD: &sp})
			}
			return c.Send(control.Reply{Event: "ok", Data: jsonData(rows)})
		case "usage":
			var u []hub.Spend
			var err error
			switch a["day"] {
			case "", "today":
				u, err = h.Usage(ctx, "")
			case "yesterday":
				u, err = h.Usage(ctx, h.Day(1))
			case "7d":
				u, err = h.UsageBetween(ctx, h.Day(6), h.Day(0))
			default:
				return say(tl(l, "--day 只能是 today（今天）、yesterday（昨天）或 7d（最近 7 天）。", "--day must be today, yesterday or 7d (the last 7 days)."))
			}
			if err != nil {
				return hubSay(l, err)
			}
			return c.Send(control.Reply{Event: "ok", Data: jsonData(u)})
		case "audit":
			u, err := h.Audit(ctx, "")
			if err != nil {
				return hubSay(l, err)
			}
			return c.Send(control.Reply{Event: "ok", Data: jsonData(u)})
		case "device.set":
			ds, err := h.Devices(ctx)
			if err != nil {
				return hubSay(l, err)
			}
			d, err := findDevice(l, ds, a["device"])
			if err != nil {
				return err
			}
			name, budget, paused, models, err := deviceUpdate(l, a)
			if err != nil {
				return err
			}
			if models, err = modelIDs(ctx, h, l, models); err != nil {
				return err
			}
			d, err = h.UpdateDevice(ctx, d.ID, name, budget, paused, models)
			if err != nil {
				return hubSay(l, err)
			}
			text := fmt.Sprintf(tl(l, "已更新“%s”。", "Updated “%s”."), d.Name)
			if ms, err := h.OwnerModels(ctx); err == nil && usableModels(ms, d.Models, true, d.BackgroundOnly) == 0 {
				switch {
				case len(ms) == 0:
					text += tl(l, "还没有接入任何模型。", " No model is connected yet.")
				case d.BackgroundOnly && usableModels(ms, d.Models, true, false) > 0:
					text += tl(l, "它只做后台调用，而这些模型只能手动对话。", " It runs in the background only, and these models are chat-only.")
				default:
					text += tl(l, "它现在一个模型都用不了（给它的模型在设备上隐藏了，或来源已关闭）。", " It can't use any model now (the ones it may use are hidden from devices or switched off).")
				}
			}
			return c.Send(control.Reply{Event: "ok", Text: text})
		case "revoke":
			// By name or ID, as device set finds it.
			ds, err := h.Devices(ctx)
			if err != nil {
				return hubSay(l, err)
			}
			d, err := findDevice(l, ds, a["id"])
			if err != nil {
				return err
			}
			if err := h.Revoke(ctx, d.ID); err != nil {
				return hubSay(l, err)
			}
			return c.Send(control.Reply{Event: "ok", Text: fmt.Sprintf(tl(l, "已移除“%s”，它会立刻用不了。", "Removed “%s”; it stops working at once."), d.Name)})
		case "app.add":
			app, tok, err := h.AddApp(ctx, a["name"], nil, 0)
			if err != nil {
				return hubSay(l, err)
			}
			text := fmt.Sprintf(tl(l,
				"已为“%s”创建令牌（只显示这一次，请保存好）：\n%s\n这个应用只能做后台调用，每天额度 %s。",
				"Token for “%s” (shown once, keep it safe):\n%s\nThis app makes background calls only; daily limit %s."), app.Name, tok, usd(app.Budget))
			if con != nil && con.APIURL != "" {
				text += "\n" + fmt.Sprintf(tl(l, "接口地址（兼容 OpenAI）：%s，令牌放在 Authorization: Bearer 里。", "Address (OpenAI-compatible): %s; send the token as Authorization: Bearer."), con.APIURL)
			} else {
				if apiSetting == "off" {
					text += "\n" + tl(l, "给应用用的本机接口被设置关掉了（OMNILLM_API=off），这个令牌暂时用不了。要打开：运行 OMNILLM_API="+defaultAPI+" omnillm install（在前台运行的，不带 OMNILLM_API=off 重新启动）。", "The local API for apps is turned off (OMNILLM_API=off), so this token can’t be used yet. To turn it on, run OMNILLM_API="+defaultAPI+" omnillm install (a hub run in the foreground: start it again without OMNILLM_API=off).")
				} else if !validAPI(apiSetting) {
					text += "\n" + fmt.Sprintf(tl(l, "给应用用的本机接口的地址要写成 127.0.0.1:端口（端口 1–65535），%s 不行，这个令牌暂时用不了。", "The local API for apps needs an address like 127.0.0.1:PORT (port 1–65535), not %s, so this token can’t be used yet."), apiSetting)
				} else {
					text += "\n" + fmt.Sprintf(tl(l, "给应用用的本机接口现在没打开（多半是 %s 被占了），这个令牌暂时用不了。", "The local API for apps isn’t running (%s is probably in use), so this token can’t be used yet."), apiSetting)
				}
			}
			return c.Send(control.Reply{Event: "ok", Text: text})
		case "app.rm":
			as, err := h.Apps(ctx)
			if err != nil {
				return hubSay(l, err)
			}
			app, err := findApp(l, as, a["id"])
			if err != nil {
				return err
			}
			if err := h.RevokeApp(ctx, app.ID); err != nil {
				return hubSay(l, err)
			}
			return c.Send(control.Reply{Event: "ok", Text: fmt.Sprintf(tl(l, "已移除“%s”。", "Removed “%s”."), app.Name)})
		case "app.set":
			as, err := h.Apps(ctx)
			if err != nil {
				return hubSay(l, err)
			}
			app, err := findApp(l, as, a["app"])
			if err != nil {
				return err
			}
			name, budget, paused, models, err := deviceUpdate(l, a)
			if err != nil {
				return err
			}
			if models, err = modelIDs(ctx, h, l, models); err != nil {
				return err
			}
			app, err = h.UpdateApp(ctx, app.ID, name, budget, paused, models)
			if err != nil {
				return hubSay(l, err)
			}
			return c.Send(control.Reply{Event: "ok", Text: fmt.Sprintf(tl(l, "已更新“%s”。", "Updated “%s”."), app.Name)})
		case "model.set":
			// --name and --shown are the console's rename and "show on
			// devices"; the rest changes the model itself.
			ms, err := h.OwnerModels(ctx)
			if err != nil {
				return hubSay(l, err)
			}
			m, err := findModel(l, ms, a["model"])
			if err != nil {
				return err
			}
			name, shown, u, err := modelUpdate(l, a)
			if err != nil {
				return err
			}
			if u != (hub.ModelUpdate{}) {
				if err := h.SetModel(ctx, m.ID, u); err != nil {
					return hubSay(l, err)
				}
			}
			if name != nil || shown != nil {
				if err := h.SetModelPref(ctx, m.ID, name, shown); err != nil {
					return hubSay(l, err)
				}
			}
			if v, ok := a["auto"]; ok {
				if err := h.SetModelAuto(ctx, m.ID, v == "true"); err != nil {
					return hubSay(l, err)
				}
			}
			return c.Send(control.Reply{Event: "ok", Text: tl(l, "已更新。", "Updated.")})
		case "provider.rm":
			ps, err := h.Providers(ctx)
			if err != nil {
				return hubSay(l, err)
			}
			p, err := findProvider(l, ps, a["name"])
			if err != nil {
				return err
			}
			if err := h.RemoveProvider(ctx, p.Name); err != nil {
				return hubSay(l, err)
			}
			text := fmt.Sprintf(tl(l, "已移除“%s”，它的模型不再能用。", "Removed “%s”; its models can’t be used any more."), sourceNameIn(l, p))
			if p.KeyLast4 != "" {
				text += tl(l, "保存的 API Key 也删了；只想停用的话，下次用 omnillm hub provider off 名字。", " Its saved API key is deleted too; to only stop using a source, use omnillm hub provider off NAME next time.")
			}
			return c.Send(control.Reply{Event: "ok", Text: text})
		case "provider.off":
			ps, err := h.Providers(ctx)
			if err != nil {
				return hubSay(l, err)
			}
			p, err := findProvider(l, ps, a["name"])
			if err != nil {
				return err
			}
			off := a["off"] == "true"
			if err := h.SetProviderOff(ctx, p.Name, off); err != nil {
				return hubSay(l, err)
			}
			if off {
				return c.Send(control.Reply{Event: "ok", Text: fmt.Sprintf(tl(l, "已关闭“%s”：设备和应用暂时用不了它的模型，登录和 Key 都保留。", "Switched off “%s”: devices and apps can’t use its models for now; its sign-in or key is kept."), sourceNameIn(l, p))})
			}
			return c.Send(control.Reply{Event: "ok", Text: fmt.Sprintf(tl(l, "已打开“%s”，它的模型又能用了。", "Switched on “%s”; its models can be used again."), sourceNameIn(l, p))})
		case "ui_lang.set":
			// The menu bar's 中文 / English; the console sets the same one.
			if v := a["lang"]; v != "" && v != "zh" && v != "en" {
				return say(tl(l, "语言只能是 zh 或 en。", "The language must be zh or en."))
			}
			if err := h.SetUILang(ctx, a["lang"]); err != nil {
				return hubSay(l, err)
			}
			return c.Send(control.Reply{Event: "ok"})
		case "daily_total.set":
			v, err := strconv.ParseInt(a["micro_usd"], 10, 64)
			if err != nil || (v <= 0 && v != hub.NoLimit) || v > maxMicroUSD {
				return say(tl(l, "每天总额度不对。", "That daily total isn’t valid."))
			}
			if err := h.SetDailyTotal(ctx, v); err != nil {
				return hubSay(l, err)
			}
			return c.Send(control.Reply{Event: "ok", Text: dailyTotalText(l, v)})
		case "region.confirm":
			if _, seen, paused := h.RegionStatus(); !paused && seen == "" {
				return c.Send(control.Reply{Event: "ok", Text: tl(l, "地区没有变，不用确认。", "The region hasn’t changed; there’s nothing to confirm.")})
			}
			if err := h.ConfirmRegion(ctx); err != nil {
				return hubSay(l, err)
			}
			return c.Send(control.Reply{Event: "ok", Text: tl(l, "已确认，模型恢复使用。", "Confirmed; models are back on.")})
		case "pair":
			if relayURL == "" {
				return say(tl(l, "这台主机的远程连接关着（OMNILLM_RELAY=off），加不了设备。", "Remote access is off on this hub (OMNILLM_RELAY=off), so devices can't be added.") + tl(l, "", " ") + relayOnText(l))
			}
			return pairOp(ctx, h, c, l, a)
		}
		return say(tl(l, "OmniLLM 不认识这个命令。运行 omnillm install 更新到同一个版本。", "OmniLLM doesn’t know this command. Run omnillm install to update it to the same version."))
	}
}

// lastDetect is when the status op last looked for logged-in subscriptions.
var lastDetect atomic.Int64

// spentToday is today's spend per device or app ("device/<id>").
func spentToday(ctx context.Context, h *hub.Hub) map[string]int64 {
	spent := map[string]int64{}
	if u, err := h.Usage(ctx, ""); err == nil {
		for _, s := range u {
			spent[s.Who] = s.MicroUSD
		}
	}
	return spent
}

// usableModels counts the models a device or app can use now: the ones
// it is allowed (all when allowed is empty) that are still there; a
// device never gets the models kept off devices, and a program (an app,
// a background-only device) never gets chat-only ones.
func usableModels(ms []hub.ModelView, allowed []string, device, background bool) int {
	n := 0
	for _, m := range ms {
		switch {
		case len(allowed) > 0 && !slices.Contains(allowed, m.ID):
		case m.Hidden && device: // kept off devices
		case m.HumanOnly && background:
		case m.SourceOff:
		default:
			n++
		}
	}
	return n
}

// modelIDs turns the models device set / app set were given, by name or
// ID, into IDs; an empty list (every model) stays as it is.
func modelIDs(ctx context.Context, h *hub.Hub, l string, models *[]string) (*[]string, error) {
	if models == nil || len(*models) == 0 {
		return models, nil
	}
	ms, err := h.OwnerModels(ctx)
	if err != nil {
		return nil, hubSay(l, err)
	}
	ids := make([]string, 0, len(*models))
	for _, q := range *models {
		m, err := findModel(l, ms, q)
		if err != nil {
			return nil, err
		}
		if !slices.Contains(ids, m.ID) {
			ids = append(ids, m.ID)
		}
	}
	return &ids, nil
}

// connectedText is what provider add says: "Connected groq: 1 model
// available."
func connectedText(l, name string, n int) string {
	if l == "zh" {
		return fmt.Sprintf("已接入 %s，%d 个模型可用。", name, n)
	}
	return fmt.Sprintf("Connected %s: %s available.", name, plural(n, "model"))
}

// dailyTotalText is what daily-total says it set.
func dailyTotalText(l string, v int64) string {
	amount := usd(v)
	if v == hub.NoLimit {
		amount = tl(l, "不设上限", "no limit")
	}
	return fmt.Sprintf(onMachineIn(l, tl(l, "%s 每天的总额度：%%s。", "Daily total for %s: %%s.")), amount)
}

func usd(micro int64) string {
	cents := (micro + 5_000) / 10_000 // rounded to the cent
	if micro < 0 {
		cents = (micro - 5_000) / 10_000
	}
	sign := ""
	if cents < 0 {
		sign, cents = "-", -cents
	}
	return fmt.Sprintf("%s$%s.%02d", sign, thousands(cents/100), cents%100)
}

func pairOp(ctx context.Context, h *hub.Hub, c *control.Conn, l string, a map[string]string) error {
	var models *[]string
	var ms []string
	for _, m := range strings.Split(a["models"], ",") {
		if m = strings.TrimSpace(m); m != "" {
			ms = append(ms, m)
		}
	}
	if len(ms) > 0 {
		var err error
		if models, err = modelIDs(ctx, h, l, &ms); err != nil {
			return err
		}
	}
	budget := int64(0)
	if v := strings.TrimSpace(a["budget_usd"]); v == "nolimit" {
		budget = hub.NoLimit
	} else if v != "" && v != "0" {
		b, err := dollars(l, v, false)
		if err != nil {
			return err
		}
		budget = b
	}
	opts := hub.PairOptions{Name: a["name"], Budget: budget, BackgroundOnly: a["background_only"] == "true",
		Confirm: func(ctx context.Context, r hub.PairRequest) (bool, error) {
			return c.Ask(ctx, fmt.Sprintf(tl(l, "设备“%s”请求加入。它屏幕上显示的数字是 %s 吗？一致才确认。",
				"Device “%s” wants to join. Does its screen show %s? Confirm only if it matches."), r.DeviceName, r.Code))
		}}
	if models != nil {
		opts.Models = *models
	}
	p, err := h.Pair(ctx, opts)
	if err != nil {
		return hubSay(l, err)
	}
	// If this command ends early (the owner pressed Ctrl-C, the terminal
	// closed), the code dies with it instead of holding a slot for 5 min.
	defer p.Cancel()
	if err := c.Send(control.Reply{Event: "qr", Text: p.Link}); err != nil {
		return err
	}
	// How long, not a clock time: a server's clock is often in UTC.
	mins := int(math.Ceil(time.Until(p.Expires).Minutes()))
	expires := fmt.Sprintf(tl(l, "%d 分钟后过期", "expires in %d minutes"), mins)
	if a["phone"] == "true" {
		// Right after an install or sign-in on a server: only the phone.
		if err := c.Send(control.Reply{Event: "info", Text: fmt.Sprintf(tl(l, "（二维码%s · 只能用一次。）", "(The QR code %s · single use.)"), expires)}); err != nil {
			return err
		}
	} else if err := c.Send(control.Reply{Event: "info", Text: fmt.Sprintf(tl(l,
		"手机或平板：用相机扫上面的二维码。电脑或服务器：在那台机器上运行 curl -fsSL https://raw.githubusercontent.com/wensuper/OmniLLM/main/install.sh | sh -s -- --device（已经装了 OmniLLM 就运行 omnillm connect pair），在提示时粘贴下面这段配对文字（%s，只能用一次；不要发给别人，也不要贴到聊天里）：\n\n%s\n",
		"Phone or tablet: scan the QR code above with the camera. Computer or server: on it, run curl -fsSL https://raw.githubusercontent.com/wensuper/OmniLLM/main/install.sh | sh -s -- --device (or omnillm connect pair if OmniLLM is already there) and paste this pairing text when asked (it %s and works once; don’t send it to anyone or paste it into a chat):\n\n%s\n"),
		expires, p.Code)}); err != nil {
		return err
	}
	d, err := p.Wait(ctx)
	if err != nil {
		return hubSay(l, err)
	}
	return c.Send(control.Reply{Event: "done", Text: fmt.Sprintf(tl(l, "已添加设备“%s”，每天额度 %s。", "Added “%s”, daily limit %s."), d.Name, limitTextIn(l, d.Budget))})
}

// ---------------- connect (device) ----------------

func connectCmd(args []string) {
	if len(args) > 0 && isHelpWord(args[0]) {
		dieArgs(errHelp)
	}
	if len(args) == 0 {
		if role() == "device" {
			fmt.Fprint(os.Stderr, usageText(false)) // the device's commands
			os.Exit(2)
		}
		subUsage()
	}
	dir := dataDir("device")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		die(err)
	}
	sec := secrets(dir, "omnillm-device")
	ctx, stop := signalCtx()
	defer stop()
	onDevice = true
	switch args[0] {
	case "pair":
		paired, err := device.CheckPaired(ctx, sec)
		if err != nil {
			die(sayf(t("读不出这台设备保存的配对（%s）。如果它已经配对过，请用配对时的同一个用户运行；确实要重新配对，先运行 omnillm connect forget。", "Can’t read this device’s saved pairing (%s). If it is already paired, run as the same user it was paired as; to pair again, run omnillm connect forget first."), dir))
		}
		if paired {
			die(say(runnable(t("这台设备已经配对过（"+dir+"）。要换一台电脑或重新配对，先运行 omnillm connect forget。", "This device is already paired ("+dir+"). To pair again (or with another computer), run omnillm connect forget first."))))
		}
		if notAPerson() {
			fmt.Println(runnable(unpairedNext()))
			return
		}
		code := readSecret(t("粘贴配对文字（在 OmniLLM 控制台“添加设备”里复制，或在装着 OmniLLM 的电脑上运行 omnillm pair；不会显示，粘贴一次后回车）：", "Paste the pairing text (copy it from “Add device” in the OmniLLM console, or run omnillm pair on the computer running OmniLLM; not shown, paste once and press Return): "))
		host, _ := os.Hostname()
		info, err := device.Pair(ctx, device.Options{Secrets: sec}, code, host, func(sas string) {
			if len(sas) == 6 {
				sas = sas[:3] + " " + sas[3:]
			}
			fmt.Printf(t("\n请核对：另一边（你的 Mac 或服务器）显示的数字应该是 %s，一致就在那边确认（在控制台里点“一致，添加”，或在终端里输入 y）。\n", "\nCheck: the other side (your Mac or server) should show %s. If it does, confirm there (“Matches, add” in the console, or type y in the terminal).\n"), sas)
		})
		if err != nil {
			die(err)
		}
		// The other side may be a Mac or a server: not known here yet.
		fmt.Printf(t("已添加：这台电脑现在能用你另一台电脑上的模型（那边叫它“%s”）。\n", "Added: this computer can now use the models on your other computer, which lists it as “%s”.\n"), info.Name)
	case "run":
		fs := flag.NewFlagSet("connect run", flag.ContinueOnError)
		listen := fs.String("listen", "127.0.0.1:8766", "")
		if err := parseFlags(fs, args[1:]); err != nil {
			dieArgs(err)
		}
		if !validAPI(*listen) || *listen == "off" {
			dieArgs(usageErr(t("--listen 要写成这台电脑上的 127.0.0.1:端口（端口 1–65535）。", "--listen is 127.0.0.1:PORT on this computer (port 1–65535).")))
		}
		d, err := device.Open(ctx, device.Options{Secrets: sec, Logf: func(f string, a ...any) { log.Printf(f, a...) }})
		if err != nil {
			openFailed(ctx, sec, dir, err)
		}
		defer d.Close()
		tokPath := filepath.Join(dir, "local.token")
		tok, err := os.ReadFile(tokPath)
		if err != nil {
			tok = []byte("omni_local_" + hex.EncodeToString(e2e.RandomBytes(32)))
			if err := secret.WriteFileAtomic(tokPath, tok); err != nil {
				die(err)
			}
		}
		// Bound first: the address is said only once it is ours.
		ln, err := localapi.Listen(*listen)
		if err != nil {
			if errors.Is(err, syscall.EADDRINUSE) {
				die(sayf(t("%s 已经被占用（可能已经有一个 omnillm connect run）；停掉它，或用 --listen 换一个端口。", "%s is in use (another omnillm connect run?); stop it or pick another port with --listen."), *listen))
			}
			die(err)
		}
		log.Printf(t("本机接口：http://%s/v1 ，令牌在 %s（只有你能读）。", "Local API: http://%s/v1 ; token in %s (readable only by you)."), *listen, tokPath)
		localapi.HubOS = d.HubOS // the messages name the right kind of computer
		if err := localapi.ServeListener(ctx, ln, localapi.Handler(localapi.NewDeviceBackend(d, strings.TrimSpace(string(tok))))); err != nil {
			die(err)
		}
	case "forget":
		if paired, err := device.CheckPaired(ctx, sec); err == nil && !paired {
			fmt.Println(t("这台设备没有配对，没有要删的。", "This device isn’t paired; there is nothing to delete."))
			return
		}
		// Leave on the Mac first, so it disappears from its device list;
		// when the Mac can't be reached the pairing is still deleted here.
		left := false
		if d, err := device.Open(ctx, device.Options{Secrets: sec}); err == nil {
			lctx, cancel := context.WithTimeout(ctx, 20*time.Second)
			left = d.Leave(lctx) == nil
			cancel()
			d.Close()
		}
		if err := device.Forget(ctx, sec); err != nil {
			die(err)
		}
		if left {
			fmt.Println(t("这台设备已从你另一台电脑上移除，这里的配对也删了。", "This device was removed from your other computer and its pairing here is deleted."))
		} else {
			fmt.Println(t("这台设备的配对已删除。你另一台电脑现在连不上，之后在那边也移除它（控制台“设备”里，或在它的终端里运行 omnillm hub devices 查名字，再 omnillm hub revoke 名字）。", "This device’s pairing is deleted. Your other computer can’t be reached now; remove it there later too (under “Devices” in its console, or in its terminal: omnillm hub devices for the name, then omnillm hub revoke NAME)."))
		}
	case "models":
		d := openDevice(ctx, sec, dir)
		defer d.Close()
		ms, err := d.Models(ctx)
		deviceHubOS = d.HubOS()
		if err != nil {
			die(err)
		}
		if len(ms) == 0 {
			die(say(noModelsHere()))
		}
		for _, m := range ms {
			note := ""
			if m.HumanOnly || (m.ID == "auto" && !backgroundAuto(ms)) { // Auto then has only chat-only models
				note = t("（只能手动对话）", " (chat only)")
			}
			fmt.Printf("%-28s %s%s\n", stripControls(m.ID), modelName(m), note)
		}
	case "chat":
		fs := flag.NewFlagSet("connect chat", flag.ContinueOnError)
		model := fs.String("model", "", "")
		if err := parseFlags(fs, args[1:]); err != nil {
			dieArgs(err)
		}
		// Ctrl-C stops a reply, not the chat: only SIGTERM ends it.
		stop()
		deviceUpdateHint(dir)
		cctx, cstop := signal.NotifyContext(context.Background(), syscall.SIGTERM)
		defer cstop()
		d := openDevice(cctx, sec, dir)
		defer d.Close()
		chatLoop(cctx, d, *model)
	case "ask":
		fs := flag.NewFlagSet("connect ask", flag.ContinueOnError)
		model := fs.String("model", "", "")
		var imagePaths multi
		fs.Var(&imagePaths, "image", "attach an image file (repeatable)")
		if err := parseFlagsRest(fs, args[1:]); err != nil {
			dieArgs(err)
		}
		if !slices.Contains(args[1:], "--") {
			for _, w := range fs.Args() {
				if f := strings.TrimLeft(strings.SplitN(w, "=", 2)[0], "-"); strings.HasPrefix(w, "-") && (f == "model" || f == "image") {
					dieArgs(usageErr(t("--model 和 --image 要写在问题前面。", "Put --model and --image before the question.")))
				}
			}
		}
		if fs.NArg() == 0 && len(imagePaths) == 0 {
			dieArgs(usageErr(t("用法：omnillm connect ask [--model ID] [--image 图片文件] \"问题\"（不读标准输入）", "Usage: omnillm connect ask [--model ID] [--image FILE] \"question\" (standard input isn't read)")))
		}
		var images []omnillm.Image
		for _, p := range imagePaths {
			im, err := readImage(p)
			if err != nil {
				die(err)
			}
			images = append(images, im)
		}
		deviceUpdateHint(dir)
		d := openDevice(ctx, sec, dir)
		defer d.Close()
		if *model == "" {
			*model = defaultModel(ctx, d)
		} else if ms, err := d.Models(ctx); err == nil && !slices.ContainsFunc(ms, func(m omnillm.ModelInfo) bool { return m.ID == *model }) {
			die(say(runnable(t("没有这个模型。运行 omnillm connect models 查看。", "No such model. Run omnillm connect models to see them."))))
		}
		// A person at a terminal is waiting; from a script or cron (no
		// terminal) it is a background call, and subscription models,
		// kept for people, refuse it.
		s, err := d.Complete(ctx, omnillm.Request{Model: *model, Caller: callerHere(),
			Messages: []omnillm.Message{{Role: "user", Content: strings.Join(fs.Args(), " "), Images: images}}})
		if err != nil {
			if ctx.Err() != nil { // Ctrl-C
				fmt.Println()
				os.Exit(130)
			}
			die(err)
		}
		defer s.Close()
		said := ""
		for {
			c, err := s.Recv()
			if errors.Is(err, io.EOF) {
				fmt.Println()
				return
			}
			if err != nil {
				fmt.Println()
				if ctx.Err() != nil { // Ctrl-C
					os.Exit(130)
				}
				die(err)
			}
			autoSays(*model, c, &said)
			fmt.Print(termSafe(c.Text))
		}
	default:
		dieArgs(badCommand("connect", args[0]))
	}
}

// ---------------- relay ----------------

func relayCmd(args []string) {
	log.SetFlags(log.LstdFlags)
	if len(args) > 0 && args[0] == "token" {
		tok := hex.EncodeToString(e2e.RandomBytes(32))
		sum := sha256.Sum256([]byte(tok))
		fmt.Printf(t("主机登记令牌（放在主机的 OMNILLM_RELAY_TOKEN，保密）：\n%s\n远程连接用的哈希（--hub-token-hash，可以公开）：\n%x\n",
			"Hub registration token (OMNILLM_RELAY_TOKEN on the hub; keep secret):\n%s\nHash for the relay (--hub-token-hash; not secret):\n%x\n"), tok, sum)
		return
	}
	fs := flag.NewFlagSet("relay", flag.ExitOnError)
	listen := fs.String("listen", ":443", "")
	domain := fs.String("autocert", "", "get a TLS certificate for this domain from Let’s Encrypt")
	cache := fs.String("cert-cache", dataDir("relay-certs"), "")
	certFile := fs.String("tls-cert", "", "")
	keyFile := fs.String("tls-key", "", "")
	open := fs.Bool("open", false, "let any hub register (a public relay)")
	behindProxy := fs.Bool("behind-tls-proxy", false, "plain HTTP on loopback behind a TLS reverse proxy")
	var hashes, origins multi
	fs.Var(&hashes, "hub-token-hash", "sha256 of an allowed hub token (repeatable)")
	fs.Var(&origins, "web-origin", "origin of a web client allowed to open device mailboxes (repeatable; default the project’s)")
	_ = fs.Parse(args)
	opts := relay.Options{Open: *open, TrustProxy: *behindProxy, Logf: func(f string, a ...any) { log.Printf(f, a...) }}
	if len(origins) > 0 {
		opts.WebOrigins = origins
	}
	for _, h := range hashes {
		b, err := hex.DecodeString(h)
		if err != nil || len(b) != 32 {
			die(errors.New("--hub-token-hash must be 64 hex characters"))
		}
		var x [32]byte
		copy(x[:], b)
		opts.HubTokenHashes = append(opts.HubTokenHashes, x)
	}
	if !*open && len(opts.HubTokenHashes) == 0 {
		die(say(t("没有允许的主机：用 --hub-token-hash 或 --open。", "No hubs allowed: pass --hub-token-hash or --open.")))
	}
	rs := relay.New(opts)
	srv := &http.Server{Addr: *listen, Handler: rs, ReadHeaderTimeout: 10 * time.Second, MaxHeaderBytes: 16 << 10,
		TLSConfig: &tls.Config{MinVersion: tls.VersionTLS13}}
	ctx, stop := signalCtx()
	defer stop()
	// The day's counts close at midnight even if no hub connects then.
	go func() {
		t := time.NewTicker(time.Minute)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				rs.RollDay()
			}
		}
	}()
	go func() {
		<-ctx.Done()
		sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(sctx)
	}()
	var err error
	switch {
	case *domain != "":
		m := &autocert.Manager{Prompt: autocert.AcceptTOS, HostPolicy: autocert.HostWhitelist(*domain), Cache: autocert.DirCache(*cache)}
		srv.TLSConfig = m.TLSConfig()
		srv.TLSConfig.MinVersion = tls.VersionTLS13
		go func() {
			s80 := &http.Server{Addr: ":80", Handler: m.HTTPHandler(nil), ReadHeaderTimeout: 10 * time.Second,
				ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second, MaxHeaderBytes: 16 << 10}
			_ = s80.ListenAndServe()
		}()
		err = srv.ListenAndServeTLS("", "")
	case *certFile != "":
		err = srv.ListenAndServeTLS(*certFile, *keyFile)
	case *behindProxy:
		host, _, _ := net.SplitHostPort(*listen)
		if ip := net.ParseIP(host); ip == nil || !ip.IsLoopback() {
			die(say(t("--behind-tls-proxy 只能监听 127.0.0.1。", "--behind-tls-proxy may only listen on 127.0.0.1.")))
		}
		err = srv.ListenAndServe()
	default:
		die(say(t("远程连接必须用 TLS：用 --autocert、--tls-cert，或在 TLS 反向代理后面用 --behind-tls-proxy。",
			"The relay requires TLS: use --autocert, --tls-cert, or --behind-tls-proxy behind a TLS reverse proxy.")))
	}
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		die(err)
	}
}

type multi []string

func (m *multi) String() string     { return strings.Join(*m, ",") }
func (m *multi) Set(v string) error { *m = append(*m, v); return nil }

// defaultAPI is where the local API for apps listens unless told otherwise.
const defaultAPI = "127.0.0.1:8765"

// envOr is the environment variable k, or def when it is unset.
func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// appUsage is the forms of omnillm hub app.
func appUsage() string {
	return t("用法：omnillm hub app add 名字 | set 名字 [--name N] [--paused true|false] [--budget-usd X | --no-limit] | rm 名字（列出应用：omnillm hub apps）",
		"Usage: omnillm hub app add NAME | set NAME [--name N] [--paused true|false] [--budget-usd X | --no-limit] | rm NAME (to list them: omnillm hub apps)")
}

// deviceHubHint: hub commands on a device-only machine.
func deviceHubHint() string {
	if !device.Paired(context.Background(), secrets(dataDir("device"), "omnillm-device")) {
		return deviceNext()
	}
	return t("这台电脑只当设备用，主机命令要在它配对的那台电脑上运行。这里能用：omnillm connect chat、ask、models。",
		"This computer is set up as a device; hub commands run on the computer it is paired with. Here: omnillm connect chat, ask or models.")
}

// nothingAnswers: the error is a failed connection (nothing listening),
// not an answer the server gave.
func nothingAnswers(err error) bool {
	m := strings.ToLower(fmt.Sprint(err, " ", omnillm.AsError(err).Detail))
	return strings.Contains(m, "refused") || strings.Contains(m, "no such host") || strings.Contains(m, "timeout") || strings.Contains(m, "unreachable")
}

// shellWords quotes args for a command line the person copies: a word with
// anything but letters, digits and -_.,=/:@ goes in single quotes.
func shellWords(args []string) []string {
	out := make([]string, len(args))
	for i, a := range args {
		a = stripControls(a)
		if a != "" && strings.Trim(a, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_.,=/:@") == "" {
			out[i] = a
		} else {
			out[i] = "'" + strings.ReplaceAll(a, "'", `'\''`) + "'"
		}
	}
	return out
}

// relayOnText is how remote access comes back on: an install keeps the
// setting it was given, so it is given the relay again.
func relayOnText(l string) string {
	return tl(l, "要打开：运行 OMNILLM_RELAY="+hub.DefaultRelayURL+" omnillm install（在前台运行的，不带 OMNILLM_RELAY=off 重新启动）。",
		"To turn it on, run OMNILLM_RELAY="+hub.DefaultRelayURL+" omnillm install (a hub run in the foreground: start it again without OMNILLM_RELAY=off).")
}

// apiSetting is the local API's address as the hub was started with it
// ("off": turned off), for what the control commands say about it.
var apiSetting string

// sentenceGap goes between two sentences: none in Chinese.
func sentenceGap() string { return t("", " ") }
