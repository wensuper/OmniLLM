package main

// Updates (design §10.7). One way in for everyone: "omnillm update" on a
// hub or a device-only machine, the menu bar's line on a Mac, and the
// hub's own automatic update. All of them run the official installer
// (install.sh: it checks the checksum and, with cosign, the signature),
// so an update is exactly a fresh install of the newest release with the
// settings kept.
//
// The hub updates itself: it checks a few minutes after it starts and
// every 6 hours, and installs a newer release once nobody has used it for
// a while, so a reply is never cut off. A version that does not start is
// rolled back by "omnillm install" and skipped from then on.

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/wensuper/OmniLLM/hub"
	"golang.org/x/term"
)

const (
	autoFirstCheck = 3 * time.Minute
	autoEvery      = 6 * time.Hour
	autoRetry      = 5 * time.Minute // while busy, or after a failed check
	// autoQuiet: how long nobody must have used the hub before it
	// restarts itself into a new version.
	autoQuiet = 10 * time.Minute
	// hintEvery: how often a device-only machine looks for a new version.
	hintEvery = 24 * time.Hour
)

// updateFile is the update state kept next to the hub's (or the device's)
// data: what the last check found, whether automatic updates are on, and
// a version to pass over.
type updateFile struct {
	Latest    string    `json:"latest,omitempty"`
	CheckedAt time.Time `json:"checked_at,omitempty"`
	// Manual turns automatic updates off ("omnillm update --auto off").
	Manual bool `json:"manual,omitempty"`
	// Skip is a version that did not start and was rolled back.
	Skip string `json:"skip,omitempty"`
	// Error is the version the last automatic update failed to install,
	// for omnillm status; ErrorAt is when. The hub tries that version
	// again only a day later (a newer release is tried at once).
	Error   string    `json:"error,omitempty"`
	ErrorAt time.Time `json:"error_at,omitempty"`
}

// failed records that installing v did not work.
func (u *updateFile) failed(v string) { u.Error, u.ErrorAt = v, time.Now() }

func updatePath(dir string) string { return filepath.Join(dir, "update.json") }

func readUpdate(dir string) updateFile {
	var u updateFile
	if b, err := os.ReadFile(updatePath(dir)); err == nil {
		_ = json.Unmarshal(b, &u)
	}
	return u
}

func writeUpdate(dir string, u updateFile) {
	b, _ := json.MarshalIndent(u, "", "  ")
	tmp := updatePath(dir) + ".new"
	if os.WriteFile(tmp, b, 0o600) == nil {
		_ = os.Rename(tmp, updatePath(dir))
	}
}

// newer is the version to offer: the latest release when it is newer than
// cur and not the one being skipped.
func (u updateFile) newer(cur string) string {
	if u.Latest == "" || sameVersion(u.Latest, u.Skip) || compareVersions(u.Latest, cur) <= 0 {
		return ""
	}
	return u.Latest
}

// sameVersion: "v0.2.0" (a tag) and "0.2.0" (what a program says) are
// the same release.
func sameVersion(a, b string) bool {
	return a != "" && b != "" && strings.TrimPrefix(a, "v") == strings.TrimPrefix(b, "v")
}

// plain is a version as people see it ("0.2.0", not the tag "v0.2.0").
func plain(v string) string { return strings.TrimPrefix(v, "v") }

// devBuild: a build made by hand (no release version). It may check, but
// it never replaces itself with a release on its own.
func devBuild(v string) bool {
	_, ok := parseVersion(v)
	return !ok || strings.HasSuffix(v, "-dev")
}

// role says what this machine was set up as: "hub" (the background
// service, which also includes the device commands) or "device" (only the
// program, paired with a hub elsewhere). "" when neither is set up.
func role() string {
	if exists(filepath.Join(dataDir("hub"), "state.json")) || serviceInstalled() {
		return "hub"
	}
	// A device-only machine: set up by "install --device" (it leaves a
	// mark), or, from before the mark, a device folder next to the
	// installed program. A folder alone can come from running the
	// program from somewhere else, which sets up nothing.
	d := dataDir("device")
	if exists(filepath.Join(d, deviceMark)) {
		return "device"
	}
	if home, err := os.UserHomeDir(); err == nil && exists(d) && exists(filepath.Join(home, ".local", "bin", "omnillm")) {
		return "device"
	}
	return ""
}

// deviceMark is left in the device folder by "omnillm install --device".
const deviceMark = "installed-as-device"

