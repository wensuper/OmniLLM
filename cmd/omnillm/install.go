package main

// "omnillm install": everything with defaults, no questions (design §10.1;
// INSTALL.md is written for an AI agent to run it). It installs the binary
// for this user, runs the hub as a background service, connects a local
// Ollama if one is running, and opens the console. Running it again is
// safe: it updates the service and restarts the hub.

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/wensuper/OmniLLM/device"
	"github.com/wensuper/OmniLLM/internal/control"
	"github.com/wensuper/OmniLLM/internal/e2e"
	"github.com/wensuper/OmniLLM/internal/relaylink"
)

const (
	launchdLabel = "dev.omnillm.hub"
	systemdUnit  = "omnillm-hub.service"
)

// Status is what "omnillm status --json" prints for people and agents.
type Status struct {
	Running         bool   `json:"running"`
	Version         string `json:"version,omitempty"`
	Providers       int    `json:"providers"`
	Devices         int    `json:"devices"`
	RelayConfigured bool   `json:"relay_configured"`
	RelayUp         bool   `json:"relay_up"`
	Console         bool   `json:"console"`
	Service         string `json:"service,omitempty"` // launchd | systemd | none
	// Headless: no display on this machine, so the console cannot be
	// opened here and the person types a key in the terminal instead.
	Headless bool   `json:"headless"`
	Next     string `json:"next"` // the one thing to do next, for a person
}

func installCmd(args []string) {
	noOpen, deviceOnly := false, false
	for _, a := range args {
		switch a {
		case "--no-open":
			noOpen = true
		case "--device":
			deviceOnly = true
		}
	}
	if os.Geteuid() == 0 {
		// Root would leave root-owned files in the person's home folder.
		die(say(t("不要用 root 或 sudo 运行：OmniLLM 只为当前用户安装。", "Don't run this as root or with sudo: OmniLLM installs for the current user only.")))
	}
	if deviceOnly {
		installDevice()
		return
	}
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		die(say(t("这一版的一步安装只支持 macOS 和 Linux。", "One-step install supports macOS and Linux in this version.")))
	}
	home, err := os.UserHomeDir()
	if err != nil {
		die(err)
	}
	bin, err := installBinary(home)
	if err != nil {
		die(err)
	}
	ensureOnPath(home)
	dir := dataDir("hub")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		die(err)
	}
	env, err := secretEnv(dir)
	if err != nil {
		die(err)
	}
	if env == nil {
		env = map[string]string{}
	}
	// A login service has no LANG: without this, what the hub says back to
	// `omnillm hub pair` (the question, the pairing text, errors) and its
	// log would always be English for a person who installed in Chinese.
	env["OMNILLM_LANG"] = lang()
	service, err := installService(home, bin, dir, env)
	if err != nil {
		die(err)
	}
	st, ok := waitRunning(15 * time.Second)
	if !ok {
		die(say(t("主机没有启动起来。日志在 ", "The hub did not start. Its log is at ") + filepath.Join(dir, "hub.log")))
	}
	if st.Providers == 0 {
		// A local model needs no key and sends nothing off the machine.
		_ = control.Call(filepath.Join(dir, "control.sock"), control.Request{Op: "ollama.auto"}, func(control.Reply) (bool, bool) { return false, false })
		st, _ = waitRunning(time.Second)
	}
	st.Service = service
	st.Headless = headless()
	fmt.Println(t("OmniLLM 已安装，主机在后台运行（开机自动启动）。", "OmniLLM is installed; the hub runs in the background and starts at login."))
	if !noOpen && !st.Headless {
		if url, err := loginURL(); err == nil && openLogin(url) == nil {
			fmt.Println(t("控制台已在浏览器里打开。", "The console is open in your browser."))
		} else {
			fmt.Println(runnable(t("在浏览器里打开控制台：运行 omnillm open", "To open the console, run: omnillm open")))
		}
	}
	st.Next = nextStep(st)
	fmt.Println(runnable(st.Next))
	b, _ := json.Marshal(st)
	fmt.Println("OMNILLM_STATUS " + string(b))
}

