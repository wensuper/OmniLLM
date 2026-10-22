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
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
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
	return control.Call(sock, req, onReply)
}

func die(err error) {
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
			helpCmd([]string{"--all"})
			os.Exit(0)
		}
		fmt.Fprintln(os.Stderr, string(u))
		fmt.Fprintln(os.Stderr)
		subUsage()
	}
	// A value that isn't right: the sentence says which, no help needed.
	fmt.Fprintln(os.Stderr, describe(err))
	os.Exit(2)
}

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
		return say(omnillm.Describe(l, err) + tl(l, "（原因写在 OmniLLM 的日志里）", " (the cause is in OmniLLM’s log)"))
	}
	return say(omnillm.Describe(l, err))
}

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
  omnillm status                     查看主机状态
  omnillm update [--check] [--auto on|off]
                                     更新到最新版本（主机默认会在空闲时自己更新）；--auto 开关自动更新
  omnillm login claude|chatgpt       用 Claude 或 ChatGPT 订阅：没装官方程序会先装上，再在浏览器里登录
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
  omnillm hub audit
  omnillm hub pair [--name N] [--models 名字或ID,…] [--budget-usd 5] [--background-only]
  omnillm hub revoke <名字或ID>      立即移除设备（名字里有空格或括号时加引号：omnillm hub revoke "iPhone (2)"）
  omnillm hub device set <名字或ID> [--name N] [--paused true|false] [--budget-usd X | --no-limit] [--models 名字或ID,… | --all-models]
                                     改设备的名字、开启或暂停、每天额度、能用的模型
  omnillm hub app add <名字>         给不能嵌入库的程序发一枚应用令牌
  omnillm hub app set <名字或ID> [--name N] [--paused true|false] [--budget-usd X | --no-limit] [--models 名字或ID,… | --all-models]
                                     改应用的名字、开启或暂停、每天额度、能用的模型
  omnillm hub app rm <名字或ID>      移除应用，它的令牌立即失效
  omnillm hub model set <名字或ID> [--name 新名字] [--shown true|false] [--human-only true|false] [--price-in X --price-out Y]
                                     改名（--name "" 恢复原名）、是否在设备上显示、只能手动对话、价格
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
  omnillm status                     show the hub’s status
  omnillm update [--check] [--auto on|off]
                                     update to the newest version (the hub also updates itself when idle); --auto turns that on or off
  omnillm login claude|chatgpt       use your Claude or ChatGPT subscription: installs the official program if needed, signs in in your browser
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
  omnillm hub audit
  omnillm hub pair [--name N] [--models name-or-id,…] [--budget-usd 5] [--background-only]
  omnillm hub revoke <name-or-id>    remove a device now (quote a name with spaces or brackets: omnillm hub revoke "iPhone (2)")
  omnillm hub device set <name-or-id> [--name N] [--paused true|false] [--budget-usd X | --no-limit] [--models name-or-id,… | --all-models]
                                     rename a device, pause or resume it, change its daily limit or models
  omnillm hub app add <name>         token for a program that can’t embed the library
  omnillm hub app set <name-or-id> [--name N] [--paused true|false] [--budget-usd X | --no-limit] [--models name-or-id,… | --all-models]
                                     rename an app, pause or resume it, change its daily limit or models
  omnillm hub app rm <name-or-id>    remove an app; its token stops working at once
  omnillm hub model set <name-or-id> [--name new-name] [--shown true|false] [--human-only true|false] [--price-in X --price-out Y]
                                     rename (--name "" restores the original), show on devices, chat only, prices
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
	return t(`用法：
  omnillm install                    一步装好：主机作为后台服务运行，打开控制台
  omnillm open                       打开本机控制台
  omnillm open --add                 打开控制台，直接到「添加设备」
  omnillm status                     查看主机状态
  omnillm update [--check] [--auto on|off]
                                     更新到最新版本（主机默认会在空闲时自己更新）；--auto 开关自动更新
  omnillm login claude|chatgpt       用 Claude 或 ChatGPT 订阅：没装官方程序会先装上，再在浏览器里登录
  omnillm uninstall [--purge]        停止并移除后台服务（--purge 连数据和 API Key 一起删）
  omnillm tray                       把菜单栏图标找回来
  omnillm version                    显示版本
  omnillm help --all                 所有命令（主机、设备、远程连接管理）
`, `Usage:
  omnillm install                    one step: run the hub as a background service, open the console
  omnillm open                       open the local console
  omnillm open --add                 open the console on Add device
  omnillm status                     show the hub’s status
  omnillm update [--check] [--auto on|off]
                                     update to the newest version (the hub also updates itself when idle); --auto turns that on or off
  omnillm login claude|chatgpt       use your Claude or ChatGPT subscription: installs the official program if needed, signs in in your browser
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
func helpCmd(args []string) {
	all := len(args) > 0 && (args[0] == "--all" || args[0] == "-a" || args[0] == "all")
	fmt.Print(usageText(all))
}

func main() {
	log.SetFlags(0)
	if len(os.Args) < 2 {
		helpCmd(nil)
		return
	}
	switch os.Args[1] {
	case "help", "-h", "--help":
		helpCmd(os.Args[2:])
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
	case "tray":
		trayCmd()
	case "version":
		fmt.Println("omnillm " + version)
	default:
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
	if err != nil || !strings.HasPrefix(url, "http://127.0.0.1:") {
		fmt.Fprintln(os.Stderr, notRunning())
		os.Exit(1)
	}
	if *add {
		url += "&add"
	}
	if err := openLogin(url); err != nil {
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
		if st.Mode().Perm()&0o077 != 0 {
			die(say(t("主密钥文件只能自己可读（0600）：", "The master key file must be readable only by you (0600): ") + f))
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
	dir := dataDir("hub")
	sock := filepath.Join(dir, "control.sock")
	switch args[0] {
	case "run":
		hubRun(dir, sock, args[1:])
	case "provider":
		if len(args) >= 3 && args[1] == "rm" {
			call(sock, control.Request{Op: "provider.rm", Args: map[string]string{"name": strings.Join(args[2:], " ")}})
			return
		}
		if len(args) >= 3 && (args[1] == "off" || args[1] == "on") {
			call(sock, control.Request{Op: "provider.off", Args: map[string]string{"name": strings.Join(args[2:], " "), "off": fmt.Sprint(args[1] == "off")}})
			return
		}
		if len(args) < 3 || args[1] != "add" || strings.HasPrefix(args[2], "-") {
			dieArgs(usageErr(t("用法：omnillm hub provider add <openai|anthropic|ollama|openai-compatible> [--name N] [--base-url U]",
				"Usage: omnillm hub provider add <openai|anthropic|ollama|openai-compatible> [--name N] [--base-url U]")))
		}
		fs := flag.NewFlagSet("provider add", flag.ContinueOnError)
		name := fs.String("name", "", "")
		base := fs.String("base-url", "", "")
		if err := parseFlags(fs, args[3:]); err != nil {
			dieArgs(err)
		}
		kind := args[2]
		key := ""
		if kind != "ollama" {
			key = readSecret(t("粘贴 API Key（不会显示）：", "Paste the API key (not shown): "))
		}
		call(sock, control.Request{Op: "provider.add", Args: map[string]string{"kind": kind, "name": *name, "base_url": *base}, Secret: key})
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
		call(sock, control.Request{Op: args[0]})
	case "pair":
		a, err := pairArgs(args[1:])
		if err != nil {
			dieArgs(err)
		}
		call(sock, control.Request{Op: "pair", Args: a})
	case "revoke":
		// The name may be typed unquoted: "hub revoke iPhone 2".
		if len(args) < 2 {
			dieArgs(usageErr(t("请写出要移除的设备的名字或 ID。", "Name the device to remove (its name or ID).")))
		}
		call(sock, control.Request{Op: "revoke", Args: map[string]string{"id": strings.Join(args[1:], " ")}})
	case "device":
		if len(args) < 2 || args[1] != "set" {
			dieArgs(badCommand(append([]string{"hub"}, args[:min(len(args), 2)]...)...))
		}
		a, err := deviceSetArgs(splitTarget(args[2:]))
		if err != nil {
			dieArgs(err)
		}
		call(sock, control.Request{Op: "device.set", Args: a})
	case "app":
		if len(args) < 2 {
			dieArgs(badCommand("hub", "app"))
		}
		if (args[1] == "add" || args[1] == "rm") && len(args) < 3 {
			dieArgs(usageErr(t("请写出应用的名字。", "Name the app.")))
		}
		switch args[1] {
		case "add":
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
			dieArgs(badCommand("hub", "app", args[1]))
		}
	case "model":
		if len(args) < 2 || args[1] != "set" {
			dieArgs(badCommand(append([]string{"hub"}, args[:min(len(args), 2)]...)...))
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
		if len(args) != 2 || args[1] != "confirm" {
			dieArgs(badCommand(append([]string{"hub"}, args...)...))
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
	if err := parseFlags(fs, args); err != nil {
		return nil, err
	}
	a := map[string]string{"name": *name, "models": *models, "budget_usd": "0", "background_only": fmt.Sprint(*bg)}
	if *budget != "" {
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
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
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
	err := ctl(sock, req, func(r control.Reply) (bool, bool) {
		switch r.Event {
		case "info", "ok", "done":
			if r.Text != "" {
				fmt.Println(r.Text)
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
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func hubRun(dir, sock string, args []string) {
	log.SetFlags(log.LstdFlags) // a service log needs times
	fs := flag.NewFlagSet("hub run", flag.ExitOnError)
	relayURL := fs.String("relay", os.Getenv("OMNILLM_RELAY"), "relay URL (wss://…; default the project’s; off: none)")
	api := fs.String("api", envOr("OMNILLM_API", defaultAPI), "serve the local API for app tokens on this loopback address (off: none)")
	consoleAddr := fs.String("console", "127.0.0.1:8770", "serve the local console on this loopback address (empty: off)")
	webApp := fs.String("web-app", "", "web client the pairing QR code opens (default: the project’s)")
	_ = fs.Parse(args)
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
	if *api != "" && *api != "off" {
		if ln, err := localapi.Listen(*api); err != nil {
			log.Printf(t("给应用用的本机接口没能启动（%v）；应用暂时连不上。", "The local API for apps did not start (%v); apps can’t connect for now."), err)
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
	log.Print(t("OmniLLM 已启动。", "OmniLLM is running."))
	if *relayURL == "" {
		log.Print(t("没有设置远程连接（--relay），其他设备暂时连不上。", "Remote access not set up (--relay); other devices can’t connect yet."))
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
		case "subscription.connect":
			// After omnillm login: connect it now, even if it was
			// removed before (logging in again is asking for it).
			if err := h.ConnectSubscription(ctx, a["kind"]); err != nil {
				return hubSay(l, err)
			}
			name := tl(l, "Claude 订阅", "Claude subscription")
			if a["kind"] == string(provider.Codex) {
				name = tl(l, "ChatGPT 订阅", "ChatGPT subscription")
			}
			return c.Send(control.Reply{Event: "ok", Text: fmt.Sprintf(tl(l, "已接入 %s：手机和其他设备现在就能用。", "Connected your %s: your phone and other devices can use it now."), name)})
		case "console.login":
			if con == nil {
				return say(tl(l, "控制台没有开启。", "The console is off."))
			}
			return c.Send(control.Reply{Event: "ok", Text: con.LoginURL()})
		case "status":
			if ps, _ := h.Providers(ctx); len(ps) == 0 {
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
				if p.Problem != "" {
					problems = append(problems, SourceProblem{Name: p.Name, Kind: p.Kind, DisplayName: p.DisplayName, Problem: p.Problem, ResetsAt: p.ResetsAt})
				}
			}
			_, _, paused := h.RegionStatus()
			st := map[string]any{"running": true, "version": version, "providers": len(ps), "devices": len(ds), "online": online,
				"relay_configured": relayURL != "", "relay_up": h.RelayUp(), "console": con != nil, "os": runtime.GOOS,
				"region_paused": paused, "problems": problems, "lang": h.UILang(ctx)}
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
			}
			ms, err := h.AddProvider(ctx, cfg, req.Secret)
			if err != nil {
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
			return c.Send(control.Reply{Event: "ok", Text: fmt.Sprintf(tl(l, "已更新“%s”。", "Updated “%s”."), d.Name)})
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
			return c.Send(control.Reply{Event: "ok", Text: fmt.Sprintf(tl(l,
				"已为“%s”创建令牌（只显示这一次，请保存好）：\n%s\n这个应用只能做后台调用，每天额度 %s。",
				"Token for “%s” (shown once, keep it safe):\n%s\nThis app makes background calls only; daily limit %s."), app.Name, tok, usd(app.Budget))})
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
			return c.Send(control.Reply{Event: "ok", Text: fmt.Sprintf(tl(l, "已移除“%s”，它的模型不再能用。", "Removed “%s”; its models can’t be used any more."), sourceNameIn(l, p))})
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
			if err := h.ConfirmRegion(ctx); err != nil {
				return hubSay(l, err)
			}
			return c.Send(control.Reply{Event: "ok", Text: tl(l, "已确认，模型恢复使用。", "Confirmed; models are back on.")})
		case "pair":
			return pairOp(ctx, h, c, l, a)
		}
		return say(tl(l, "OmniLLM 不认识这个命令。运行 omnillm install 更新到同一个版本。", "OmniLLM doesn’t know this command. Run omnillm install to update it to the same version."))
	}
}

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
	if v := strings.TrimSpace(a["budget_usd"]); v != "" && v != "0" {
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
	expires := p.Expires.Format(tl(l, "15:04", "3:04 PM"))
	if err := c.Send(control.Reply{Event: "info", Text: fmt.Sprintf(tl(l,
		"手机或平板：用相机扫上面的二维码。电脑或服务器：运行 omnillm connect pair，粘贴下面这段配对文字（%s 前有效，只能用一次；不要发给别人，也不要贴到聊天里）：\n\n%s\n",
		"Phone or tablet: scan the QR code above with the camera. Computer or server: run omnillm connect pair and paste this pairing text (valid until %s, single use; don’t send it to anyone or paste it into a chat):\n\n%s\n"),
		expires, p.Code)}); err != nil {
		return err
	}
	d, err := p.Wait(ctx)
	if err != nil {
		return hubSay(l, err)
	}
	return c.Send(control.Reply{Event: "done", Text: fmt.Sprintf(tl(l, "已添加设备“%s”（%s），每天额度 %s。", "Added “%s” (%s), daily limit %s."), d.Name, d.ID, limitTextIn(l, d.Budget))})
}

// ---------------- connect (device) ----------------

func connectCmd(args []string) {
	if len(args) == 0 {
		subUsage()
	}
	dir := dataDir("device")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		die(err)
	}
	sec := secrets(dir, "omnillm-device")
	ctx, stop := signalCtx()
	defer stop()
	switch args[0] {
	case "pair":
		paired, err := device.CheckPaired(ctx, sec)
		if err != nil {
			die(sayf(t("读不出这台设备保存的配对（%s）。如果它已经配对过，请用配对时的同一个用户运行；确实要重新配对，先运行 omnillm connect forget。", "Can’t read this device’s saved pairing (%s). If it is already paired, run as the same user it was paired as; to pair again, run omnillm connect forget first."), dir))
		}
		if paired {
			die(say(t("这台设备已经配对过（"+dir+"）。要换一台 Mac 或重新配对，先运行 omnillm connect forget。", "This device is already paired ("+dir+"). To pair again (or with another Mac), run omnillm connect forget first.")))
		}
		code := readSecret(t("粘贴你的 Mac 上“添加设备”给的配对文字：", "Paste the pairing text from “Add device” on your Mac: "))
		host, _ := os.Hostname()
		info, err := device.Pair(ctx, device.Options{Secrets: sec}, code, host, func(sas string) {
			if len(sas) == 6 {
				sas = sas[:3] + " " + sas[3:]
			}
			fmt.Printf(t("\n请核对：你的 Mac 上显示的数字应该是 %s，一致就在 Mac 上点“一致，添加”。\n", "\nCheck: your Mac should show %s. If it does, choose “Matches, add” there.\n"), sas)
		})
		if err != nil {
			die(err)
		}
		fmt.Printf(t("已添加到你的 Mac（这台设备在 Mac 上叫“%s”）。\n", "Added to your Mac (it lists this device as “%s”).\n"), info.Name)
	case "run":
		fs := flag.NewFlagSet("connect run", flag.ContinueOnError)
		listen := fs.String("listen", "127.0.0.1:8766", "")
		if err := parseFlags(fs, args[1:]); err != nil {
			dieArgs(err)
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
		log.Printf(t("本机接口：http://%s/v1 ，令牌在 %s（只有你能读）。", "Local API: http://%s/v1 ; token in %s (readable only by you)."), *listen, tokPath)
		if err := localapi.Serve(ctx, *listen, localapi.Handler(localapi.NewDeviceBackend(d, strings.TrimSpace(string(tok))))); err != nil {
			die(err)
		}
	case "forget":
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
			fmt.Println(t("这台设备已从你的 Mac 上移除，这里的配对也删了。", "This device was removed from your Mac and its pairing here is deleted."))
		} else {
			fmt.Println(t("这台设备的配对已删除。你的 Mac 现在连不上，记得在 Mac 的设备列表里也移除它。", "This device’s pairing is deleted. Your Mac can’t be reached now; remove it from the Mac’s device list too."))
		}
	case "models":
		d := openDevice(ctx, sec, dir)
		defer d.Close()
		ms, err := d.Models(ctx)
		if err != nil {
			die(err)
		}
		for _, m := range ms {
			note := ""
			if m.HumanOnly {
				note = t("（只能手动对话）", " (chat only)")
			}
			fmt.Printf("%-28s %s%s\n", m.ID, modelName(m), note)
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
		if fs.NArg() == 0 && len(imagePaths) == 0 {
			dieArgs(usageErr(t("请写出要问的问题。", "Write the question to ask.")))
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
		}
		// A person at a terminal is waiting; from a script or cron (no
		// terminal) it is a background call, and subscription models,
		// kept for people, refuse it.
		s, err := d.Complete(ctx, omnillm.Request{Model: *model, Caller: callerHere(),
			Messages: []omnillm.Message{{Role: "user", Content: strings.Join(fs.Args(), " "), Images: images}}})
		if err != nil {
			die(err)
		}
		defer s.Close()
		for {
			c, err := s.Recv()
			if errors.Is(err, io.EOF) {
				fmt.Println()
				return
			}
			if err != nil {
				fmt.Println()
				die(err)
			}
			fmt.Print(c.Text)
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
	srv := &http.Server{Addr: *listen, Handler: relay.New(opts), ReadHeaderTimeout: 10 * time.Second, MaxHeaderBytes: 16 << 10,
		TLSConfig: &tls.Config{MinVersion: tls.VersionTLS13}}
	ctx, stop := signalCtx()
	defer stop()
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
