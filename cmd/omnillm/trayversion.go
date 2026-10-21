package main

// The menu bar's version and update lines, as in NEXA's menu: the logic
// here has no build tags so it is tested everywhere; tray_darwin.go only
// draws it.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/wensuper/OmniLLM/internal/provider"
)

// semver is a parsed vMAJOR.MINOR.PATCH[-pre.release.N].
type semver struct {
	nums [3]int
	pre  []string
}

var versionRe = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z.]+))?$`)

func parseVersion(s string) (semver, bool) {
	m := versionRe.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return semver{}, false
	}
	var v semver
	for i := range v.nums {
		n, err := strconv.Atoi(m[i+1])
		if err != nil {
			return semver{}, false
		}
		v.nums[i] = n
	}
	if m[4] != "" {
		v.pre = strings.Split(m[4], ".")
	}
	return v, true
}

// compareVersions orders two versions as semver does, numbers as numbers
// (alpha.10 after alpha.9) and a release after its pre-releases. A version
// that does not parse sorts first.
func compareVersions(a, b string) int {
	va, oka := parseVersion(a)
	vb, okb := parseVersion(b)
	switch {
	case !oka && !okb:
		return 0
	case !oka:
		return -1
	case !okb:
		return 1
	}
	for i := range va.nums {
		if c := cmpInt(va.nums[i], vb.nums[i]); c != 0 {
			return c
		}
	}
	switch {
	case len(va.pre) == 0 && len(vb.pre) == 0:
		return 0
	case len(va.pre) == 0:
		return 1
	case len(vb.pre) == 0:
		return -1
	}
	for i := 0; i < len(va.pre) && i < len(vb.pre); i++ {
		x, y := va.pre[i], vb.pre[i]
		nx, ex := strconv.Atoi(x)
		ny, ey := strconv.Atoi(y)
		var c int
		switch {
		case ex == nil && ey == nil:
			c = cmpInt(nx, ny)
		case ex == nil:
			c = -1 // numbers before words
		case ey == nil:
			c = 1
		default:
			c = strings.Compare(x, y)
		}
		if c != 0 {
			return c
		}
	}
	return cmpInt(len(va.pre), len(vb.pre))
}

func cmpInt(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

// release is the part of GitHub's release list the check reads.
type release struct {
	Tag        string `json:"tag_name"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
}

// newestRelease is the version install.sh would install: the highest
// stable release, or while there is none the highest pre-release (its
// newest()). Offering anything else would offer, again and again, a
// version the installer does not install. "" when none parses.
func newestRelease(rels []release) string {
	best, bestStable := "", ""
	for _, r := range rels {
		if r.Draft || !strings.HasPrefix(r.Tag, "v") {
			continue
		}
		v, ok := parseVersion(r.Tag)
		if !ok {
			continue
		}
		if best == "" || compareVersions(r.Tag, best) > 0 {
			best = r.Tag
		}
		if !r.Prerelease && len(v.pre) == 0 && (bestStable == "" || compareVersions(r.Tag, bestStable) > 0) {
			bestStable = r.Tag
		}
	}
	if bestStable != "" {
		return bestStable
	}
	return best
}

// releasesURL lists the project's releases (a variable for tests).
var releasesURL = "https://api.github.com/repos/wensuper/OmniLLM/releases?per_page=30"

// installerURL is the official installer, run as a person would run it.
var installerURL = "https://raw.githubusercontent.com/wensuper/OmniLLM/main/install.sh"

// updateClient fetches from GitHub: the hub's guarded transport (public
// addresses only, no proxy from the environment), bounded in time, and
// it sends nothing but the request itself.
func updateClient() *http.Client {
	return &http.Client{Transport: provider.GuardedTransport(), Timeout: 30 * time.Second}
}

func get(ctx context.Context, c *http.Client, url string, max int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "omnillm/"+version)
	if strings.Contains(url, "api.github.com") {
		req.Header.Set("Accept", "application/vnd.github+json")
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: HTTP %d", url, resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, max))
}

// latestRelease asks GitHub for the newest release's tag ("v0.2.0-alpha.9").
func latestRelease(ctx context.Context, c *http.Client) (string, error) {
	b, err := get(ctx, c, releasesURL, 4<<20)
	if err != nil {
		return "", err
	}
	var rels []release
	if err := json.Unmarshal(b, &rels); err != nil {
		return "", err
	}
	tag := newestRelease(rels)
	if tag == "" {
		return "", errors.New("no release")
	}
	return tag, nil
}