// headless reports a machine without a display (a server, an SSH session).
func headless() bool {
	if runtime.GOOS == "darwin" {
		return os.Getenv("SSH_CONNECTION") != "" && os.Getenv("TERM_PROGRAM") == ""
	}
	return os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == ""
}

func nextStep(st Status) string {
	switch {
	case !st.Running:
		return t("主机没有运行：运行 omnillm install。", "The hub is not running: run omnillm install.")
	case st.Providers == 0 && st.Headless:
		return t("下一步：这台机器没有浏览器。请在这台机器的终端里亲自运行 omnillm hub provider add anthropic（或 openai、openai-compatible），然后在提示时粘贴 API Key；Key 不要发到聊天里。",
			"Next: this machine has no browser. In this machine's terminal, run omnillm hub provider add anthropic (or openai, openai-compatible) yourself and paste the API key when asked; don't send the key in a chat.")
	case st.Providers == 0:
		return t("下一步：接入一个模型。用 Claude 或 ChatGPT 订阅的，在这台电脑上装好 Claude Code 或 Codex 并登录一次，再运行 omnillm status，会自动接上；也可以在控制台里粘贴 API Key（omnillm open），或安装 Ollama 用本地模型。",
			"Next: connect a model. With a Claude or ChatGPT subscription, install Claude Code or Codex on this computer and log in once, then run omnillm status: it connects by itself. Or paste an API key in the console (omnillm open), or install Ollama for local models.")
	case !st.RelayConfigured:
		return t("已经可以在这台电脑上使用。要让其他设备连进来，需要先配置连接服务。",
			"Ready on this computer. To let other devices connect, a connection service must be set up first.")
	case st.Devices == 0:
		return t("下一步：在控制台的“设备”里添加手机或其他电脑。", "Next: add a phone or another computer under Devices in the console.")
	}
	return t("一切正常。", "All set.")
}

// installBinary copies this executable to ~/.local/bin/omnillm (0755), so
// the service has a stable path.
func installBinary(home string) (string, error) {
	self, err := os.Executable()
	if err != nil {
		return "", err
	}
	self, _ = filepath.EvalSymlinks(self)
	dst := filepath.Join(home, ".local", "bin", "omnillm")
	if self == dst {
		return dst, nil
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return "", err
	}
	in, err := os.Open(self)
	if err != nil {
		return "", err
	}
	defer in.Close()
	tmp := dst + ".new"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return "", err
	}
	if err := out.Close(); err != nil {
		return "", err
	}
	return dst, os.Rename(tmp, dst)
}

// secretEnv picks where secrets live for the service. macOS: the keychain
// (nothing to set). Linux: a background service usually has no unlocked
// keyring, so secrets go into an encrypted file whose key is in a separate
// 0600 file next to it; that is as strong as the account's file
// permissions, and INSTALL.md says so.
func secretEnv(dir string) (map[string]string, error) {
	if runtime.GOOS == "darwin" {
		return nil, nil
	}
	kf := filepath.Join(dir, "master.key")
	if err := ensureMasterKey(kf); err != nil {
		return nil, err
	}
	return map[string]string{"OMNILLM_MASTER_KEY_FILE": kf}, nil
}

// ensureMasterKey creates kf once (exclusively, 0600) or checks the one
// that is there. It never replaces an existing key.
func ensureMasterKey(kf string) error {
	if b, err := os.ReadFile(kf); err == nil {
		if raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(b))); err != nil || len(raw) != 32 {
			// Never silently replace a key: the secrets file depends on it.
			return sayf(t("主密钥文件 %s 损坏，请检查或从备份恢复。", "The master key file %s is damaged; check it or restore it from a backup."), kf)
		}
	} else if errors.Is(err, os.ErrNotExist) {
		k := base64.StdEncoding.EncodeToString(e2e.RandomBytes(32)) + "\n"
		f, err := os.OpenFile(kf+".new", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			return err
		}
		if _, err := f.WriteString(k); err != nil {
			f.Close()
			return err
		}
		if err := f.Sync(); err != nil {
			f.Close()
			return err
		}
		f.Close()
		defer os.Remove(kf + ".new")
		return os.Link(kf+".new", kf) // fails if one appeared meanwhile
	} else {
		return err
	}
	return nil
}

