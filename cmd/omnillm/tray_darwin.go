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
	systray.Run(trayReady, func() {})
}

func trayReady() {
	systray.SetTemplateIcon(trayDown, trayDown)
	systray.SetTooltip("OmniLLM · Your AI, Everywhere")
	head := systray.AddMenuItem("OmniLLM", "")
	head.Disable()
	relay := systray.AddMenuItem("", "")
	relay.Disable()
	counts := systray.AddMenuItem("", "")
	counts.Disable()
	// The version line, and 检查更新… under it while the version line
	// offers nothing (trayversion.go); the check line keeps the menu open
	// so its result shows where it was clicked.
	ver := systray.AddMenuItem("", "")
	ver.Disable()
	check := systray.AddMenuItem("", "")
	check.SetKeepOpen(true)
	systray.AddSeparator()
	open := systray.AddMenuItem(t("打开控制台", "Open the console"), "")
	add := systray.AddMenuItem(t("添加设备…", "Add device…"), "")
	start := systray.AddMenuItem(t("启动 OmniLLM", "Start OmniLLM"), "")
	start.Hide()
	systray.AddSeparator()
	quit := systray.AddMenuItem(t("隐藏菜单栏图标（下次登录时恢复）", "Hide the menu bar icon (back at next login)"), "")

	startFailed := false // launchctl kickstart failed: say what to do instead
	// Updates: checked a minute after start, then every day, and on a click.
	logPath := filepath.Join(dataDir("hub"), "update.log")
	up := updateState{Log: logPath}
	show := func() {
		st, ok := queryStatus()
		if !ok {
			systray.SetTemplateIcon(trayDown, trayDown)
			head.SetTitle(t("OmniLLM 没有运行", "OmniLLM isn’t running"))
			relay.SetTitle(t("设备暂时连不上", "Devices can’t connect now"))
			if startFailed {
				relay.SetTitle(t("没能启动。在终端运行 omnillm install。", "Couldn’t start it. Run omnillm install in Terminal."))
			}
			counts.Hide()
			open.Disable()
			add.Disable()
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
		open.Enable()
		add.Enable()
		counts.Show()
		head.SetTitle(t("OmniLLM 在运行", "OmniLLM is running")) // the version has its own line
		switch {
		case !st.RelayConfigured:
			systray.SetTemplateIcon(trayUp, trayUp)
			relay.SetTitle(t("只在这台 Mac 上用（没有设置远程连接）", "This Mac only (remote access not set up)"))
		case st.RelayUp:
			systray.SetTemplateIcon(trayUp, trayUp)
			relay.SetTitle(t("● 手机和其他设备可以连上", "● Phones and other devices can connect"))
		default:
			systray.SetTemplateIcon(trayDown, trayDown)
			relay.SetTitle(t("○ 远程连接断开，设备暂时连不上", "○ Remote access is down; devices can’t connect now"))
		}
		counts.SetTitle(fmt.Sprintf(t("设备 %d 台 · 模型来源 %d 个", "Devices: %d · Model sources: %d"), st.Devices, st.Providers))
	}
	show()

	checkClick, installClick := false, false
	showUpdate := func() {
		title, inst := versionLine(version, up)
		ver.SetTitle(title)
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
			installDone <- runInstaller(ctx, updateClient(), logPath)
		}()
	}
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
				up = up.checked(version, r.latest, r.err, time.Now())
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
				var restart bool
				up, restart = up.installed(err, have)
				showUpdate()
				if restart {
					// The lock file is close-on-exec: the new menu takes it.
					if e := syscall.Exec(bin, []string{bin, "tray"}, os.Environ()); e != nil {
						fmt.Fprintln(os.Stderr, "omnillm tray: restart:", e)
					}
				}
				show()
			case <-open.ClickedCh:
				trayOpen("")
			case <-add.ClickedCh:
				trayOpen("&add")
			case <-start.ClickedCh:
				err := exec.Command("launchctl", "kickstart", "gui/"+strconv.Itoa(os.Getuid())+"/"+launchdLabel).Run()
				startFailed = err != nil
				time.Sleep(2 * time.Second)
				show()
			case <-quit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

// trayOpen opens the console with a fresh one-time login; extra is added
// to the link's fragment (&add opens "add device").
func trayOpen(extra string) {
	if url, err := loginURL(); err == nil {
		_ = openLogin(url + extra)
	}
}
