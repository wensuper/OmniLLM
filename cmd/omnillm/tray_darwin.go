//go:build darwin && cgo

package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"fyne.io/systray"
)

// The menu bar mark as macOS template images (black + alpha; the system
// tints them): the orbit and core of the logo. up: phones can reach the
// hub; down (dashed orbit, hollow core): they can't right now.
var (
	//go:embed tray/up.png
	trayUp []byte
	//go:embed tray/down.png
	trayDown []byte
)

// trayCmd shows OmniLLM in the menu bar: whether phones can reach the hub,
// and one click to the console or to adding a device. It only asks the hub
// over its control socket, so either can restart without the other.
func trayCmd() {
	lock, err := os.OpenFile(filepath.Join(dataDir("hub"), "tray.lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err == nil && syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB) != nil {
		fmt.Fprintln(os.Stderr, t("OmniLLM 已经在菜单栏里了。", "OmniLLM is already in the menu bar."))
		return
	}
	// The menu follows the system language, as NEXA's does; the login
	// item's OMNILLM_LANG is only the shell's at install time.
	if l := trayLang(); l != "" {
		os.Setenv("OMNILLM_LANG", l)
	}
	systray.Run(trayReady, func() {})
}

// trayLang is the language the menu shows: OMNILLM_TRAY_LANG when set (for
// tests), else the first of macOS's preferred languages ("" when unknown).
func trayLang() string {
	if v := os.Getenv("OMNILLM_TRAY_LANG"); v != "" {
		return v
	}
	out, err := exec.Command("defaults", "read", "-g", "AppleLanguages").Output()
	if err != nil {
		return ""
	}
	// ( "en-SG", "zh-Hans-SG" ): the first entry wins.
	for _, f := range strings.FieldsFunc(string(out), func(r rune) bool { return strings.ContainsRune("(),\"\n\t ", r) }) {
		if strings.HasPrefix(strings.ToLower(f), "zh") {
			return "zh"
		}
		return "en"
	}
	return ""
}

// The menu, laid out as NEXA's: one status line; the counts, each opening
// its console page, with the actions under them; the version and
// 检查更新…. There is no quit item: the icon stays in the menu bar (zc
// 2026-10-21).
func trayReady() {
	systray.SetTemplateIcon(trayDown, trayDown)
	systray.SetTooltip("OmniLLM · Your AI, Everywhere")
	head := systray.AddMenuItem("", "")
	head.Disable()
	// Shown only while the hub does not answer.
	start := systray.AddMenuItem(t("启动 OmniLLM", "Start OmniLLM"), "")
	start.Hide()
	systray.AddSeparator()
	devices := systray.AddMenuItem("", "")
	sources := systray.AddMenuItem("", "")
	// No separator here: while the hub is down the counts are hidden, and
	// two separators would stand together.
	add := systray.AddMenuItem(t("添加设备…", "Add a device…"), "")
	open := systray.AddMenuItem(t("打开控制台…", "Open the console…"), "")
	systray.AddSeparator()
	// The version line: grey while this is the latest; 有新版本 X when a
	// newer one is out, and one click installs it (trayversion.go).
	ver := systray.AddMenuItem("", "")
	ver.Disable()
	// 检查更新… under it while the version line offers nothing; a click
	// keeps the menu open so its result shows where it was clicked.
	check := systray.AddMenuItem("", "")
	check.SetKeepOpen(true)

	startFailed := false // launchctl kickstart failed: say what to do instead
	// Updates: checked a minute after start, then every day, and on a click.
	logPath := filepath.Join(dataDir("hub"), "update.log")
	up := updateState{Log: logPath}
	show := func() {
		st, ok := queryStatus()
		head.SetTitle(statusLine(st, ok, up.Installing != "", startFailed))
		if !ok {
			systray.SetTemplateIcon(trayDown, trayDown)
			devices.Hide()
			sources.Hide()
			add.Disable()
			open.Disable()
			// While an update installs, the hub is down on purpose and
			// comes back by itself.
			if up.Installing != "" {
				start.Hide()
			} else {
				start.Show()
			}
			return
		}
		startFailed = false
		start.Hide()
		add.Enable()
		open.Enable()
		devices.SetTitle(devicesLine(st))
		sources.SetTitle(sourcesLine(st))
		devices.Show()
		sources.Show()
		if !st.RelayConfigured || st.RelayUp {
			systray.SetTemplateIcon(trayUp, trayUp)
		} else {
			systray.SetTemplateIcon(trayDown, trayDown)
		}
	}
	show()

	checkClick, installClick := false, false
	showUpdate := func() {
		title, inst := versionLine(version, up)
		ver.SetTitle(title)
		ver.SetTooltip(versionTip(version, up))
		installClick = inst
		if inst {
			ver.Enable()
		} else {
			ver.Disable()
		}
		ct, click, vis := checkLine(version, up, time.Now())
		checkClick = click
		if !vis {
			check.Hide()
			return
		}
		check.SetTitle(ct)
		if click {
			check.Enable()
		} else {
			check.Disable()
		}
		check.Show()
	}
	type checkResult struct {
		latest string
		err    error
	}
	checkDone := make(chan checkResult, 1)
	installDone := make(chan error, 1)
	startCheck := func() {
		up.Checking = true
		showUpdate()
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			v, err := latestRelease(ctx, updateClient())
			checkDone <- checkResult{v, err}
		}()
	}
	startInstall := func() {
		up.Installing, up.Failed = up.Latest, false
		showUpdate()
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
			defer cancel()
			logf, err := os.OpenFile(logPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
			if err != nil {
				installDone <- err
				return
			}
			defer logf.Close()
			installDone <- update(ctx, dataDir("hub"), "hub", logf, true)
		}()
	}
	up = up.fromHub(version, readUpdate(dataDir("hub")))
	showUpdate()
	firstCheck := time.NewTimer(time.Minute)
	daily := time.NewTicker(24 * time.Hour)
	go func() {
		tick := time.NewTicker(5 * time.Second)
		defer tick.Stop()
		for {
			select {
			case <-tick.C:
				show()
				showUpdate() // "✓ 已是最新版本" goes back after a minute
			case <-firstCheck.C:
				if !up.Checking && up.Installing == "" {
					startCheck()
				}
			case <-daily.C:
				if !up.Checking && up.Installing == "" {
					startCheck()
				}
			case r := <-checkDone:
				up = up.checked(version, r.latest, r.err, time.Now()).fromHub(version, readUpdate(dataDir("hub")))
				showUpdate()
			case <-check.ClickedCh:
				if checkClick && !up.Checking {
					startCheck()
				}
			case <-ver.ClickedCh:
				if installClick && up.Installing == "" {
					startInstall()
				}
			case err := <-installDone:
				// Success normally replaces this process (the installer
				// starts the menu bar icon again). Still here: if the
				// new version is installed all the same, run it.
				home, _ := os.UserHomeDir()
				bin := filepath.Join(home, ".local", "bin", "omnillm")
				have := ""
				if err == nil {
					have, _ = installedVersion(context.Background(), bin)
				}
				offered := up.Installing
				var restart bool
				up, restart = up.installed(err, have)
				showUpdate()
				if restart {
					// The lock file is close-on-exec: the new menu takes it.
					// Exec returns only when it failed: then the update
					// did not take here, and the menu says so.
					e := syscall.Exec(bin, []string{bin, "tray"}, os.Environ())
					fmt.Fprintln(os.Stderr, "omnillm tray: restart:", e)
					noteLog(logPath, "restart into the new version failed: %v", e)
					up = up.restartFailed(offered)
					showUpdate()
				}
				show()
			case <-open.ClickedCh:
				trayOpen("")
			case <-devices.ClickedCh:
				trayOpen("&tab=devices")
			case <-sources.ClickedCh:
				trayOpen("&tab=models")
			case <-add.ClickedCh:
				trayOpen("&add")
			case <-start.ClickedCh:
				err := exec.Command("launchctl", "kickstart", "gui/"+strconv.Itoa(os.Getuid())+"/"+launchdLabel).Run()
				startFailed = err != nil
				time.Sleep(2 * time.Second)
				show()
			}
		}
	}()
}

// trayOpen opens the console with a fresh one-time login; extra is added
// to the link's fragment (&add opens "add device", &tab=models a tab).
func trayOpen(extra string) {
	if url, err := loginURL(); err == nil {
		_ = openLogin(url + extra)
	}
}
