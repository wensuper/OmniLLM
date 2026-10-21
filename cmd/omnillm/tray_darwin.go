//go:build darwin && cgo

package main

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"fyne.io/systray"

	"github.com/wensuper/OmniLLM/internal/control"
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
	// Until the owner picks a language (menu or console), the menu follows
	// the system's; the login item's OMNILLM_LANG is only the shell's at
	// install time.
	sysLang = trayLang()
	if st, ok := queryStatus(); ok && st.Lang != "" {
		os.Setenv("OMNILLM_LANG", st.Lang)
	} else if sysLang != "" {
		os.Setenv("OMNILLM_LANG", sysLang)
	}
	systray.Run(trayReady, func() {})
}

// sysLang is the system's language, for "Use the system language".
var sysLang string

// trayLang is the system's language for the menu: OMNILLM_TRAY_LANG when set (for
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
	fields := strings.FieldsFunc(string(out), func(r rune) bool { return strings.ContainsRune("(),\"\n\t ", r) })
	if len(fields) == 0 {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(fields[0]), "zh") {
		return "zh"
	}
	return "en"
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
	add := systray.AddMenuItem("", "")
	open := systray.AddMenuItem("", "")
	systray.AddSeparator()
	// The version line: grey while this is the latest; 有新版本 X when a
	// newer one is out, and one click installs it (trayversion.go).
	ver := systray.AddMenuItem("", "")
	ver.Disable()
	// 检查更新… under it while the version line offers nothing; a click
	// keeps the menu open so its result shows where it was clicked.
	check := systray.AddMenuItem("", "")
	check.SetKeepOpen(true)
	// Language, on its own at the bottom as macOS menus keep their
	// settings: the title names itself in both languages, so it is found
	// whichever is showing; the choices name themselves (中文, English) and
	// the one in use carries the check. The console has the same choice:
	// the hub keeps it, and each follows the other (show reads it).
	systray.AddSeparator()
	langMenu := systray.AddMenuItem("", "")
	langZh := langMenu.AddSubMenuItemCheckbox("中文", "", false)
	langEn := langMenu.AddSubMenuItemCheckbox("English", "", false)
	langSys := langMenu.AddSubMenuItemCheckbox("", "", false)
	chosen := ""          // the hub's choice: "zh", "en", or "" (the system's)
	var show func()       // set below; a language change redraws everything
	var showUpdate func() // likewise
	// label sets the titles that change only with the language.
	label := func() {
		start.SetTitle(t("启动 OmniLLM", "Start OmniLLM"))
		add.SetTitle(t("添加设备…", "Add device…"))
		open.SetTitle(t("打开控制台…", "Open the console…"))
		langMenu.SetTitle(t("语言 · Language", "Language · 语言"))
		langSys.SetTitle(t("跟随系统", "Use system language"))
		for _, c := range []struct {
			item *systray.MenuItem
			on   bool
		}{{langZh, chosen == "zh"}, {langEn, chosen == "en"}, {langSys, chosen == ""}} {
			if c.on {
				c.item.Check()
			} else {
				c.item.Uncheck()
			}
		}
	}
	// useLang shows the menu in the hub's choice (or the system's).
	useLang := func(l string) {
		chosen = l
		eff := l
		if eff == "" {
			eff = sysLang
		}
		if eff != "" {
			os.Setenv("OMNILLM_LANG", eff)
		}
		label()
		if show != nil {
			show()
			showUpdate()
		}
	}
	if st, ok := queryStatus(); ok {
		chosen = st.Lang
	}
	label()
	// pickLang keeps the choice in the hub (the console follows it) and
	// redraws at once.
	pickLang := func(l string) {
		sock := filepath.Join(dataDir("hub"), "control.sock")
		if ctl(sock, control.Request{Op: "ui_lang.set", Args: map[string]string{"lang": l}}, func(control.Reply) (bool, bool) { return false, false }) == nil {
			useLang(l)
		}
	}

	startFailed := false // launchctl kickstart failed: say what to do instead
	// Updates: checked a minute after start, then every day, and on a click.
	logPath := filepath.Join(dataDir("hub"), "update.log")
	// Held while anyone installs: this menu, the hub's own updater or
	// omnillm update (update.go).
	lockPath := filepath.Join(dataDir("hub"), "update.lock")
	up := updateState{Log: logPath}
	starting := false // a launchctl kickstart is under way
	show = func() {
		st, ok := queryStatus()
		if ok && st.Lang != chosen {
			// Chosen in the console: follow it.
			useLang(st.Lang)
			return
		}
		up.HubUpdating = up.Installing == "" && lockHeld(lockPath)
		updating := up.Installing != "" || up.HubUpdating
		head.SetTitle(statusLine(st, ok, updating, startFailed))
		if !ok {
			systray.SetTemplateIcon(trayDown, trayDown)
			devices.Hide()
			sources.Hide()
			add.Disable()
			open.Disable()
			// While an update installs, the hub is down on purpose and
			// comes back by itself: starting it would race the installer.
			if updating || starting {
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
	showUpdate = func() {
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
	startDone := make(chan error, 1)
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
			installDone <- update(ctx, dataDir("hub"), "hub", true)
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
					if lockHeld(lockPath) {
						// Someone else is installing already.
						up.HubUpdating = true
						showUpdate()
						show()
					} else {
						startInstall()
					}
				}
			case err := <-installDone:
				if errors.Is(err, errLocked) {
					// The hub (or a terminal) is installing it already:
					// not a failure, and that one replaces this process too.
					up.Installing, up.HubUpdating = "", true
					showUpdate()
					show()
					break
				}
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
			case <-langZh.ClickedCh:
				pickLang("zh")
			case <-langEn.ClickedCh:
				pickLang("en")
			case <-langSys.ClickedCh:
				pickLang("")
			case <-start.ClickedCh:
				if starting {
					break
				}
				starting = true
				start.Hide()
				go func() {
					err := exec.Command("launchctl", "kickstart", "gui/"+strconv.Itoa(os.Getuid())+"/"+launchdLabel).Run()
					if err == nil {
						time.Sleep(2 * time.Second) // the hub opens its control socket
					}
					startDone <- err
				}()
			case err := <-startDone:
				starting, startFailed = false, err != nil
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
