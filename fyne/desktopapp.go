package fynex

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	themex "github.com/xyzj/gopsu/fyne/theme"
)

// DesktopApp DesktopApp
type DesktopApp struct {
	a fyne.App
	w fyne.Window
}

// DesktopAppOptions DesktopAppOptions
type DesktopAppOptions struct {
	AppID         string
	WindowTitle   string
	Icon          *fyne.StaticResource
	Size          fyne.Size
	EnableTray    bool
	TrayMenuItems []*fyne.MenuItem
}

// NewDesktopApp 创建一个新的桌面app,需要在main()函数的最开始就调用，否则，创建页面控件的时候会报错
func NewDesktopApp(opt *DesktopAppOptions) *DesktopApp {
	a := app.NewWithID(opt.AppID)
	a.Settings().SetTheme(&themex.ZhHans{})
	if opt.Icon != nil {
		a.SetIcon(opt.Icon)
	}
	w := a.NewWindow(opt.WindowTitle)
	if opt.EnableTray {
		trayMenu := fyne.NewMenu("Tray Menu")
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
		w.SetCloseIntercept(func() {
			w.Hide()
			hideItem.Label = "Show"
			hideItem.Icon = theme.VisibilityIcon()
			trayMenu.Refresh()
		})
		// 托盘图标
		if len(opt.TrayMenuItems) == 0 {
			trayMenu.Items = []*fyne.MenuItem{hideItem}
		} else {
			trayMenu.Items = opt.TrayMenuItems
		}
		if desk, ok := a.(desktop.App); ok {
			desk.SetSystemTrayMenu(trayMenu)
		}
	}

	w.Resize(opt.Size)
	w.CenterOnScreen()
	return &DesktopApp{
		a: a,
		w: w,
	}
}

// MainWindow MainWindow
func (d *DesktopApp) MainWindow() fyne.Window {
	return d.w
}

// MainApp MainApp
func (d *DesktopApp) MainApp() fyne.App {
	return d.a
}

// ShowAndRun ShowAndRun
func (d *DesktopApp) ShowAndRun(obj fyne.CanvasObject) {
	d.w.SetContent(obj)
	d.w.ShowAndRun()
}