func exists(p string) bool { _, err := os.Stat(p); return err == nil }

func serviceInstalled() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	switch runtime.GOOS {
	case "darwin":
		return exists(filepath.Join(home, "Library", "LaunchAgents", launchdLabel+".plist"))
	case "linux":
		return exists(filepath.Join(home, ".config", "systemd", "user", systemdUnit))
	}
	return false
}

// installerArgs: an update installs the same role again, without opening
// the console (the person did not ask to see it).
func installerArgs(r string) []string {
	if r == "device" {
		return []string{"--device"}
	}
	return []string{"--no-open"}
}

// updateLock lets one update run at a time (the menu bar, the hub and a
// person at a terminal could all start one). A lock older than the
// longest install is left over from a crash and taken over.
func updateLock(dir string) (release func(), ok bool) {
	p := filepath.Join(dir, "update.lock")
	// The lock appears with its holder already in it (written aside, then
	// linked into place), so nobody ever reads an empty one as stale.
	tmp := fmt.Sprintf("%s.%d", p, os.Getpid())
	if err := os.WriteFile(tmp, []byte(strconv.Itoa(os.Getpid())+"\n"), 0o600); err != nil {
		return nil, false
	}
	defer os.Remove(tmp)
	for i := 0; i < 2; i++ {
		if err := os.Link(tmp, p); err == nil {
			return func() { _ = os.Remove(p) }, true
		}
		st, serr := os.Stat(p)
		if serr != nil || (holderAlive(p) && time.Since(st.ModTime()) < 20*time.Minute) {
			return nil, false
		}
		_ = os.Remove(p)
	}
	return nil, false
}

