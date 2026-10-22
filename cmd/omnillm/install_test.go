package main

import (
	"github.com/wensuper/OmniLLM/internal/provider"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	if s := nextStep(headless); !strings.Contains(s, "omnillm hub pair") || strings.Contains(s, "omnillm open") {
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
