package widgetx

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
)

// DesktopTray 添加托盘图标
func DesktopTray(a fyne.App, w fyne.Window, trayMenuItems ...*fyne.MenuItem) {
	if desk, ok := a.(desktop.App); ok {
		trayMenu := fyne.NewMenu("Tray Menu", trayMenuItems...)
		if len(trayMenuItems) == 0 {
			hideItem := fyne.NewMenuItem("Hide", func() {})
			hideItem.Icon = theme.VisibilityOffIcon()
			hideItem.Action = func() {
				if w.Content().Visible() {
					w.Hide()
					hideItem.Label = "Show"
					hideItem.Icon = theme.VisibilityIcon()
				} else {
					w.Show()
					hideItem.Label = "Hide"
					hideItem.Icon = theme.VisibilityOffIcon()
				}
				trayMenu.Refresh()
			}
			trayMenu.Items = []*fyne.MenuItem{hideItem}
		}
		desk.SetSystemTrayMenu(trayMenu)
	}
}