// holderAlive: the process that took the lock still runs. A hub that
// updated itself is gone (the installer replaced it) and so is its lock.
func holderAlive(lock string) bool {
	b, err := os.ReadFile(lock)
	if err != nil {
		return false
	}
	if unit, ok := strings.CutPrefix(strings.TrimSpace(string(b)), "unit:"); ok {
		// Handed to an installer running in its own systemd unit.
		return exec.Command("systemctl", "--user", "is-active", "--quiet", unit).Run() == nil
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil || pid <= 0 {
		return false
	}
	p, err := os.FindProcess(pid)
	return err == nil && p.Signal(syscall.Signal(0)) == nil
}

// update installs the newest release over this one. log receives the
// installer's output; detached runs the installer in its own session (and
// on Linux, under systemd, in its own unit) so it outlives this process,
// which it replaces.
func update(ctx context.Context, dir, r string, logw io.Writer, detached bool) error {
	release, ok := updateLock(dir)
	if !ok {
		return say(t("另一个更新正在进行，请过几分钟再看。", "Another update is already running; check again in a few minutes."))
	}
	err := runInstaller(ctx, updateClient(), dir, installerArgs(r), logw, detached)
	if !errors.Is(err, errHandedOff) {
		release()
	}
	return err
}

// ---------------- omnillm update ----------------

func updateCmd(args []string) {
	fs := flag.NewFlagSet("update", flag.ExitOnError)
	check := fs.Bool("check", false, "only check for a new version")
	auto := fs.String("auto", "", "on|off: whether the hub updates itself")
	_ = fs.Parse(args)
	r := role()
	if r == "" {
		die(say(t("这台电脑上还没有安装 OmniLLM。安装方法见 https://github.com/wensuper/OmniLLM#quick-start",
			"OmniLLM isn’t installed on this computer yet. See https://github.com/wensuper/OmniLLM#quick-start")))
	}
	dir := dataDir(r)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		die(err)
	}
	u := readUpdate(dir)
	switch *auto {
	case "":
	case "on", "off":
		if r != "hub" {
			die(say(t("自动更新只用在主机上。这台机器上运行 omnillm update 就能更新。", "Automatic updates are for the hub. On this machine, run omnillm update to update.")))
		}
		u.Manual = *auto == "off"
		writeUpdate(dir, u)
		if u.Manual {
			fmt.Println(runnable(t("已关闭自动更新。有新版本时 omnillm status 会提示，运行 omnillm update 更新。", "Automatic updates are off. omnillm status says when a new version is out; run omnillm update to install it.")))
		} else {
			fmt.Println(t("已打开自动更新：有新版本时，OmniLLM 会在没人使用时自己更新。", "Automatic updates are on: OmniLLM installs a new version by itself while nobody is using it."))
		}
		return
	default:
		usage()
	}
	ctx, stop := signalCtx()
	defer stop()
	fmt.Printf(t("当前版本 %s，正在检查新版本……\n", "This is version %s. Checking for a new version…\n"), version)
	cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	latest, err := latestRelease(cctx, updateClient())
	cancel()
	if err != nil {
		die(say(t("连不上 GitHub，查不到新版本。请检查网络后再试。", "Can’t reach GitHub to look for a new version. Check the network and try again.")))
	}
	u.Latest, u.CheckedAt = latest, time.Now()
	writeUpdate(dir, u)
	if compareVersions(latest, version) <= 0 {
		fmt.Println(t("已经是最新版本。", "You’re up to date."))
		return
	}
	if *check {
		fmt.Println(runnable(fmt.Sprintf(t("有新版本 %s。运行 omnillm update 更新。", "Version %s is out. Run omnillm update to install it."), plain(latest))))
		return
	}
	fmt.Printf(t("正在更新到 %s：下载、核对、安装……\n", "Updating to %s: downloading, verifying, installing…\n"), plain(latest))
	if r == "hub" {
		fmt.Println(t("OmniLLM 会重启几秒，手机上正在出的回答会中断。", "OmniLLM restarts for a few seconds; a reply on its way to your phone is cut off."))
	}
	logPath := filepath.Join(dir, "update.log")
	logf, err := os.OpenFile(logPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		die(err)
	}
	ictx, icancel := context.WithTimeout(ctx, 15*time.Minute)
	err = update(ictx, dir, r, logf, false)
	icancel()
	logf.Close()
	if err != nil {
		fmt.Fprintln(os.Stderr, tail(logPath, 12))
		var s sayErr
		if errors.As(err, &s) {
			die(err)
		}
		die(sayf(t("更新没有完成，原来的版本照常在用。完整记录在 %s", "The update didn’t finish; the version you had keeps working. The full log is at %s"), logPath))
	}
	home, _ := os.UserHomeDir()
	now, _ := installedVersion(ctx, filepath.Join(home, ".local", "bin", "omnillm"))
	if now == "" || compareVersions(now, version) <= 0 {
		fmt.Fprintln(os.Stderr, tail(logPath, 12))
		die(sayf(t("更新没有生效，原来的版本照常在用。完整记录在 %s", "The update didn’t take; the version you had keeps working. The full log is at %s"), logPath))
	}
	u = readUpdate(dir)
	u.Error = ""
	writeUpdate(dir, u)
	if r == "hub" {
		fmt.Printf(t("已更新到 %s。OmniLLM 已重新启动，手机和其他设备会自己重新连上。\n", "Updated to %s. OmniLLM restarted; your phone and other devices reconnect by themselves.\n"), now)
		return
	}
	fmt.Printf(t("已更新到 %s。\n", "Updated to %s.\n"), now)
}

// tail is the last n lines of a log, indented, for an error report.
func tail(path string, n int) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return "  " + strings.Join(lines, "\n  ")
}

// ---------------- the hub updates itself ----------------

// autoAction is what the hub's updater does now, given the update state,
// how long nobody has used the hub, and whether it runs as a service that
// comes back by itself after the installer restarts it.
func autoAction(cur string, u updateFile, quiet time.Duration, service bool) string {
	switch {
	case devBuild(cur) || u.Manual || !service:
		return "wait"
	case u.newer(cur) == "":
		return "wait"
	case sameVersion(u.Error, u.Latest) && time.Since(u.ErrorAt) < 24*time.Hour:
		return "wait"
	case quiet < autoQuiet:
		return "busy"
	}
	return "install"
}

// underService: this hub was started by launchd or systemd (and so comes
// back after the installer restarts the service).
func underService() bool {
	if runtime.GOOS == "darwin" {
		return os.Getenv("XPC_SERVICE_NAME") == launchdLabel
	}
	return os.Getenv("INVOCATION_ID") != ""
}

// quietFor is how long ago the hub last answered a call (the audit log
// records each call when it ends); a call still under way, or a pairing,
// is Busy.
func quietFor(ctx context.Context, h *hub.Hub, started time.Time) time.Duration {
	last := started
	for back := 0; back < 2; back++ {
		es, err := h.Audit(ctx, h.Day(back))
		if err != nil {
			return 0
		}
		for _, e := range es {
			if e.Time.After(last) {
				last = e.Time
			}
		}
	}
	return time.Since(last)
}

