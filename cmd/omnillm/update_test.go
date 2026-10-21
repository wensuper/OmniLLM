package main

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestAutoAction(t *testing.T) {
	newer := updateFile{Latest: "v0.2.0-alpha.30"}
	cases := []struct {
		name    string
		cur     string
		u       updateFile
		quiet   time.Duration
		service bool
		want    string
	}{
		{"idle hub installs", "0.2.0-alpha.21", newer, time.Hour, true, "install"},
		{"someone is chatting", "0.2.0-alpha.21", newer, time.Minute, true, "busy"},
		{"up to date", "0.2.0-alpha.30", newer, time.Hour, true, "wait"},
		{"turned off", "0.2.0-alpha.21", updateFile{Latest: "v0.2.0-alpha.30", Manual: true}, time.Hour, true, "wait"},
		{"rolled back once", "0.2.0-alpha.21", updateFile{Latest: "v0.2.0-alpha.30", Skip: "v0.2.0-alpha.30"}, time.Hour, true, "wait"},
		{"no service to come back", "0.2.0-alpha.21", newer, time.Hour, false, "wait"},
		{"dev build never replaces itself", "0.1.0-dev", newer, time.Hour, true, "wait"},
		{"older release is not an update", "0.2.0", newer, time.Hour, true, "wait"},
		{"failed an hour ago", "0.2.0-alpha.21", updateFile{Latest: "v0.2.0-alpha.30", Error: "0.2.0-alpha.30", ErrorAt: time.Now().Add(-time.Hour)}, time.Hour, true, "wait"},
		{"failed two days ago: try again", "0.2.0-alpha.21", updateFile{Latest: "v0.2.0-alpha.30", Error: "v0.2.0-alpha.30", ErrorAt: time.Now().Add(-48 * time.Hour)}, time.Hour, true, "install"},
		{"an older failure doesn't hold back a newer release", "0.2.0-alpha.21", updateFile{Latest: "v0.2.0-alpha.31", Error: "v0.2.0-alpha.30", ErrorAt: time.Now()}, time.Hour, true, "install"},
	}
	for _, c := range cases {
		if got := autoAction(c.cur, c.u, c.quiet, c.service); got != c.want {
			t.Errorf("%s: autoAction = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestSkipMatchesWithOrWithoutV(t *testing.T) {
	// install.go records the running version ("0.2.0-alpha.30", no v);
	// GitHub's tag has one. Both name the same release.
	u := updateFile{Latest: "v0.2.0-alpha.30", Skip: "0.2.0-alpha.30"}
	if v := u.newer("0.2.0-alpha.21"); v != "" {
		t.Fatalf("skipped release still offered: %q", v)
	}
}

func TestStatusUpdateLine(t *testing.T) {
	t.Setenv("OMNILLM_LANG", "en")
	old := version
	version = "0.2.0-alpha.21"
	defer func() { version = old }()
	if l := statusUpdateLine(version, updateFile{Latest: "v0.2.0-alpha.21"}); l != "" {
		t.Errorf("up to date, got %q", l)
	}
	if l := statusUpdateLine(version, updateFile{Latest: "v0.2.0-alpha.22"}); !strings.Contains(l, "installs by itself") {
		t.Errorf("auto: %q", l)
	}
	if l := statusUpdateLine(version, updateFile{Latest: "v0.2.0-alpha.22", Manual: true}); !strings.Contains(l, "omnillm update") || strings.Contains(l, "by itself") {
		t.Errorf("manual: %q", l)
	}
	if l := statusUpdateLine(version, updateFile{Latest: "v0.2.0-alpha.22", Error: "v0.2.0-alpha.22"}); !strings.Contains(l, "didn’t work") {
		t.Errorf("failed: %q", l)
	}
}

func TestInstallerArgsKeepTheRole(t *testing.T) {
	if a := installerArgs("device"); len(a) != 1 || a[0] != "--device" {
		t.Errorf("device: %v", a)
	}
	if a := installerArgs("hub"); len(a) != 1 || a[0] != "--no-open" {
		t.Errorf("hub: %v", a)
	}
}

func TestUpdateLockOneAtATime(t *testing.T) {
	dir := t.TempDir()
	release, ok := updateLock(dir)
	if !ok {
		t.Fatal("first lock refused")
	}
	if _, ok := updateLock(dir); ok {
		t.Fatal("second lock granted while the first is held")
	}
	release()
	if r, ok := updateLock(dir); !ok {
		t.Fatal("lock not free after release")
	} else {
		r()
	}
}

func TestRoleFromDataDirs(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("OMNILLM_HOME", home+"/omni")
	if r := role(); r != "" {
		t.Fatalf("nothing set up: %q", r)
	}
	if err := mkdir(home + "/omni/device"); err != nil {
		t.Fatal(err)
	}
	if r := role(); r != "" {
		t.Fatalf("a device folder alone (the program run from elsewhere) is no setup: %q", r)
	}
	if err := writeFile(home+"/omni/device/"+deviceMark, ""); err != nil {
		t.Fatal(err)
	}
	if r := role(); r != "device" {
		t.Fatalf("device only: %q", r)
	}
	if err := mkdir(home + "/omni/hub"); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(home+"/omni/hub/state.json", "{}"); err != nil {
		t.Fatal(err)
	}
	if r := role(); r != "hub" {
		t.Fatalf("hub: %q", r)
	}
}

func mkdir(p string) error { return os.MkdirAll(p, 0o700) }

func writeFile(p, s string) error { return os.WriteFile(p, []byte(s), 0o600) }

func TestSavedSettingsRoundTrip(t *testing.T) {
	relay, web := "wss://relay.example.com/x?a=1&b=2", "https://web.example.com"
	plist := "<key>OMNILLM_RELAY</key><string>" + xmlEscape(relay) + "</string><key>OMNILLM_WEB_APP</key><string>" + xmlEscape(web) + "</string>"
	unit := "[Service]\nEnvironment=OMNILLM_RELAY=" + systemdQuote(relay) + "\nEnvironment=OMNILLM_WEB_APP=" + systemdQuote(web) + "\n"
	for name, body := range map[string]string{"launchd": plist, "systemd": unit} {
		got := savedSettings(body)
		if got["OMNILLM_RELAY"] != relay || got["OMNILLM_WEB_APP"] != web {
			t.Errorf("%s: %v", name, got)
		}
	}
	if got := savedSettings("[Service]\nExecStart=x\n"); len(got) != 0 {
		t.Errorf("nothing saved: %v", got)
	}
}

func TestUpdateLockLeftByAGoneProcess(t *testing.T) {
	dir := t.TempDir()
	// A pid that cannot be running.
	if err := writeFile(dir+"/update.lock", "999999999\n"); err != nil {
		t.Fatal(err)
	}
	r, ok := updateLock(dir)
	if !ok {
		t.Fatal("a lock whose holder is gone still blocks updates")
	}
	r()
}

func TestNoteHandedOff(t *testing.T) {
	dir := t.TempDir()
	old := version
	version = "0.2.0-alpha.21"
	defer func() { version = old }()
	writeUpdate(dir, updateFile{Latest: "v0.2.0-alpha.22"})
	noteHandedOff(dir) // nothing left behind: nothing changes
	if u := readUpdate(dir); u.Error != "" {
		t.Fatalf("no status file, yet error %q", u.Error)
	}
	_ = writeFile(dir+"/update.status", "1\n")
	noteHandedOff(dir)
	if u := readUpdate(dir); !sameVersion(u.Error, "0.2.0-alpha.22") {
		t.Fatalf("failed installer not recorded: %+v", u)
	}
	if exists(dir + "/update.status") {
		t.Fatal("status file read twice")
	}
}
