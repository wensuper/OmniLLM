//go:build darwin && cgo

package main

import (
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
	systray.AddSeparator()
	open := systray.AddMenuItem(t("打开控制台", "Open the console"), "")
	add := systray.AddMenuItem(t("添加手机或电脑…", "Add a phone or computer…"), "")
	start := systray.AddMenuItem(t("启动 OmniLLM", "Start OmniLLM"), "")
	start.Hide()
	systray.AddSeparator()
	quit := systray.AddMenuItem(t("隐藏菜单栏图标（下次登录时恢复）", "Hide the menu bar icon (back at next login)"), "")

	show := func() {
		st, ok := queryStatus()
		if !ok {
			systray.SetTemplateIcon(trayDown, trayDown)
			head.SetTitle(t("OmniLLM 没有运行", "OmniLLM isn't running"))
			relay.SetTitle(t("手机和其他设备现在用不了", "Phones and other devices can't use it now"))
			counts.Hide()
			open.Disable()
			add.Disable()
			start.Show()
			return
		}
		start.Hide()
		open.Enable()
		add.Enable()
		counts.Show()
		head.SetTitle(fmt.Sprintf(t("OmniLLM 在运行（%s）", "OmniLLM is running (%s)"), st.Version))
		switch {
		case !st.RelayConfigured:
			systray.SetTemplateIcon(trayUp, trayUp)
			relay.SetTitle(t("只在这台 Mac 上用（没有设置远程连接）", "This Mac only (remote access not set up)"))
		case st.RelayUp:
			systray.SetTemplateIcon(trayUp, trayUp)
			relay.SetTitle(t("● 手机和其他设备可以连上", "● Phones and other devices can connect"))
		default:
			systray.SetTemplateIcon(trayDown, trayDown)
			relay.SetTitle(t("○ 远程连接断开，其他设备暂时连不上", "○ Remote access is down; other devices can't connect now"))
		}
		counts.SetTitle(fmt.Sprintf(t("设备 %d 台 · 模型来源 %d 个", "Devices: %d · Model sources: %d"), st.Devices, st.Providers))
	}
	show()
	go func() {
		tick := time.NewTicker(5 * time.Second)
		defer tick.Stop()
		for {
			select {
			case <-tick.C:
				show()
			case <-open.ClickedCh:
				trayOpen("")
			case <-add.ClickedCh:
				trayOpen("&add")
			case <-start.ClickedCh:
				_ = exec.Command("launchctl", "kickstart", "gui/"+strconv.Itoa(os.Getuid())+"/"+launchdLabel).Run()
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
