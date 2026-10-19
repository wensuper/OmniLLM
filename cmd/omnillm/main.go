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
	"runtime"
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

// t picks the Chinese or English text.
func t(z, e string) string {
	if zh() {
		return z
	}
	return e
}

func die(err error) {
	fmt.Fprintln(os.Stderr, omnillm.Describe(lang(), err))
	if e := omnillm.AsError(err); e.Code == omnillm.CodeInternal {
		fmt.Fprintln(os.Stderr, "  ("+err.Error()+")")
	}
	os.Exit(1)
}

func usage() {
	fmt.Fprint(os.Stderr, t(`用法：
  omnillm install                    一步装好：主机作为后台服务运行，打开控制台
  omnillm open                       打开本机控制台
  omnillm status                     查看主机状态
  omnillm hub run                    启动主机（前台运行）
  omnillm hub provider add <openai|anthropic|ollama|openai-compatible> [--name N] [--base-url U]
                                     接入模型提供方（Key 从标准输入读，不写在命令行里）
  omnillm hub providers | models | devices | apps | usage | audit
  omnillm hub pair [--name N] [--models a,b] [--budget-usd 5] [--background-only]
  omnillm hub revoke <设备ID>        立即移除设备
  omnillm hub app add <名字>         给不能嵌入库的程序发一枚应用令牌
  omnillm hub app rm <ID>
  omnillm hub model set <ID> [--price-in X --price-out Y] [--name 显示名] [--human-only true|false]
  omnillm hub region confirm         确认主机所在地区变化并恢复调用
  omnillm connect pair               在这台设备上输入主机给的配对文字
  omnillm connect run [--listen 127.0.0.1:8766]
  omnillm connect chat [--model ID]  在终端里和主机上的模型对话
  omnillm connect ask [--model ID] [--image 图片文件] "问题"
  omnillm connect models             列出能用的模型
  omnillm connect forget             删除这台设备的配对
  omnillm relay --listen :443 --autocert relay.example.com --hub-token-hash HEX
  omnillm relay token                生成一枚主机登记令牌和它的哈希
`, `usage:
  omnillm install                    one step: run the hub as a background service, open the console
  omnillm open                       open the local console
  omnillm status                     show the hub's status
  omnillm hub run                    run the hub (foreground)
  omnillm hub provider add <openai|anthropic|ollama|openai-compatible> [--name N] [--base-url U]
                                     connect a provider (the key is read from stdin, never argv)
  omnillm hub providers | models | devices | apps | usage | audit
  omnillm hub pair [--name N] [--models a,b] [--budget-usd 5] [--background-only]
  omnillm hub revoke <device-id>     remove a device now
  omnillm hub app add <name>         token for a program that cannot embed the library
  omnillm hub app rm <id>
  omnillm hub model set <id> [--price-in X --price-out Y] [--name display] [--human-only true|false]
  omnillm hub region confirm         accept the hub's new region and resume calls
  omnillm connect pair               enter the hub's pairing text on this device
  omnillm connect run [--listen 127.0.0.1:8766]
  omnillm connect chat [--model ID]  chat with the hub's models in this terminal
  omnillm connect ask [--model ID] [--image FILE] "question"
  omnillm connect models             list the models you can use
  omnillm connect forget             delete this device's pairing
  omnillm relay --listen :443 --autocert relay.example.com --hub-token-hash HEX
  omnillm relay token                make a hub registration token and its hash
`))
	os.Exit(2)
}

func main() {
	log.SetFlags(0)
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "hub":
		hubCmd(os.Args[2:])
	case "connect":
		connectCmd(os.Args[2:])
	case "relay":
		relayCmd(os.Args[2:])
	case "open":
		openCmd()
	case "install":
		installCmd(os.Args[2:])
	case "uninstall":
		uninstallCmd(os.Args[2:])
	case "status":
		statusCmd(os.Args[2:])
	case "version":
		fmt.Println("omnillm " + version)
	default:
		usage()
	}
}

