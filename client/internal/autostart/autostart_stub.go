//go:build !windows

package autostart

import "fmt"

// IsEnabled 在非 Windows 平台上始终返回 false，表示不支持开机自启动检测。
func IsEnabled(appName string) bool {
	return false
}

// Enable 在非 Windows 平台上返回错误，提示仅 Windows 支持开机自启动。
func Enable(appName, exePath string) error {
	return fmt.Errorf("auto start only supported on windows")
}

// Disable 在非 Windows 平台上返回错误，提示仅 Windows 支持取消开机自启动。
func Disable(appName string) error {
	return fmt.Errorf("auto start only supported on windows")
}
