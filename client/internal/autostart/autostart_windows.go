//go:build windows

package autostart

import (
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

// Enable 将指定应用的可执行路径写入 Windows 注册表 Run 项，实现开机自启动。
func Enable(appName, exePath string) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, runKey, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.SetStringValue(appName, exePath)
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