func installService(home, bin, dir string, env map[string]string) (string, error) {
	switch runtime.GOOS {
	case "darwin":
		return "launchd", installLaunchd(home, bin, dir, env)
	default:
		if _, err := exec.LookPath("systemctl"); err == nil {
			if err := installSystemd(home, bin, dir, env); err == nil {
				return "systemd", nil
			}
		}
		return "none", startDetached(bin, dir, env)
	}
}

func xmlEscape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;").Replace(s)
}

func installLaunchd(home, bin, dir string, env map[string]string) error {
	plist := filepath.Join(home, "Library", "LaunchAgents", launchdLabel+".plist")
	var envXML strings.Builder
	for k, v := range env {
		fmt.Fprintf(&envXML, "<key>%s</key><string>%s</string>", xmlEscape(k), xmlEscape(v))
	}
	if relay := relayFromEnv(); relay != "" {
		fmt.Fprintf(&envXML, "<key>OMNILLM_RELAY</key><string>%s</string>", xmlEscape(relay))
	}
	if web := webAppFromEnv(); web != "" {
		fmt.Fprintf(&envXML, "<key>OMNILLM_WEB_APP</key><string>%s</string>", xmlEscape(web))
	}
	body := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>Label</key><string>` + launchdLabel + `</string>
<key>ProgramArguments</key><array><string>` + xmlEscape(bin) + `</string><string>hub</string><string>run</string></array>
<key>EnvironmentVariables</key><dict>` + envXML.String() + `</dict>
<key>RunAtLoad</key><true/>
<key>KeepAlive</key><true/>
<key>ProcessType</key><string>Standard</string>
<key>StandardOutPath</key><string>` + xmlEscape(filepath.Join(dir, "hub.log")) + `</string>
<key>StandardErrorPath</key><string>` + xmlEscape(filepath.Join(dir, "hub.log")) + `</string>
</dict></plist>
`
	if err := os.MkdirAll(filepath.Dir(plist), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(plist, []byte(body), 0o644); err != nil {
		return err
	}
	domain := "gui/" + strconv.Itoa(os.Getuid())
	_ = exec.Command("launchctl", "bootout", domain+"/"+launchdLabel).Run()
	// bootout returns before the old service is gone; bootstrapping
	// meanwhile fails ("Input/output error") and an upgrade would leave no
	// service at all. Wait for it to go, then retry a few times.
	for i := 0; i < 50 && exec.Command("launchctl", "print", domain+"/"+launchdLabel).Run() == nil; i++ {
		time.Sleep(100 * time.Millisecond)
	}
	var out []byte
	var err error
	for try := 0; try < 5; try++ {
		if out, err = exec.Command("launchctl", "bootstrap", domain, plist).CombinedOutput(); err == nil {
			return nil
		}
		time.Sleep(time.Second)
	}
	// Older systems or a non-GUI session: fall back to load.
	if out2, err2 := exec.Command("launchctl", "load", "-w", plist).CombinedOutput(); err2 != nil {
		return fmt.Errorf("launchctl: %s %s", strings.TrimSpace(string(out)), strings.TrimSpace(string(out2)))
	}
	return nil
}

// systemdQuote quotes a value for a unit file: systemd expands %
// specifiers and $VARIABLES even inside quotes.
func systemdQuote(v string) string {
	return strings.NewReplacer("%", "%%", "$", "$$").Replace(strconv.Quote(v))
}

// relayFromEnv passes OMNILLM_RELAY to the service only if it is a valid
// relay address.
func relayFromEnv() string {
	r := os.Getenv("OMNILLM_RELAY")
	if r == "" {
		return ""
	}
	if _, err := relaylink.CheckURL(r); err != nil {
		die(fmt.Errorf("OMNILLM_RELAY: %w", err))
	}
	return r
}

// webAppFromEnv passes OMNILLM_WEB_APP (a self-hosted web client) to the
// service: https, or http on this machine for testing.
func webAppFromEnv() string {
	w := os.Getenv("OMNILLM_WEB_APP")
	if w == "" {
		return ""
	}
	u, err := url.Parse(w)
	local := err == nil && u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1")
	if err != nil || (u.Scheme != "https" && !local) || u.Host == "" || u.RawQuery != "" || u.Fragment != "" || u.User != nil {
		die(errors.New("OMNILLM_WEB_APP must be an https:// address"))
	}
	return w
}

func installSystemd(home, bin, dir string, env map[string]string) error {
	unit := filepath.Join(home, ".config", "systemd", "user", systemdUnit)
	var envLines strings.Builder
	for k, v := range env {
		fmt.Fprintf(&envLines, "Environment=%s=%s\n", k, systemdQuote(v))
	}
	if relay := relayFromEnv(); relay != "" {
		fmt.Fprintf(&envLines, "Environment=OMNILLM_RELAY=%s\n", systemdQuote(relay))
	}
	if web := webAppFromEnv(); web != "" {
		fmt.Fprintf(&envLines, "Environment=OMNILLM_WEB_APP=%s\n", systemdQuote(web))
	}
	body := "[Unit]\nDescription=OmniLLM hub\nAfter=network-online.target\n\n[Service]\n" +
		"ExecStart=" + systemdQuote(bin) + " hub run\n" + envLines.String() +
		"StandardOutput=append:" + filepath.Join(dir, "hub.log") + "\nStandardError=append:" + filepath.Join(dir, "hub.log") + "\n" +
		"Restart=on-failure\nRestartSec=3\nNoNewPrivileges=yes\nPrivateTmp=yes\nUMask=0077\n\n[Install]\nWantedBy=default.target\n"
	if err := os.MkdirAll(filepath.Dir(unit), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(unit, []byte(body), 0o644); err != nil {
		return err
	}
	for _, args := range [][]string{{"--user", "daemon-reload"}, {"--user", "enable", systemdUnit}, {"--user", "restart", systemdUnit}} {
		if out, err := exec.Command("systemctl", args...).CombinedOutput(); err != nil {
			return fmt.Errorf("systemctl %s: %s", strings.Join(args, " "), strings.TrimSpace(string(out)))
		}
	}
	// Keep running after logout on servers.
	_ = exec.Command("loginctl", "enable-linger").Run()
	return nil
}

// startDetached runs the hub without a service manager (no autostart).
func startDetached(bin, dir string, env map[string]string) error {
	logf, err := os.OpenFile(filepath.Join(dir, "hub.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	cmd := exec.Command(bin, "hub", "run")
	cmd.Stdout, cmd.Stderr = logf, logf
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	detach(cmd)
	return cmd.Start()
}

func queryStatus() (Status, bool) {
	var st Status
	sock := filepath.Join(dataDir("hub"), "control.sock")
	ok := false
	err := control.Call(sock, control.Request{Op: "status"}, func(r control.Reply) (bool, bool) {
		if json.Unmarshal(r.Data, &st) == nil {
			ok = true
		}
		return false, false
	})
	return st, err == nil && ok
}

func waitRunning(d time.Duration) (Status, bool) {
	deadline := time.Now().Add(d)
	for {
		if st, ok := queryStatus(); ok {
			return st, true
		}
		if time.Now().After(deadline) {
			return Status{}, false
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func loginURL() (string, error) {
	var url string
	err := control.Call(filepath.Join(dataDir("hub"), "control.sock"), control.Request{Op: "console.login"},
		func(r control.Reply) (bool, bool) { url = r.Text; return false, false })
	if err != nil || !strings.HasPrefix(url, "http://127.0.0.1:") {
		return "", errors.New("no console")
	}
	return url, nil
}

func statusCmd(args []string) {
	st, ok := queryStatus()
	st.Running = ok
	st.Headless = headless()
	st.Next = nextStep(st)
	for _, a := range args {
		if a == "--json" {
			b, _ := json.Marshal(st)
			fmt.Println(string(b))
			return
		}
	}
	if !ok {
		fmt.Println(runnable(st.Next))
		os.Exit(1)
	}
	fmt.Printf(t("主机在运行（%s）。模型提供方 %d 个，设备 %d 台。\n", "The hub is running (%s). Providers: %d, devices: %d.\n"), st.Version, st.Providers, st.Devices)
	fmt.Println(runnable(st.Next))
}

func uninstallCmd(args []string) {
	purge := false
	for _, a := range args {
		if a == "--purge" {
			purge = true
		}
	}
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "darwin":
		plist := filepath.Join(home, "Library", "LaunchAgents", launchdLabel+".plist")
		_ = exec.Command("launchctl", "bootout", "gui/"+strconv.Itoa(os.Getuid())+"/"+launchdLabel).Run()
		_ = os.Remove(plist)
	case "linux":
		_ = exec.Command("systemctl", "--user", "disable", "--now", systemdUnit).Run()
		_ = os.Remove(filepath.Join(home, ".config", "systemd", "user", systemdUnit))
		_ = exec.Command("systemctl", "--user", "daemon-reload").Run()
	}
	fmt.Println(t("后台服务已停止并移除。", "The background service is stopped and removed."))
	if purge {
		sec := secrets(dataDir("hub"), "omnillm-hub")
		ctx, stop := signalCtx()
		defer stop()
		if names, err := sec.List(ctx, ""); err == nil {
			for _, n := range names {
				_ = sec.Delete(ctx, n)
			}
		}
		for _, d := range purgeDirs() {
			_ = os.RemoveAll(d)
		}
		if os.Getenv("OMNILLM_HOME") == "" {
			_ = os.Remove(filepath.Dir(dataDir("hub"))) // only if now empty
		}
		fmt.Println(t("数据和保存的 Key 已删除。提供方后台的 Key 本身没有作废，需要的话请到提供方后台作废。",
			"Data and saved keys are deleted. The keys themselves are still valid at the providers; revoke them there if needed."))
	} else {
		fmt.Println(t("数据和 Key 保留着；要一并删除，运行 omnillm uninstall --purge。", "Data and keys are kept; to delete them too, run omnillm uninstall --purge."))
	}
}

// purgeDirs is what --purge deletes: only the folders OmniLLM creates,
// never their parent (OMNILLM_HOME may point anywhere, even at $HOME).
func purgeDirs() []string {
	return []string{dataDir("hub"), dataDir("device"), dataDir("relay-certs"), dataDir("codex-home")}
}

// installDevice sets this machine up as a device only (a server that uses
// the hub's models): the program in ~/.local/bin, no background service,
// nothing else. Then it pairs, if a person is at the terminal.
func installDevice() {
	home, err := os.UserHomeDir()
	if err != nil {
		die(err)
	}
	bin, err := installBinary(home)
	if err != nil {
		die(err)
	}
	fmt.Printf(t("已安装到 %s（只是一个程序，没有后台服务）。\n", "Installed to %s (just the program; no background service).\n"), bin)
	ensureOnPath(home)
	dir := dataDir("device")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		die(err)
	}
	if device.Paired(context.Background(), secrets(dir, "omnillm-device")) {
		fmt.Println(runnable(t("这台机器已经配对过，可以直接用：omnillm connect chat", "This machine is already paired. Ready: omnillm connect chat")))
		return
	}
	tty, err := os.Open("/dev/tty")
	if err != nil {
		fmt.Println(runnable(t("下一步：运行 omnillm connect pair，粘贴主机给的配对文字。", "Next: run omnillm connect pair and paste the pairing text from the hub.")))
		return
	}
	tty.Close()
	cmd := exec.Command(bin, "connect", "pair")
	if in, err := os.Open("/dev/tty"); err == nil {
		defer in.Close()
		cmd.Stdin = in
	}
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		os.Exit(1)
	}
	fmt.Println(runnable(t("可以用了：在终端里运行 omnillm connect chat 对话，或 omnillm connect ask \"问题\" 问一句。", "Ready: run omnillm connect chat to chat, or omnillm connect ask \"question\" for one answer.")))
}
