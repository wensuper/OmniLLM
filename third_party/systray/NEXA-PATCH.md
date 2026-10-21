# NEXA patch of fyne.io/systray v1.12.2

Copied from fyne.io/systray v1.12.2 (Apache-2.0, see LICENSE) and used (as in NEXA, which carries the same patch) through
`replace fyne.io/systray => ./third_party/systray` in OmniLLM's go.mod. The
example directory is left out.

What is changed (zc 2026-10-11: clicking 检查更新… closed the menu, so the
result shown in that line could only be seen by opening the menu again):

- `MenuItem.SetKeepOpen(bool)`: on macOS the item is drawn by `KeepOpenView`
  (systray_darwin.m) and a click on it runs `ClickedCh` without closing the
  menu; its title and enabled state still follow SetTitle / Enable / Disable,
  live while the menu is open. Other platforms ignore it.
- `add_or_update_menu_item` takes a `keepOpen` argument (systray.h,
  systray_darwin.go, systray_darwin.m).

To take a newer upstream, copy it over this directory and re-apply these
changes; `git diff` of this directory against the upstream tag shows them.
