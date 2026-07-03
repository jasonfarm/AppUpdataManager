//go:build windows

package autostart

import (
	"fmt"

	"golang.org/x/sys/windows/registry"
)

const runKey = `SOFTWARE\Microsoft\Windows\CurrentVersion\Run`

// IsEnabled 查询 Windows 注册表，判断指定名称的应用是否已设置为开机自启动。
func IsEnabled(appName string) bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	_, _, err = k.GetStringValue(appName)
	return err == nil
}

// autostartFlag 是写入注册表启动项的参数，用于让程序识别自己是由开机自启动触发的。
const autostartFlag = "--autostart"

// Enable 将指定应用的可执行路径写入 Windows 注册表 Run 项，实现开机自启动。
// 写入的命令行会附加 --autostart 参数，以便程序启动时隐藏主窗口、只显示托盘图标。
func Enable(appName, exePath string) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, runKey, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.SetStringValue(appName, fmt.Sprintf(`"%s" %s`, exePath, autostartFlag))
}

// Disable 从 Windows 注册表 Run 项中删除指定应用的自启动项。
func Disable(appName string) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.DeleteValue(appName)
}