// autoUpdate runs inside "omnillm hub run" for as long as the hub runs.
func autoUpdate(ctx context.Context, h *hub.Hub, dir string) {
	started := time.Now()
	wait := autoFirstCheck
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
		wait = autoEvery
		noteHandedOff(dir)
		u := readUpdate(dir)
		if time.Since(u.CheckedAt) > autoEvery-time.Minute {
			cctx, cancel := context.WithTimeout(ctx, time.Minute)
			latest, err := latestRelease(cctx, updateClient())
			cancel()
			if err != nil {
				wait = autoRetry
				continue
			}
			u = readUpdate(dir)
			u.Latest, u.CheckedAt = latest, time.Now()
			writeUpdate(dir, u)
		}
		quiet := quietFor(ctx, h, started)
		if h.Busy() {
			quiet = 0
		}
		switch autoAction(version, u, quiet, underService()) {
		case "busy":
			wait = autoRetry
		case "install":
			log.Printf("hub: updating %s → %s (automatic)", version, u.Latest)
			logf, err := os.OpenFile(filepath.Join(dir, "update.log"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
			if err != nil {
				wait = autoRetry
				continue
			}
			ictx, cancel := context.WithTimeout(ctx, 15*time.Minute)
			err = update(ictx, dir, "hub", logf, true)
			cancel()
			logf.Close()
			switch {
			case errors.Is(err, errHandedOff):
				// The installer runs in its own unit and restarts us; if
				// it fails, the next round reads its result.
				wait = autoRetry
			case err != nil:
				// Success restarts the service, which ends this process
				// before we get here; still running means it did not work.
				log.Printf("hub: automatic update to %s failed: %v", u.Latest, err)
				u = readUpdate(dir)
				u.failed(u.Latest)
				writeUpdate(dir, u)
			}
		}
	}
}

// noteHandedOff reads what an installer handed to its own systemd unit left
// behind (its exit status), once it has ended. Success restarted this hub
// into the new version, which clears any failure; a failure is recorded so
// omnillm status says so and the hub waits a day before trying it again.
func noteHandedOff(dir string) {
	p := filepath.Join(dir, "update.status")
	b, err := os.ReadFile(p)
	if err != nil {
		return
	}
	_ = os.Remove(p)
	if strings.TrimSpace(string(b)) == "0" {
		return
	}
	u := readUpdate(dir)
	if v := u.newer(version); v != "" {
		log.Printf("hub: automatic update to %s failed (exit %s); see update.log", v, strings.TrimSpace(string(b)))
		u.failed(v)
		writeUpdate(dir, u)
	}
}

// ---------------- device-only machines ----------------

// deviceUpdateHint tells a person at a terminal, once a day at most, that
// a new version is out. It reads what an earlier check found and starts
// a fresh check when that is old; scripts (no terminal) see nothing.
func deviceUpdateHint(dir string) {
	if !term.IsTerminal(int(os.Stderr.Fd())) || devBuild(version) {
		return
	}
	u := readUpdate(dir)
	if v := u.newer(version); v != "" {
		fmt.Fprintln(os.Stderr, runnable(fmt.Sprintf(t("（有新版本 %s：运行 omnillm update 更新。）", "(Version %s is out: run omnillm update to install it.)"), plain(v))))
	}
	if time.Since(u.CheckedAt) < hintEvery {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if latest, err := latestRelease(ctx, updateClient()); err == nil {
			u := readUpdate(dir)
			u.Latest, u.CheckedAt = latest, time.Now()
			writeUpdate(dir, u)
		}
	}()
}

// statusUpdateLine is omnillm status's line about updates, "" when there
// is nothing to say.
func statusUpdateLine(u updateFile) string {
	v := plain(u.newer(version))
	switch {
	case v == "":
		return ""
	case sameVersion(u.Error, v):
		return runnable(fmt.Sprintf(t("新版本 %s 自动更新没有成功，运行 omnillm update 再试一次。", "The automatic update to %s didn’t work; run omnillm update to try again."), v))
	case u.Manual || devBuild(version):
		return runnable(fmt.Sprintf(t("有新版本 %s。运行 omnillm update 更新。", "Version %s is out. Run omnillm update to install it."), v))
	}
	return runnable(fmt.Sprintf(t("有新版本 %s，会在没人使用时自动更新；想现在更新，运行 omnillm update。", "Version %s is out and installs by itself while nobody is using OmniLLM; to update now, run omnillm update."), v))
}
