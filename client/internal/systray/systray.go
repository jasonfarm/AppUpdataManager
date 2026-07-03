package systray

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
)

// Setup 配置 Fyne 应用的系统托盘菜单和图标，包含显示主窗口与退出两项。
// 先设置菜单再设置图标，避免 Windows 上托盘图标空白或菜单不显示。
func Setup(app fyne.App, icon fyne.Resource, showCallback, exitCallback func()) {
	if desk, ok := app.(desktop.App); ok {
		menu := fyne.NewMenu("AppUpdateManager",
			fyne.NewMenuItem("显示主窗口", func() {
				if showCallback != nil {
					showCallback()
				}
			}),
			fyne.NewMenuItem("退出", func() {
				if exitCallback != nil {
					exitCallback()
				}
			}),
		)
		desk.SetSystemTrayMenu(menu)
		if icon != nil {
			desk.SetSystemTrayIcon(icon)
		}
	}
}
