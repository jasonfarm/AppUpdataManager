//go:build !windows

package winutil

// EnsureSingleInstance 在非 Windows 平台上始终返回 true，不做单实例限制。
func EnsureSingleInstance(mutexName, windowTitle string) bool {
	return true
}

// CenterWindow 在非 Windows 平台上为空实现。
func CenterWindow(title string) {}