// version is set at build time (-ldflags "-X main.version=…").
var version = "0.1.0-dev"

func openCmd() {
	sock := filepath.Join(dataDir("hub"), "control.sock")
	var url string
	err := control.Call(sock, control.Request{Op: "console.login"}, func(r control.Reply) (bool, bool) {
		url = r.Text
		return false, false
	})
	if err != nil || !strings.HasPrefix(url, "http://127.0.0.1:") {
		fmt.Fprintln(os.Stderr, t("主机没有在运行。先运行 omnillm install（或 omnillm hub run）。", "The hub is not running. Run omnillm install (or omnillm hub run) first."))
		os.Exit(1)
	}
	if err := openLogin(url); err != nil {
		fmt.Println(t("在这台电脑的浏览器里打开（2 分钟内有效，只能用一次）：", "Open this in a browser on this computer (valid 2 minutes, single use):"))
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
			die(errors.New(t("主密钥文件只能自己可读（0600）：", "The master key file must be readable only by you (0600): ") + f))
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
		usage()
	}
	dir := dataDir("hub")
	sock := filepath.Join(dir, "control.sock")
	switch args[0] {
	case "run":
		hubRun(dir, sock, args[1:])
	case "provider":
		if len(args) < 3 || args[1] != "add" {
			usage()
		}
		fs := flag.NewFlagSet("provider add", flag.ExitOnError)
		name := fs.String("name", "", "")
		base := fs.String("base-url", "", "")
		_ = fs.Parse(args[3:])
		kind := args[2]
		key := ""
		if kind != "ollama" {
			key = readSecret(t("粘贴 API Key（不会显示）：", "Paste the API key (not shown): "))
		}
		call(sock, control.Request{Op: "provider.add", Args: map[string]string{"kind": kind, "name": *name, "base_url": *base}, Secret: key})
	case "providers", "models", "devices", "apps", "usage", "audit":
		call(sock, control.Request{Op: args[0]})
	case "pair":
		fs := flag.NewFlagSet("pair", flag.ExitOnError)
		name := fs.String("name", "", "")
		models := fs.String("models", "", "")
		budget := fs.Float64("budget-usd", 0, "")
		bg := fs.Bool("background-only", false, "for servers and scripts: calls never count as a person waiting")
		_ = fs.Parse(args[1:])
		call(sock, control.Request{Op: "pair", Args: map[string]string{"name": *name, "models": *models, "budget_usd": fmt.Sprint(*budget), "background_only": fmt.Sprint(*bg)}})
	case "revoke":
		if len(args) != 2 {
			usage()
		}
		call(sock, control.Request{Op: "revoke", Args: map[string]string{"id": args[1]}})
	case "app":
		if len(args) != 3 {
			usage()
		}
		switch args[1] {
		case "add":
			call(sock, control.Request{Op: "app.add", Args: map[string]string{"name": args[2]}})
		case "rm":
			call(sock, control.Request{Op: "app.rm", Args: map[string]string{"id": args[2]}})
		default:
			usage()
		}
	case "model":
		if len(args) < 3 || args[1] != "set" {
			usage()
		}
		fs := flag.NewFlagSet("model set", flag.ExitOnError)
		pin := fs.Float64("price-in", -1, "")
		pout := fs.Float64("price-out", -1, "")
		disp := fs.String("name", "", "")
		human := fs.String("human-only", "", "true or false; omit to leave unchanged")
		_ = fs.Parse(args[3:])
		call(sock, control.Request{Op: "model.set", Args: map[string]string{"id": args[2], "price_in": fmt.Sprint(*pin),
			"price_out": fmt.Sprint(*pout), "name": *disp, "human_only": *human}})
	case "region":
		if len(args) != 2 || args[1] != "confirm" {
			usage()
		}
		call(sock, control.Request{Op: "region.confirm"})
	default:
		usage()
	}
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

func call(sock string, req control.Request) {
	err := control.Call(sock, req, func(r control.Reply) (bool, bool) {
		switch r.Event {
		case "info", "ok", "done":
			if r.Text != "" {
				fmt.Println(r.Text)
			}
			if len(r.Data) > 0 {
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
	relayURL := fs.String("relay", os.Getenv("OMNILLM_RELAY"), "relay URL (wss://…; default the project's; off: none)")
	api := fs.String("api", "", "serve the local API for app tokens on this loopback address, e.g. 127.0.0.1:8765")
	consoleAddr := fs.String("console", "127.0.0.1:8770", "serve the local console on this loopback address (empty: off)")
	webApp := fs.String("web-app", "", "web client the pairing QR code opens (default: the project's)")
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
	h, err := hub.New(ctx, hub.Options{Secrets: sec, Store: st, RelayURL: *relayURL, RelayToken: token, WebAppURL: *webApp,
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
			log.Printf(t("控制台没能启动：%v", "Console did not start: %v"), err)
		} else {
			con, err = console.New(h, cln.Addr().String())
			if err != nil {
				die(err)
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
	if *api != "" {
		go func() {
			if err := localapi.Serve(ctx, *api, localapi.Handler(localapi.HubBackend{Hub: h})); err != nil {
				log.Printf("local API: %v", err)
			}
		}()
	}
	log.Print(t("主机已启动。", "Hub running."))
	if *relayURL == "" {
		log.Print(t("没有配置连接服务（--relay），其他设备暂时连不上。", "No relay configured (--relay); other devices cannot connect yet."))
	} else {
		sum := sha256.Sum256([]byte(token))
		log.Printf(t("连接服务登记令牌的哈希（自己部署连接服务时填进 --hub-token-hash）：%x", "Relay registration token hash (for your own relay's --hub-token-hash): %x"), sum)
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
					return errors.New(omnillm.Describe(lang(), err))
				}
			}
			return c.Send(control.Reply{Event: "ok"})
		case "console.login":
			if con == nil {
				return errors.New(t("控制台没有开启。", "The console is off."))
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
			st := map[string]any{"running": true, "version": version, "providers": len(ps), "devices": len(ds),
				"relay_configured": relayURL != "", "relay_up": h.RelayUp(), "console": con != nil}
			return c.Send(control.Reply{Event: "ok", Data: jsonData(st)})
		case "provider.add":
			cfg, err := provider.Default(provider.Kind(a["kind"]))
			if a["kind"] == string(provider.OpenAICompatible) {
				cfg, err = provider.Config{Kind: provider.OpenAICompatible, NeedsKey: true}, nil
			}
			if err != nil {
				return errors.New(t("不认识这个提供方类型。", "Unknown provider kind."))
			}
			if a["name"] != "" {
				cfg.Name = a["name"]
			}
			if a["base_url"] != "" {
				cfg.BaseURL = a["base_url"]
			}
			ms, err := h.AddProvider(ctx, cfg, req.Secret)
			if err != nil {
				return errors.New(omnillm.Describe(lang(), err))
			}
			return c.Send(control.Reply{Event: "ok", Text: fmt.Sprintf(t("已接入 %s，%d 个模型可用。", "Connected %s: %d models available."), cfg.Name, len(ms))})
		case "providers":
			ps, err := h.Providers(ctx)
			if err != nil {
				return err
			}
			return c.Send(control.Reply{Event: "ok", Data: jsonData(ps)})
		case "models":
			ms, err := h.Models(ctx)
			if err != nil {
				return err
			}
			return c.Send(control.Reply{Event: "ok", Data: jsonData(ms)})
		case "devices":
			ds, err := h.Devices(ctx)
			if err != nil {
				return err
			}
			type row struct {
				ID, Name string
				Budget   string
				Created  time.Time
				LastSeen time.Time `json:",omitempty"`
				Models   []string  `json:",omitempty"`
			}
			var rows []row
			for _, d := range ds {
				rows = append(rows, row{d.ID, d.Name, usd(d.Budget), d.Created, d.LastSeen, d.Models})
			}
			return c.Send(control.Reply{Event: "ok", Data: jsonData(rows)})
		case "apps":
			as, err := h.Apps(ctx)
			if err != nil {
				return err
			}
			type row struct{ ID, Name, Budget string }
			var rows []row
			for _, x := range as {
				rows = append(rows, row{x.ID, x.Name, usd(x.Budget)})
			}
			return c.Send(control.Reply{Event: "ok", Data: jsonData(rows)})
		case "usage":
			u, err := h.Usage(ctx, "")
			if err != nil {
				return err
			}
			return c.Send(control.Reply{Event: "ok", Data: jsonData(u)})
		case "audit":
			u, err := h.Audit(ctx, "")
			if err != nil {
				return err
			}
			return c.Send(control.Reply{Event: "ok", Data: jsonData(u)})
		case "revoke":
			if err := h.Revoke(ctx, a["id"]); err != nil {
				return errors.New(omnillm.Describe(lang(), err))
			}
			return c.Send(control.Reply{Event: "ok", Text: t("已移除，立即生效。", "Removed. Effective now.")})
		case "app.add":
			app, tok, err := h.AddApp(ctx, a["name"], nil, 0)
			if err != nil {
				return err
			}
			return c.Send(control.Reply{Event: "ok", Text: fmt.Sprintf(t(
				"已为“%s”创建令牌（只显示这一次，请保存好）：\n%s\n这个应用只能做后台调用，每天额度 %s。",
				"Token for \"%s\" (shown once, keep it safe):\n%s\nThis app makes background calls only; daily budget %s."), app.Name, tok, usd(app.Budget))})
		case "app.rm":
			if err := h.RevokeApp(ctx, a["id"]); err != nil {
				return err
			}
			return c.Send(control.Reply{Event: "ok", Text: t("已移除。", "Removed.")})
		case "model.set":
			var u hub.ModelUpdate
			if n := a["name"]; n != "" {
				u.DisplayName = &n
			}
			if a["price_in"] != "-1" || a["price_out"] != "-1" {
				pi, err1 := strconv.ParseFloat(a["price_in"], 64)
				po, err2 := strconv.ParseFloat(a["price_out"], 64)
				if err1 != nil || err2 != nil || pi < 0 || po < 0 {
					return errors.New(t("请同时给出 --price-in 和 --price-out（美元/百万 token）。", "Give both --price-in and --price-out (USD per million tokens)."))
				}
				u.PriceIn, u.PriceOut = &pi, &po
			}
			switch a["human_only"] {
			case "true":
				v := true
				u.HumanOnly = &v
			case "false":
				v := false
				u.HumanOnly = &v
			}
			if err := h.SetModel(ctx, a["id"], u); err != nil {
				return errors.New(omnillm.Describe(lang(), err))
			}
			return c.Send(control.Reply{Event: "ok", Text: t("已更新。", "Updated.")})
		case "region.confirm":
			if err := h.ConfirmRegion(ctx); err != nil {
				return err
			}
			return c.Send(control.Reply{Event: "ok", Text: t("已确认，恢复调用。", "Confirmed; calls resume.")})
		case "pair":
			return pairOp(ctx, h, c, a)
		}
		return errors.New("unknown command")
	}
}

func usd(micro int64) string { return fmt.Sprintf("$%.2f", float64(micro)/1e6) }

func pairOp(ctx context.Context, h *hub.Hub, c *control.Conn, a map[string]string) error {
	var models []string
	if a["models"] != "" {
		models = strings.Split(a["models"], ",")
	}
	budget := int64(0)
	if f, err := strconv.ParseFloat(a["budget_usd"], 64); err == nil && f > 0 {
		budget = int64(f * 1e6)
	}
	p, err := h.Pair(ctx, hub.PairOptions{Name: a["name"], Models: models, Budget: budget, BackgroundOnly: a["background_only"] == "true",
		Confirm: func(ctx context.Context, r hub.PairRequest) (bool, error) {
			return c.Ask(ctx, fmt.Sprintf(t("设备“%s”请求加入。它屏幕上显示的数字是 %s 吗？一致才确认。",
				"Device \"%s\" wants to join. Does its screen show %s? Confirm only if it matches."), r.DeviceName, r.Code))
		}})
	if err != nil {
		return errors.New(omnillm.Describe(lang(), err))
	}
	// If this command ends early (the owner pressed Ctrl-C, the terminal
	// closed), the code dies with it instead of holding a slot for 5 min.
	defer p.Cancel()
	if err := c.Send(control.Reply{Event: "qr", Text: p.Link}); err != nil {
		return err
	}
	if err := c.Send(control.Reply{Event: "info", Text: fmt.Sprintf(t(
		"手机或平板：用相机扫上面的二维码。电脑或服务器：运行 omnillm connect pair，粘贴下面这段配对文字（%s 前有效，只能用一次；不要发给别人，也不要贴到聊天里）：\n\n%s\n",
		"Phone or tablet: scan the QR code above with the camera. Computer or server: run omnillm connect pair and paste this pairing text (valid until %s, single use; do not send it to anyone or paste it into a chat):\n\n%s\n"),
		p.Expires.Format("15:04"), p.Code)}); err != nil {
		return err
	}
	d, err := p.Wait(ctx)
	if err != nil {
		return errors.New(omnillm.Describe(lang(), err))
	}
	return c.Send(control.Reply{Event: "done", Text: fmt.Sprintf(t("已添加设备“%s”（%s），每天额度 %s。", "Added \"%s\" (%s), daily budget %s."), d.Name, d.ID, usd(d.Budget))})
}

// ---------------- connect (device) ----------------

func connectCmd(args []string) {
	if len(args) == 0 {
		usage()
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
		if device.Paired(ctx, sec) {
			die(errors.New(t("这台设备已经配对过。要换一台主机或重新配对，先运行 omnillm connect forget。", "This device is already paired. To pair again (or with another hub), run omnillm connect forget first.")))
		}
		code := readSecret(t("粘贴主机给的配对文字：", "Paste the pairing text from the host: "))
		host, _ := os.Hostname()
		info, err := device.Pair(ctx, device.Options{Secrets: sec}, code, host, func(sas string) {
			fmt.Printf(t("\n请核对：主机上显示的数字应该是  %s  。一致就在主机上确认。\n", "\nCheck: the host should show  %s . Confirm on the host if it matches.\n"), sas)
		})
		if err != nil {
			die(err)
		}
		fmt.Printf(t("配对成功，设备 ID：%s\n", "Paired. Device ID: %s\n"), info.DeviceID)
	case "run":
		fs := flag.NewFlagSet("connect run", flag.ExitOnError)
		listen := fs.String("listen", "127.0.0.1:8766", "")
		_ = fs.Parse(args[1:])
		d, err := device.Open(ctx, device.Options{Secrets: sec, Logf: func(f string, a ...any) { log.Printf(f, a...) }})
		if err != nil {
			die(errors.New(t("这台设备还没配对，先运行 omnillm connect pair。", "This device is not paired yet. Run omnillm connect pair first.")))
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
		if err := device.Forget(ctx, sec); err != nil {
			die(err)
		}
		fmt.Println(t("这台设备的配对已删除。主机上的设备列表里它还在，可以在主机上移除。", "This device's pairing is deleted. It still shows in the hub's device list; remove it there too."))
	case "models":
		d := openDevice(ctx, sec)
		defer d.Close()
		ms, err := d.Models(ctx)
		if err != nil {
			die(err)
		}
		for _, m := range ms {
			note := ""
			if m.HumanOnly {
				note = t("（只在对话里用）", " (conversations only)")
			}
			fmt.Printf("%-28s %s%s\n", m.ID, m.DisplayName, note)
		}
	case "chat":
		fs := flag.NewFlagSet("connect chat", flag.ExitOnError)
		model := fs.String("model", "", "")
		_ = fs.Parse(args[1:])
		// Ctrl-C stops a reply, not the chat: only SIGTERM ends it.
		stop()
		cctx, cstop := signal.NotifyContext(context.Background(), syscall.SIGTERM)
		defer cstop()
		d := openDevice(cctx, sec)
		defer d.Close()
		chatLoop(cctx, d, *model)
	case "ask":
		fs := flag.NewFlagSet("connect ask", flag.ExitOnError)
		model := fs.String("model", "", "")
		var imagePaths multi
		fs.Var(&imagePaths, "image", "attach an image file (repeatable)")
		_ = fs.Parse(args[1:])
		if fs.NArg() == 0 && len(imagePaths) == 0 {
			usage()
		}
		var images []omnillm.Image
		for _, p := range imagePaths {
			im, err := readImage(p)
			if err != nil {
				die(err)
			}
			images = append(images, im)
		}
		d := openDevice(ctx, sec)
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
		usage()
	}
}

// ---------------- relay ----------------

func relayCmd(args []string) {
	log.SetFlags(log.LstdFlags)
	if len(args) > 0 && args[0] == "token" {
		tok := hex.EncodeToString(e2e.RandomBytes(32))
		sum := sha256.Sum256([]byte(tok))
		fmt.Printf(t("主机登记令牌（放在主机的 OMNILLM_RELAY_TOKEN，保密）：\n%s\n连接服务用的哈希（--hub-token-hash，可以公开）：\n%x\n",
			"Hub registration token (OMNILLM_RELAY_TOKEN on the hub; keep secret):\n%s\nHash for the relay (--hub-token-hash; not secret):\n%x\n"), tok, sum)
		return
	}
	fs := flag.NewFlagSet("relay", flag.ExitOnError)
	listen := fs.String("listen", ":443", "")
	domain := fs.String("autocert", "", "get a TLS certificate for this domain from Let's Encrypt")
	cache := fs.String("cert-cache", dataDir("relay-certs"), "")
	certFile := fs.String("tls-cert", "", "")
	keyFile := fs.String("tls-key", "", "")
	open := fs.Bool("open", false, "let any hub register (a public relay)")
	behindProxy := fs.Bool("behind-tls-proxy", false, "plain HTTP on loopback behind a TLS reverse proxy")
	var hashes, origins multi
	fs.Var(&hashes, "hub-token-hash", "sha256 of an allowed hub token (repeatable)")
	fs.Var(&origins, "web-origin", "origin of a web client allowed to open device mailboxes (repeatable; default the project's)")
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
		die(errors.New(t("没有允许的主机：用 --hub-token-hash 或 --open。", "No hubs allowed: pass --hub-token-hash or --open.")))
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
			die(errors.New(t("--behind-tls-proxy 只能监听 127.0.0.1。", "--behind-tls-proxy may only listen on 127.0.0.1.")))
		}
		err = srv.ListenAndServe()
	default:
		die(errors.New(t("连接服务必须用 TLS：用 --autocert、--tls-cert，或在 TLS 反向代理后面用 --behind-tls-proxy。",
			"The relay requires TLS: use --autocert, --tls-cert, or --behind-tls-proxy behind a TLS reverse proxy.")))
	}
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		die(err)
	}
}

type multi []string

func (m *multi) String() string     { return strings.Join(*m, ",") }
func (m *multi) Set(v string) error { *m = append(*m, v); return nil }
