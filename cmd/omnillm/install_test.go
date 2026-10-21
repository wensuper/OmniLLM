package main

import (
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
	if got := runningLine(Status{Version: "0.2.0", Providers: 1, Devices: 3}); got != "OmniLLM is running (0.2.0). 1 model source, 3 devices." {
		t.Fatal(got)
	}
	if got := nextStep(Status{}); !strings.Contains(got, "isn't running") || !strings.Contains(got, "settings are kept") {
		t.Fatal(got)
	}
	t.Setenv("OMNILLM_LANG", "zh")
	if got := runningLine(Status{Version: "0.2.0", Providers: 2, Devices: 0}); got != "OmniLLM 在运行（0.2.0）。模型来源 2 个，设备 0 台。" {
		t.Fatal(got)
	}
	for _, st := range []Status{{}, {Running: true}, {Running: true, Providers: 1}, {Running: true, Providers: 1, RelayConfigured: true}, {Running: true, Providers: 1, RelayConfigured: true, RelayUp: true}} {
		if s := nextStep(st); strings.Contains(s, "主机") || strings.Contains(s, "连接服务") {
			t.Fatalf("%+v: %s", st, s)
		}
	}
}
