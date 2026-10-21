//go:build !(darwin && cgo)

package main

import "fmt"

func trayCmd() {
	fmt.Println(t("菜单栏图标只在 macOS 上有。", "The menu bar icon is only on macOS."))
}
