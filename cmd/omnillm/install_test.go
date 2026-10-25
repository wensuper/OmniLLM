package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wensuper/OmniLLM/internal/provider"
)

func TestPurgeNeverTakesTheParent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("OMNILLM_HOME", home)
	for _, d := range purgeDirs() {
		if d == home || !strings.HasPrefix(d, home+string(os.PathSeparator)) || filepath.Dir(d) != home {
			t.Fatalf("purge would delete %s (home %s)", d, home)
		}
	}
}

func TestMasterKeyCreatedOnceNeverReplaced(t *testing.T) {
	kf := filepath.Join(t.TempDir(), "master.key")
	if err := ensureMasterKey(kf); err != nil {
		t.Fatal(err)
	}
	st, _ := os.Stat(kf)
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("mode %v", st.Mode().Perm())
	}
	first, _ := os.ReadFile(kf)
	if err := ensureMasterKey(kf); err != nil {
		t.Fatal(err)
	}
	again, _ := os.ReadFile(kf)
	if string(first) != string(again) {
		t.Fatal("the key was replaced")
	}
	_ = os.WriteFile(kf, []byte("short"), 0o600)
	if err := ensureMasterKey(kf); err == nil {
		t.Fatal("a damaged key was accepted (or silently replaced)")
	}
	if b, _ := os.ReadFile(kf); string(b) != "short" {
		t.Fatal("a damaged key was overwritten")
	}
}

func TestSystemdQuote(t *testing.T) {
	got := systemdQuote(`/home/a%b/$HOME/"x"`)
	if !strings.Contains(got, "%%") || !strings.Contains(got, "$$HOME") || !strings.HasPrefix(got, `"`) {
		t.Fatalf("%s", got)
	}
}

func TestNextStepAlwaysSaysSomething(t *testing.T) {
	for _, st := range []Status{{}, {Running: true}, {Running: true, Providers: 1}, {Running: true, Providers: 1, RelayConfigured: true}, {Running: true, Providers: 1, RelayConfigured: true, Devices: 1}, {Running: true, Providers: 1, RelayConfigured: true, RelayUp: true, Devices: 1}} {
		if nextStep(st) == "" {
			t.Fatalf("%+v", st)
		}
	}
}

func TestStatusLinesFollowTheGlossary(t *testing.T) {
	t.Setenv("OMNILLM_LANG", "en")
	if got := runningLine(Status{Version: "0.2.0", Providers: 1, Devices: 3, Online: 1}); got != "OmniLLM is running (0.2.0). Devices: 3 (1 online), model sources: 1." {
		t.Fatal(got)
	}
	if got := nextStep(Status{}); !strings.Contains(got, "isn’t running") || !strings.Contains(got, "settings are kept") {
		t.Fatal(got)
	}
	t.Setenv("OMNILLM_LANG", "zh")
	if got := runningLine(Status{Version: "0.2.0", Providers: 2, Devices: 0}); got != "OmniLLM 在运行（0.2.0）。设备 0 台（0 台在线），模型来源 2 个。" {
		t.Fatal(got)
	}
	for _, st := range []Status{{}, {Running: true}, {Running: true, Providers: 1}, {Running: true, Providers: 1, RelayConfigured: true}, {Running: true, Providers: 1, RelayConfigured: true, RelayUp: true}} {
		if s := nextStep(st); strings.Contains(s, "主机") || strings.Contains(s, "连接服务") {
			t.Fatalf("%+v: %s", st, s)
		}
	}
	// A changed region comes first, with the command that resumes.
	st := Status{Running: true, Providers: 1, RelayConfigured: true, RelayUp: true, Devices: 1, RegionPaused: true}
	if s := nextStep(st); !strings.Contains(s, "所在的地区变了，已暂停使用模型。") || !strings.Contains(s, "omnillm hub region confirm") || strings.Contains(s, "  ") {
		t.Fatal(s)
	}
	t.Setenv("OMNILLM_LANG", "en")
	if s := nextStep(st); !strings.HasPrefix(s, "This ") || !strings.Contains(s, "’s region changed; models are paused.") {
		t.Fatal(s)
	}
}

