//go:build windows

package winutil

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32                       = windows.NewLazySystemDLL("user32.dll")
	procFindWindowW              = user32.NewProc("FindWindowW")
	procSetForegroundWindow      = user32.NewProc("SetForegroundWindow")
	procIsIconic                 = user32.NewProc("IsIconic")
	procShowWindow               = user32.NewProc("ShowWindow")
	procGetWindowRect            = user32.NewProc("GetWindowRect")
	procSetWindowPos             = user32.NewProc("SetWindowPos")
	procGetSystemMetrics         = user32.NewProc("GetSystemMetrics")
)

const (
	// swRestore 用于恢复最小化的窗口。
	swRestore = 9
	// swpNosize 保持窗口大小不变。
	swpNosize = 0x0001
	// swpNozorder 保持窗口 Z 顺序不变。
	swpNozorder = 0x0004
	// smCXScreen 是屏幕宽度的系统度量索引。
	smCXScreen = 0
	// smCYScreen 是屏幕高度的系统度量索引。
	smCYScreen = 1
)

// EnsureSingleInstance 尝试创建命名互斥量。
// 如果互斥量已存在，说明已有实例在运行，此时将现有窗口置前并返回 false；
// 否则返回 true，表示当前实例可以继续运行。
func EnsureSingleInstance(mutexName, windowTitle string) bool {
	name, err := windows.UTF16PtrFromString(mutexName)
	if err != nil {
		return true
	}

	mutex, err := windows.CreateMutex(nil, false, name)
	if err != nil {
		if err == windows.ERROR_ALREADY_EXISTS {
			bringToFront(windowTitle)
			return false
		}
		// 其他错误允许继续运行，避免误拦截。
		return true
	}
	// 保持互斥量打开，进程退出时系统会自动释放。
	_ = mutex
	return true
}

// bringToFront 根据窗口标题找到已有窗口，并将其恢复到前台。
func bringToFront(title string) {
	hwnd := findWindow(title)
	if hwnd == 0 {
		return
	}
	if isIconic(hwnd) {
		showWindow(hwnd, swRestore)
	}
	setForegroundWindow(hwnd)
}

// CenterWindow 根据屏幕尺寸将指定标题的窗口居中显示。
func CenterWindow(title string) {
	hwnd := findWindow(title)
	if hwnd == 0 {
		return
	}

	var rect windows.Rect
	if err := getWindowRect(hwnd, &rect); err != nil {
		return
	}

	screenW := getSystemMetrics(smCXScreen)
	screenH := getSystemMetrics(smCYScreen)
	winW := rect.Right - rect.Left
	winH := rect.Bottom - rect.Top
	x := (screenW - winW) / 2
	y := (screenH - winH) / 2

	setWindowPos(hwnd, 0, x, y, 0, 0, swpNosize|swpNozorder)
}

// findWindow 根据窗口类名和标题查找窗口句柄，类名传空表示不匹配类名。
func findWindow(title string) windows.HWND {
	t, err := windows.UTF16PtrFromString(title)
	if err != nil {
		return 0
	}
	ret, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(t)))
	return windows.HWND(ret)
}

// setForegroundWindow 将指定窗口设置为前台窗口。
func setForegroundWindow(hwnd windows.HWND) {
	procSetForegroundWindow.Call(uintptr(hwnd))
}

// isIconic 判断窗口是否处于最小化状态。
func isIconic(hwnd windows.HWND) bool {
	ret, _, _ := procIsIconic.Call(uintptr(hwnd))
	return ret != 0
}

// showWindow 改变窗口显示状态。
func showWindow(hwnd windows.HWND, cmdShow int32) {
	procShowWindow.Call(uintptr(hwnd), uintptr(cmdShow))
}

// getWindowRect 获取窗口在屏幕坐标系下的矩形区域。
func getWindowRect(hwnd windows.HWND, rect *windows.Rect) error {
	ret, _, err := procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(rect)))
	if ret == 0 {
		return err
	}
	return nil
}

// getSystemMetrics 获取指定索引的系统度量值。
func getSystemMetrics(index int32) int32 {
	ret, _, _ := procGetSystemMetrics.Call(uintptr(index))
	return int32(ret)
}

// setWindowPos 设置窗口的位置、大小和 Z 顺序。
func setWindowPos(hwnd windows.HWND, hWndInsertAfter windows.HWND, x, y, cx, cy int32, flags uint32) {
	procSetWindowPos.Call(uintptr(hwnd), uintptr(hWndInsertAfter), uintptr(x), uintptr(y), uintptr(cx), uintptr(cy), uintptr(flags))
}
