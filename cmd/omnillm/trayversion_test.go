package main

import (
	"context"
	"net/http"
	"net/http/httptest"
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
	if c, click, _ := checkLine(cur, s, now); c != "正在检查更新…" || click {
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
	if v, inst := versionLine(cur, s); v != "有新版本 v0.2.0-alpha.10 · 点击更新" || !inst {
		t.Fatal(v)
	}
	if _, _, show := checkLine(cur, s, now); show {
		t.Fatal("check line shown while an update is offered")
	}
	s.Installing = s.Latest
	if v, inst := versionLine(cur, s); v != "正在更新到 v0.2.0-alpha.10…" || inst {
		t.Fatal(v)
	}
	s.Installing, s.Failed = "", true
	if v, inst := versionLine(cur, s); v != "更新失败，点击重试" || !inst {
		t.Fatal(v)
	}
	t.Setenv("OMNILLM_LANG", "en")
	if v, _ := versionLine(cur, updateState{}); v != "Version 0.2.0-alpha.9" {
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
