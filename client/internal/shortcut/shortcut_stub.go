//go:build !windows

package shortcut

import "fmt"

// DesktopDir 在非 Windows 平台返回错误。
func DesktopDir() (string, error) {
	return "", fmt.Errorf("desktop shortcuts are only supported on Windows")
}

// CreateShortcut 在非 Windows 平台返回错误。
func CreateShortcut(_, _ string) error {
	return fmt.Errorf("desktop shortcuts are only supported on Windows")
}

// ResolveShortcutTarget 在非 Windows 平台返回错误。
func ResolveShortcutTarget(_ string) (string, error) {
	return "", fmt.Errorf("desktop shortcuts are only supported on Windows")
}

// EnsureSoftwareShortcut 在非 Windows 平台为空操作。
func EnsureSoftwareShortcut(_, _ string) error {
	return nil
}

// RemoveOldVersionShortcuts 在非 Windows 平台为空操作。
func RemoveOldVersionShortcuts(_, _ string) error {
	return nil
}
