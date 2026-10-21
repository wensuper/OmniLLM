package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCompareVersions(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want int
	}{
		{"v0.2.0-alpha.10", "v0.2.0-alpha.9", 1},
		{"0.2.0-alpha.9", "v0.2.0-alpha.9", 0},
		{"v0.2.0", "v0.2.0-alpha.12", 1},
		{"v0.2.1-alpha.1", "v0.2.0", 1},
		{"v0.10.0", "v0.9.9", 1},
		{"v0.2.0-alpha.1", "v0.2.0-beta.1", -1},
		{"v0.2.0-alpha", "v0.2.0-alpha.1", -1},
		{"0.1.0-dev", "v0.2.0-alpha.1", -1},
		{"garbage", "v0.0.1", -1},
	} {
		if got := compareVersions(c.a, c.b); got != c.want {
			t.Errorf("compare(%s, %s) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestNewestRelease(t *testing.T) {
	pre := []release{{Tag: "v0.2.0-alpha.9", Prerelease: true}, {Tag: "v0.2.0-alpha.10", Prerelease: true},
		{Tag: "v0.3.0-alpha.1", Draft: true}, {Tag: "nightly"}}
	if got := newestRelease(pre); got != "v0.2.0-alpha.10" {
		t.Fatal(got)
	}
	// Once a stable release exists, install.sh installs the newest stable.
	mixed := append(pre, release{Tag: "v0.2.0"}, release{Tag: "v0.2.1-alpha.1", Prerelease: true})
	if got := newestRelease(mixed); got != "v0.2.0" {
		t.Fatal(got)
	}
}

func TestUpdateLines(t *testing.T) {
	t.Setenv("OMNILLM_LANG", "zh")
	now := time.Now()
	cur := "0.2.0-alpha.9"
	var s updateState
	if v, inst := versionLine(cur, s); v != "版本 0.2.0-alpha.9" || inst {
		t.Fatal(v)
	}
	if c, click, show := checkLine(cur, s, now); c != "检查更新…" || !click || !show {
		t.Fatal(c)
	}
	s.Checking = true
	if c, click, _ := checkLine(cur, s, now); c != "正在检查更新……" || click {
		t.Fatal(c)
	}
	s = s.checked(cur, "v0.2.0-alpha.9", nil, now)
	if c, click, _ := checkLine(cur, s, now); c != "✓ 已是最新版本" || click {
		t.Fatal(c)
	}
	if c, _, _ := checkLine(cur, s, now.Add(61*time.Second)); c != "检查更新…" {
		t.Fatal("up to date did not go back after a minute: " + c)
	}
	s = s.checked(cur, "", context.DeadlineExceeded, now)
	if c, click, _ := checkLine(cur, s, now); c != "检查失败，点击重试" || !click {
		t.Fatal(c)
	}
	s = s.checked(cur, "v0.2.0-alpha.10", nil, now)
	if v, inst := versionLine(cur, s); v != "有新版本 0.2.0-alpha.10 · 点击更新（正在进行的回答会中断）" || !inst {
		t.Fatal(v)
	}
	if _, _, show := checkLine(cur, s, now); show {
		t.Fatal("check line shown while an update is offered")
	}
	s.Installing = s.Latest
	if v, inst := versionLine(cur, s); v != "正在更新到 0.2.0-alpha.10……" || inst {
		t.Fatal(v)
	}
	s.Installing, s.Failed = "", true
	if v, inst := versionLine(cur, s); v != "更新到 0.2.0-alpha.10 没有成功，点击再试" || !inst {
		t.Fatal(v)
	}
	home, _ := os.UserHomeDir()
	s.Log = filepath.Join(home, ".omnillm", "hub", "update.log")
	if v, _ := versionLine(cur, s); v != "更新到 0.2.0-alpha.10 没有成功，点击再试（日志：~/.omnillm/hub/update.log）" {
		t.Fatal(v)
	}
	t.Setenv("OMNILLM_LANG", "en")
	if v, _ := versionLine(cur, updateState{}); v != "Version 0.2.0-alpha.9" {
		t.Fatal(v)
	}
	if v, _ := versionLine(cur, s); v != "Updating to 0.2.0-alpha.10 didn’t work; click to try again (log: ~/.omnillm/hub/update.log)" {
		t.Fatal(v)
	}
	if v, _ := versionLine(cur, updateState{Latest: "v0.2.0-alpha.10"}); v != "New version 0.2.0-alpha.10 · click to update (a reply in progress is cut off)" {
		t.Fatal(v)
	}
}

func TestLatestRelease(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{"tag_name":"v0.2.0-alpha.5","prerelease":true},{"tag_name":"v0.2.0-alpha.12","prerelease":true}]`))
	}))
	defer srv.Close()
	defer func(u string) { releasesURL = u }(releasesURL)
	releasesURL = srv.URL
	got, err := latestRelease(context.Background(), srv.Client())
	if err != nil || got != "v0.2.0-alpha.12" {
		t.Fatal(got, err)
	}
}

func TestTildePath(t *testing.T) {
	home, _ := os.UserHomeDir()
	for in, want := range map[string]string{
		filepath.Join(home, "a", "b.log"): "~/a/b.log",
		home:                              "~",
		"/var/log/x.log":                  "/var/log/x.log",
		home + "x/y":                      home + "x/y", // a sibling folder, not inside home
	} {
		if got := tildePath(in); got != want {
			t.Errorf("tildePath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestInstalledVersion(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "omnillm")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n[ \"$1\" = version ] && echo 'omnillm 0.2.0-alpha.10'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if v, err := installedVersion(context.Background(), bin); err != nil || v != "0.2.0-alpha.10" {
		t.Fatal(v, err)
	}
	bad := filepath.Join(dir, "other")
	_ = os.WriteFile(bad, []byte("#!/bin/sh\necho hello there world\n"), 0o700)
	if v, err := installedVersion(context.Background(), bad); err == nil {
		t.Fatal("accepted", v)
	}
	if _, err := installedVersion(context.Background(), filepath.Join(dir, "missing")); err == nil {
		t.Fatal("a missing program has a version")
	}
}

func TestAfterAnInstallThatDidNotReplaceTheMenu(t *testing.T) {
	offer := updateState{Latest: "v0.2.0-alpha.10", Installing: "v0.2.0-alpha.10"}
	// The new version is there: the offer is done, run it.
	s, restart := offer.installed(nil, "0.2.0-alpha.10")
	if !restart || s.Latest != "" || s.Installing != "" || s.Failed {
		t.Fatalf("%+v %v", s, restart)
	}
	if _, restart := offer.installed(nil, "0.2.0-alpha.11"); !restart {
		t.Fatal("a newer one than offered is fine too")
	}
	// Still the old version, or the installer failed: offer a retry.
	for _, c := range []struct {
		err  error
		have string
	}{{nil, "0.2.0-alpha.9"}, {errors.New("exit 1"), "0.2.0-alpha.10"}, {nil, ""}} {
		s, restart := offer.installed(c.err, c.have)
		if restart || !s.Failed || s.Latest != offer.Latest || s.Installing != "" {
			t.Fatalf("%v %q: %+v %v", c.err, c.have, s, restart)
		}
	}
}

func TestTrayFollowsTheHubsUpdater(t *testing.T) {
	cur := "0.2.0-alpha.21"
	// A version the hub rolled back is not offered again.
	s := updateState{}.checked(cur, "v0.2.0-alpha.22", nil, time.Now()).fromHub(cur, updateFile{Latest: "v0.2.0-alpha.22", Skip: "0.2.0-alpha.22"})
	if _, inst := versionLine(cur, s); inst {
		t.Fatal("rolled-back version offered")
	}
	// An automatic update that didn't work shows as one to try again.
	s = updateState{}.fromHub(cur, updateFile{Latest: "v0.2.0-alpha.22", Error: "v0.2.0-alpha.22"})
	if !s.Failed || s.Latest == "" {
		t.Fatalf("failed automatic update not shown: %+v", s)
	}
}
