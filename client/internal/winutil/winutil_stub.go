//go:build !windows

package winutil

// EnsureSingleInstance 在非 Windows 平台上始终返回 true，不做单实例限制。
func EnsureSingleInstance(mutexName, windowTitle string) bool {
	return true
}

// CenterWindow 在非 Windows 平台上为空实现。
func CenterWindow(title string) {}

// BringWindowToFront 在非 Windows 平台上为空实现。
func BringWindowToFront(title string) {}

// IsWindowVisible 在非 Windows 平台上始终返回 false。
func IsWindowVisible(title string) bool { return false }

// RegisterShowWindowCallback 在非 Windows 平台上为空实现。
func RegisterShowWindowCallback(callback func()) {}

// SubclassWindow 在非 Windows 平台上为空实现。
func SubclassWindow(title string) error { return nil }

// UnsubclassWindow 在非 Windows 平台上为空实现。
func UnsubclassWindow(title string) {}

// SendShowWindowMessage 在非 Windows 平台上始终返回 false。
func SendShowWindowMessage(title string) bool { return false }