// The last sentence an assistant passes on matches the screen: the QR
// code the install just opened, or the one command that shows it again.
func TestNextStepForTheFirstDevice(t *testing.T) {
	t.Setenv("OMNILLM_LANG", "zh")
	ready := Status{Running: true, Providers: 1, RelayConfigured: true, RelayUp: true}
	opened := ready
	opened.Opened = true
	if s := nextStep(opened); !strings.Contains(s, "浏览器里已经打开了添加手机的二维码") || !strings.Contains(s, "“一致，添加”") || strings.Contains(s, "omnillm open") {
		t.Fatal(s)
	}
	if s := nextStep(ready); !strings.Contains(s, "omnillm open --add") {
		t.Fatal(s)
	}
	headless := ready
	headless.Headless = true
	if s := nextStep(headless); !strings.Contains(s, "omnillm pair") || strings.Contains(s, "omnillm open") {
		t.Fatal(s)
	}
	t.Setenv("OMNILLM_LANG", "en")
	if s := nextStep(opened); !strings.HasPrefix(s, "The QR code for adding your phone is open in your browser") || !strings.Contains(s, "“Matches, add”") {
		t.Fatal(s)
	}
	// A device already added: nothing about QR codes.
	ready.Devices = 1
	if s := nextStep(ready); strings.Contains(s, "QR") {
		t.Fatal(s)
	}
}

// The assistant running the install comes first while no model is
// connected: its own subscription is the sign-in asked for. Once any model
// is connected (the person chose a key instead), next moves on.
func TestNextStepAsksForTheInstallingAssistantsSignIn(t *testing.T) {
	t.Setenv("OMNILLM_LANG", "zh")
	st := Status{Running: true, RelayConfigured: true, RelayUp: true, Assistant: "codex"}
	if s := nextStep(st); !strings.Contains(s, "你在用 Codex，但 OmniLLM 还用不上它的登录") || !strings.Contains(s, "omnillm login chatgpt") {
		t.Fatal(s)
	}
	if s := nextStep(Status{Running: true, Providers: 1, RelayConfigured: true, RelayUp: true, Assistant: "codex"}); strings.Contains(s, "omnillm login") {
		t.Fatal("a connected model moves on:", s)
	}
	st.LoginOpened = true
	if s := nextStep(st); !strings.Contains(s, "已经打开一个终端窗口") || strings.Contains(s, "omnillm login") {
		t.Fatal(s)
	}
	t.Setenv("OMNILLM_LANG", "en")
	st = Status{Running: true, Assistant: "claude-code", Headless: true}
	if s := nextStep(st); !strings.Contains(s, "you’re using Claude Code") || !strings.Contains(s, "omnillm login claude yourself") {
		t.Fatal(s)
	}
	// Not running comes before anything else.
	if s := nextStep(Status{Assistant: "codex"}); strings.Contains(s, "Codex") {
		t.Fatal(s)
	}
}

func TestInstallingAssistantFromClaudeCodesVariable(t *testing.T) {
	t.Setenv("CLAUDECODE", "1")
	if k := installingAssistant(); k != "claude-code" {
		t.Fatal(k)
	}
}