// runInstaller downloads the official installer into dir and runs it as a
// person would (sh install.sh ARGS), with its output in logw. It checks
// checksums and the signature itself and starts the hub (and the menu bar
// icon) again. detached: the installer runs in its own session, and under
// systemd in its own unit, so it outlives this process when it restarts
// the service this process belongs to (systemd would stop every process in
// the service, and the service's /tmp is private, hence dir).
func runInstaller(ctx context.Context, c *http.Client, dir string, args []string, logw io.Writer, detached bool) error {
	script, err := get(ctx, c, installerURL, 1<<20)
	if err != nil {
		return err
	}
	work := filepath.Join(dir, "update-installer")
	_ = os.RemoveAll(work)
	if err := os.MkdirAll(work, 0o700); err != nil {
		return err
	}
	p := filepath.Join(work, "install.sh")
	if err := os.WriteFile(p, script, 0o600); err != nil {
		return err
	}
	fmt.Fprintf(logw, "omnillm %s: update started %s\n", version, time.Now().Format(time.RFC3339))
	if detached && os.Getenv("INVOCATION_ID") != "" {
		if _, err := exec.LookPath("systemd-run"); err == nil {
			// A transient unit inherits nothing: hand over the settings
			// the installer writes into the service again.
			run := []string{"--user", "--collect", "--quiet", "--unit", "omnillm-update-" + strconv.FormatInt(time.Now().Unix(), 10),
				"--setenv=OMNILLM_NO_BROWSER=1"}
			for _, kv := range os.Environ() {
				if strings.HasPrefix(kv, "OMNILLM_") || strings.HasPrefix(kv, "LANG=") || strings.HasPrefix(kv, "LC_ALL=") {
					run = append(run, "--setenv="+kv)
				}
			}
			log := filepath.Join(dir, "update.log")
			run = append(run, "/bin/sh", "-c", `exec /bin/sh "$0" "$@" >>"`+log+`" 2>&1`, p)
			run = append(run, args...)
			out, err := exec.CommandContext(ctx, "systemd-run", run...).CombinedOutput()
			if err != nil {
				return fmt.Errorf("systemd-run: %v: %s", err, strings.TrimSpace(string(out)))
			}
			return nil
		}
	}
	defer os.RemoveAll(work)
	cmd := exec.CommandContext(ctx, "/bin/sh", append([]string{p}, args...)...)
	cmd.Dir = work
	cmd.Env = append(os.Environ(), "OMNILLM_NO_BROWSER=1")
	cmd.Stdout, cmd.Stderr = logw, logw
	if detached {
		detach(cmd)
	}
	return cmd.Run()
}

// updateState is what the menu knows about updates.
type updateState struct {
	Checking    bool      // a check is under way
	CheckFailed bool      // the last check failed
	CheckedAt   time.Time // when the last check found nothing newer
	Latest      string    // a newer release, if one was found
	Installing  string    // the version being installed now
	Failed      bool      // the last install failed
	Log         string    // the installer's log, for the failure line
}

// upToDateFor is how long "✓ 已是最新版本" stays before the line offers
// 检查更新… again.
const upToDateFor = time.Minute

// versionLine is the version line's title and whether a click installs.
func versionLine(cur string, s updateState) (title string, install bool) {
	switch {
	case s.Installing != "":
		return fmt.Sprintf(t("正在更新到 %s……", "Updating to %s…"), s.Installing), false
	case s.Failed && s.Latest != "":
		if s.Log != "" {
			return fmt.Sprintf(t("更新失败，点击重试（日志：%s）", "Update failed; click to retry (log: %s)"), tildePath(s.Log)), true
		}
		return t("更新失败，点击重试", "Update failed; click to retry"), true
	case s.Latest != "" && compareVersions(s.Latest, cur) > 0:
		// Installing restarts the hub, which ends replies in progress.
		return fmt.Sprintf(t("有新版本 %s · 点击更新（正在进行的回答会中断）", "New version %s · click to update (replies in progress will stop)"), s.Latest), true
	}
	return fmt.Sprintf(t("版本 %s", "Version %s"), cur), false
}

// checkLine is the line under it: shown only while the version line offers
// nothing (one way to update at a time); click says a click starts a check.
func checkLine(cur string, s updateState, now time.Time) (title string, click, show bool) {
	if _, install := versionLine(cur, s); install || s.Installing != "" {
		return "", false, false
	}
	switch {
	case s.Checking:
		return t("正在检查更新……", "Checking for updates…"), false, true
	case s.CheckFailed:
		return t("检查失败，点击重试", "Check failed; click to retry"), true, true
	case !s.CheckedAt.IsZero() && now.Sub(s.CheckedAt) < upToDateFor:
		return t("✓ 已是最新版本", "✓ Up to date"), false, true
	}
	return t("检查更新…", "Check for updates…"), true, true
}

// checked folds a finished check into the state.
func (s updateState) checked(cur, latest string, err error, now time.Time) updateState {
	s.Checking = false
	s.CheckFailed = err != nil
	if err != nil {
		return s
	}
	if compareVersions(latest, cur) > 0 {
		if latest != s.Latest {
			s.Failed = false
		}
		s.Latest = latest
		return s
	}
	s.Latest, s.CheckedAt = "", now
	return s
}

// tildePath shortens a path in the home folder to ~/… for a menu line.
func tildePath(p string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return p
	}
	if p == home {
		return "~"
	}
	if rel, ok := strings.CutPrefix(p, home+string(os.PathSeparator)); ok {
		return "~/" + rel
	}
	return p
}

// installedVersion asks the installed program for its version
// ("omnillm 0.2.0-alpha.9" → "0.2.0-alpha.9").
func installedVersion(ctx context.Context, bin string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, bin, "version").Output()
	if err != nil {
		return "", err
	}
	f := strings.Fields(string(out))
	if len(f) != 2 || f[0] != "omnillm" {
		return "", fmt.Errorf("unexpected version output %q", out)
	}
	return f[1], nil
}

// installed folds a finished install into the state, when this process is
// still running afterwards (the installer normally replaces it). have is
// the version the installed program reports. restart says the new version
// is there: the offer is done, and the menu should run the new program.
func (s updateState) installed(err error, have string) (next updateState, restart bool) {
	offered := s.Installing
	s.Installing = ""
	if err == nil && have != "" && compareVersions(have, offered) >= 0 {
		s.Latest, s.Failed = "", false
		return s, true
	}
	s.Failed = true
	return s, false
}