func TestUpdateKeepsTheOwnersRelayAndWebApp(t *testing.T) {
	plist := filepath.Join(t.TempDir(), "hub.plist")
	write := func(env map[string]string) {
		if err := os.WriteFile(plist, []byte(hubPlist("/x/omnillm", "/x/data", env)), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// First install, with a self-hosted relay and web client.
	t.Setenv("OMNILLM_RELAY", "wss://relay.example.org/a&b")
	t.Setenv("OMNILLM_WEB_APP", "https://chat.example.org/")
	write(serviceEnv(map[string]string{"K": "v"}, plistEnv(plist)))
	// The update runs with neither set: both are kept.
	t.Setenv("OMNILLM_RELAY", "")
	t.Setenv("OMNILLM_WEB_APP", "")
	got := serviceEnv(nil, plistEnv(plist))
	if got["OMNILLM_RELAY"] != "wss://relay.example.org/a&b" || got["OMNILLM_WEB_APP"] != "https://chat.example.org/" {
		t.Fatalf("lost on update: %v", got)
	}
	if _, ok := got["K"]; ok {
		t.Fatal("only the owner's relay and web client are carried over")
	}
	// The local API's address (or off) is kept the same way; anything but
	// a loopback address is not.
	t.Setenv("OMNILLM_API", "off")
	write(serviceEnv(nil, nil))
	t.Setenv("OMNILLM_API", "")
	if got := serviceEnv(nil, plistEnv(plist)); got["OMNILLM_API"] != "off" {
		t.Fatalf("local API off undone: %v", got)
	}
	t.Setenv("OMNILLM_API", "0.0.0.0:8765")
	if got := serviceEnv(nil, nil); got["OMNILLM_API"] != "" {
		t.Fatalf("a non-loopback API address was carried: %v", got)
	}
	t.Setenv("OMNILLM_API", "127.0.0.1:9000")
	if got := serviceEnv(nil, nil); got["OMNILLM_API"] != "127.0.0.1:9000" {
		t.Fatalf("API address ignored: %v", got)
	}
	t.Setenv("OMNILLM_API", "")
	// "off" is a choice too.
	t.Setenv("OMNILLM_RELAY", "off")
	write(serviceEnv(nil, nil))
	t.Setenv("OMNILLM_RELAY", "")
	if got := serviceEnv(nil, plistEnv(plist)); got["OMNILLM_RELAY"] != "off" {
		t.Fatalf("relay off undone: %v", got)
	}
	// Set in the environment, it wins over the old one.
	t.Setenv("OMNILLM_RELAY", "wss://other.example.org")
	if got := serviceEnv(nil, plistEnv(plist)); got["OMNILLM_RELAY"] != "wss://other.example.org" {
		t.Fatalf("environment ignored: %v", got)
	}
	// A broken old value is dropped, not carried on.
	t.Setenv("OMNILLM_RELAY", "")
	write(map[string]string{"OMNILLM_RELAY": "ws://evil.example.org", "OMNILLM_WEB_APP": "javascript:x"})
	if got := serviceEnv(nil, plistEnv(plist)); len(got) != 0 {
		t.Fatalf("invalid values carried over: %v", got)
	}
	// No earlier install.
	if got := serviceEnv(nil, plistEnv(filepath.Join(t.TempDir(), "none"))); len(got) != 0 {
		t.Fatalf("%v", got)
	}
}

func TestNextStepWithoutModelsIsOneCommandALine(t *testing.T) {
	old := hasDesktopApp
	hasDesktopApp = func(provider.Kind) bool { return false }
	defer func() { hasDesktopApp = old }()
	for _, lang := range []string{"zh", "en"} {
		t.Setenv("OMNILLM_LANG", lang)
		for _, st := range []Status{{Running: true}, {Running: true, Headless: true}} {
			s := nextStep(st)
			lines := strings.Split(s, "\n")
			if len(lines) != 4 || strings.Contains(s, "%!") {
				t.Fatalf("%s %+v: %q", lang, st, s)
			}
			for _, l := range lines[1:] {
				if !strings.HasPrefix(l, "· ") || strings.Count(l, "omnillm ")+strings.Count(l, "claude")+strings.Count(l, "codex login") == 0 {
					t.Errorf("%s: %q", lang, l)
				}
			}
			if st.Headless == strings.Contains(s, "omnillm open") {
				t.Errorf("%s %+v: %q", lang, st, s)
			}
		}
	}
}

// Only the desktop app (Claude.app) and no command-line program: the first
// thing said is that the same subscription works, and the line to run.
func TestNextStepNamesTheDesktopApp(t *testing.T) {
	old := hasDesktopApp
	hasDesktopApp = func(k provider.Kind) bool { return k == provider.ClaudeCode }
	defer func() { hasDesktopApp = old }()
	for _, l := range []string{"zh", "en"} {
		t.Setenv("OMNILLM_LANG", l)
		s := nextStep(Status{Running: true})
		lines := strings.Split(s, "\n")
		if len(lines) != 5 || !strings.Contains(lines[0], "Claude") || strings.Contains(lines[0], "ChatGPT") || !strings.Contains(s, "omnillm login claude") {
			t.Fatalf("%s: %q", l, s)
		}
	}
}

// After a pairing in the terminal: it says the device is ready, and how to
// find the way later ("type omnillm"), spaced as Chinese is written.
func TestPairedNote(t *testing.T) {
	t.Setenv("OMNILLM_LANG", "zh")
	s := pairedNote()
	if !strings.Contains(s, "输入 omnillm") || !strings.Contains(s, "omnillm pair") || strings.Contains(s, "Mac上") || strings.Contains(s, "电脑 上") {
		t.Fatal(s)
	}
	t.Setenv("OMNILLM_LANG", "en")
	if s := pairedNote(); !strings.HasPrefix(s, "The new device can now use") || !strings.Contains(s, "type omnillm") {
		t.Fatal(s)
	}
}

// The console that just opened on its first step is where a model is
// connected: next points there, not at other commands.
func TestNextStepWithTheConsoleOpenAndNoModel(t *testing.T) {
	t.Setenv("OMNILLM_LANG", "en")
	s := nextStep(Status{Running: true, RelayConfigured: true, RelayUp: true, Opened: true})
	if !strings.Contains(s, "browser page that just opened") || strings.Contains(s, "omnillm open") || strings.Contains(s, "omnillm login") {
		t.Fatal(s)
	}
}

// OMNILLM_HOME: a full path, a folder of this user's, made private.
func TestPrivateHome(t *testing.T) {
	t.Setenv("OMNILLM_HOME", "")
	if h, err := privateHome(); h != "" || err != nil {
		t.Fatal(h, err)
	}
	t.Setenv("OMNILLM_HOME", "relative/dir")
	if _, err := privateHome(); err == nil {
		t.Fatal("a relative path is refused")
	}
	if os.Getuid() != 0 {
		t.Setenv("OMNILLM_HOME", "/")
		if _, err := privateHome(); err == nil {
			t.Fatal("a folder of another user (root) is refused")
		}
	}
	d := filepath.Join(t.TempDir(), "h")
	if err := os.MkdirAll(d, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OMNILLM_HOME", d)
	if h, err := privateHome(); h != d || err != nil {
		t.Fatal(h, err)
	}
	if fi, _ := os.Stat(d); fi.Mode().Perm() != 0o700 {
		t.Fatalf("mode %v", fi.Mode().Perm())
	}
}

// A master key someone else could read is never used.
func TestMasterKeyMustBePrivate(t *testing.T) {
	kf := filepath.Join(t.TempDir(), "master.key")
	if err := ensureMasterKey(kf); err != nil {
		t.Fatal(err)
	}
	if err := ensureMasterKey(kf); err != nil {
		t.Fatal("its own key is kept:", err)
	}
	if err := os.Chmod(kf, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ensureMasterKey(kf); err == nil {
		t.Fatal("a readable key is refused")
	}
}

// OMNILLM_HOME's parents: no one else can swap the folder out.
func TestSafeParents(t *testing.T) {
	d := t.TempDir()
	if !safeParents(d) {
		t.Fatal("a temp folder of this user's is safe")
	}
	open := filepath.Join(d, "open")
	if err := os.MkdirAll(open, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.Chmod(open, 0o777)
	if safeParents(open) || safeParents(filepath.Join(open, "x")) {
		t.Fatal("a folder others can write to (not sticky) is not")
	}
	_ = os.Chmod(open, 0o777|os.ModeSticky)
	if !safeParents(open) {
		t.Fatal("a sticky folder (like /tmp) is")
	}
	t.Setenv("OMNILLM_HOME", filepath.Join(d, "link"))
	if err := os.Symlink(d, filepath.Join(d, "link")); err != nil {
		t.Fatal(err)
	}
	if _, err := privateHome(); err == nil {
		t.Fatal("a link is not a data folder")
	}
}

// All set on a server: nothing to open there.
func TestAllSetOnAServer(t *testing.T) {
	t.Setenv("OMNILLM_LANG", "en")
	st := Status{Running: true, Providers: 1, Devices: 1, RelayConfigured: true, RelayUp: true, Headless: true}
	if s := nextStep(st); s != "All set." {
		t.Fatal(s)
	}
}
