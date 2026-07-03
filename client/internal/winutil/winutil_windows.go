//go:build windows

package winutil

import (
	"errors"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32                       = windows.NewLazySystemDLL("user32.dll")
	procFindWindowW              = user32.NewProc("FindWindowW")
	procSetForegroundWindow      = user32.NewProc("SetForegroundWindow")
	procIsIconic                 = user32.NewProc("IsIconic")
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")
	procShowWindow               = user32.NewProc("ShowWindow")
	procGetWindowRect            = user32.NewProc("GetWindowRect")
	procSetWindowPos             = user32.NewProc("SetWindowPos")
	procGetSystemMetrics         = user32.NewProc("GetSystemMetrics")
	procGetForegroundWindow      = user32.NewProc("GetForegroundWindow")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procAttachThreadInput        = user32.NewProc("AttachThreadInput")
	procGetCurrentThreadId       = user32.NewProc("GetCurrentThreadId")
	procCallWindowProcW          = user32.NewProc("CallWindowProcW")
	procSetWindowLongPtrW        = user32.NewProc("SetWindowLongPtrW")
	procRegisterWindowMessageW   = user32.NewProc("RegisterWindowMessageW")
	procSendMessageTimeoutW      = user32.NewProc("SendMessageTimeoutW")
	procAllowSetForegroundWindow = user32.NewProc("AllowSetForegroundWindow")
)

const (
	// swShow 用于显示隐藏的窗口。
	swShow = 5
	// swRestore 用于恢复最小化的窗口。
	swRestore = 9
	// swpNosize 保持窗口大小不变。
	swpNosize = 0x0001
	// swpNomove 保持窗口位置不变。
	swpNomove = 0x0002
	// swpNozorder 保持窗口 Z 顺序不变。
	swpNozorder = 0x0004
	// swpShowwindow 显示窗口（配合 SetWindowPos 使用）。
	swpShowwindow = 0x0040
	// smCXScreen 是屏幕宽度的系统度量索引。
	smCXScreen = 0
	// smCYScreen 是屏幕高度的系统度量索引。
	smCYScreen = 1
)

const (
	// asfwAny 允许任何进程设置前台窗口。
	asfwAny = 0xFFFFFFFF
	// smtoAbortIfHung 在目标线程挂起时中止发送。
	smtoAbortIfHung = 0x0002
	// smtoBlock 阻塞等待消息处理完成。
	smtoBlock = 0x0001
)

var (
	// gwlWndProc 是窗口过程索引。
	gwlWndProc = int32(-4)
)

const (
	// hwndTopmost 将窗口置于所有非顶层窗口之上。
	hwndTopmost uintptr = ^uintptr(0)
	// hwndNoTopmost 将窗口置于所有顶层窗口之后，即取消置顶。
	hwndNoTopmost uintptr = ^uintptr(0) - 1
)

const showWindowMessageName = "AppUpdateManagerClient_ShowWindow"

var (
	showWindowMsg     uint32
	showWindowMsgOnce sync.Once

	showWindowCallback func()
	originalWndProc    uintptr
	wndProcCallback    uintptr
)

// RegisterShowWindowCallback 注册收到显示窗口消息时的回调。
// 该回调会在窗口所属的线程（即 Fyne 主消息线程）中被调用。
func RegisterShowWindowCallback(callback func()) {
	showWindowCallback = callback
}

// ensureShowWindowMessage 注册自定义窗口消息，保证只执行一次。
func ensureShowWindowMessage() uint32 {
	showWindowMsgOnce.Do(func() {
		name, err := windows.UTF16PtrFromString(showWindowMessageName)
		if err != nil {
			return
		}
		ret, _, _ := procRegisterWindowMessageW.Call(uintptr(unsafe.Pointer(name)))
		showWindowMsg = uint32(ret)
	})
	return showWindowMsg
}

// SubclassWindow 查找指定标题的窗口并子类化其 WNDPROC，用于接收自定义显示消息。
func SubclassWindow(title string) error {
	msg := ensureShowWindowMessage()
	if msg == 0 {
		return errors.New("register window message failed")
	}

	hwnd := findWindow(title)
	if hwnd == 0 {
		return errors.New("window not found")
	}

	if originalWndProc != 0 {
		return nil // 已经子类化过
	}

	if wndProcCallback == 0 {
		wndProcCallback = syscall.NewCallback(wndProc)
	}

	ret, _, err := procSetWindowLongPtrW.Call(uintptr(hwnd), uintptr(gwlWndProc), wndProcCallback)
	if ret == 0 {
		return err
	}
	originalWndProc = ret
	return nil
}

// UnsubclassWindow 恢复指定标题窗口的原始 WNDPROC。
func UnsubclassWindow(title string) {
	if originalWndProc == 0 {
		return
	}
	hwnd := findWindow(title)
	if hwnd == 0 {
		return
	}
	_, _, _ = procSetWindowLongPtrW.Call(uintptr(hwnd), uintptr(gwlWndProc), originalWndProc)
	originalWndProc = 0
}

// wndProc 是自定义窗口过程，处理显示窗口消息并将其他消息转发给原过程。
func wndProc(hwnd windows.HWND, msg uint32, wParam uintptr, lParam uintptr) uintptr {
	if msg == showWindowMsg {
		if showWindowCallback != nil {
			showWindowCallback()
		}
		// 在主消息线程中直接激活并置前，避免跨 goroutine 竞争。
		setForegroundWindow(hwnd)
		setWindowPos(hwnd, windows.HWND(hwndTopmost), 0, 0, 0, 0, swpNomove|swpNosize)
		setWindowPos(hwnd, windows.HWND(hwndNoTopmost), 0, 0, 0, 0, swpNomove|swpNosize)
		return 1
	}
	ret, _, _ := procCallWindowProcW.Call(originalWndProc, uintptr(hwnd), uintptr(msg), wParam, lParam)
	return ret
}

// SendShowWindowMessage 向指定标题的窗口发送显示消息。
// 返回 true 表示消息已被对方的自定义 WNDPROC 处理。
func SendShowWindowMessage(title string) bool {
	msg := ensureShowWindowMessage()
	if msg == 0 {
		return false
	}

	// 允许接收方调用 SetForegroundWindow 激活窗口。
	procAllowSetForegroundWindow.Call(uintptr(asfwAny))

	hwnd := findWindow(title)
	if hwnd == 0 {
		return false
	}

	var result uintptr
	ret, _, _ := procSendMessageTimeoutW.Call(
		uintptr(hwnd),
		uintptr(msg),
		0,
		0,
		uintptr(smtoBlock|smtoAbortIfHung),
		1000,
		uintptr(unsafe.Pointer(&result)),
	)
	return ret != 0 && result == 1
}

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
			// 先尝试通过窗口消息通知已有实例显示并激活窗口，
			// 给予对方一定时间完成 WNDPROC 子类化。
			shown := false
			for i := 0; i < 20; i++ {
				if SendShowWindowMessage(windowTitle) {
					shown = true
					break
				}
				time.Sleep(50 * time.Millisecond)
			}
			if !shown {
				BringWindowToFront(windowTitle)
			}
			return false
		}
		// 其他错误允许继续运行，避免误拦截。
		return true
	}
	// 保持互斥量打开，进程退出时系统会自动释放。
	_ = mutex
	return true
}

// BringWindowToFront 根据窗口标题找到已有窗口，并将其恢复到前台。
// 窗口处于最小化、隐藏或系统托盘状态时都会被显示并激活。
func BringWindowToFront(title string) {
	hwnd := findWindow(title)
	if hwnd == 0 {
		return
	}

	// 最小化则恢复，隐藏则显示。
	if isIconic(hwnd) {
		showWindow(hwnd, swRestore)
	} else if !isWindowVisible(hwnd) {
		showWindow(hwnd, swShow)
	}

	// 临时置顶再取消置顶，可将窗口强制带到最前面。
	setWindowPos(hwnd, windows.HWND(hwndTopmost), 0, 0, 0, 0, swpNomove|swpNosize)
	setWindowPos(hwnd, windows.HWND(hwndNoTopmost), 0, 0, 0, 0, swpNomove|swpNosize)

	// 通过附加前台线程输入，绕过 SetForegroundWindow 的进程限制。
	attachToForegroundThread(hwnd)
	setForegroundWindow(hwnd)
}

// attachToForegroundThread 将当前线程附加到当前前台窗口的输入线程，
// 使得当前进程有资格调用 SetForegroundWindow。
func attachToForegroundThread(hwnd windows.HWND) {
	foregroundHwnd := getForegroundWindow()
	if foregroundHwnd == 0 {
		return
	}

	var pid uint32
	foregroundThread := getWindowThreadProcessId(foregroundHwnd, &pid)
	currentThread := getCurrentThreadId()
	if foregroundThread == currentThread {
		return
	}

	attachThreadInput(foregroundThread, currentThread, true)
	defer attachThreadInput(foregroundThread, currentThread, false)
	_ = pid
}

// IsWindowVisible 根据窗口标题判断窗口是否可见。
func IsWindowVisible(title string) bool {
	hwnd := findWindow(title)
	return hwnd != 0 && isWindowVisible(hwnd)
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

// isWindowVisible 判断窗口是否可见。
func isWindowVisible(hwnd windows.HWND) bool {
	ret, _, _ := procIsWindowVisible.Call(uintptr(hwnd))
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

// getForegroundWindow 获取当前前台窗口句柄。
func getForegroundWindow() windows.HWND {
	ret, _, _ := procGetForegroundWindow.Call()
	return windows.HWND(ret)
}

// getWindowThreadProcessId 获取窗口所属线程与进程 ID。
func getWindowThreadProcessId(hwnd windows.HWND, pid *uint32) uint32 {
	ret, _, _ := procGetWindowThreadProcessId.Call(uintptr(hwnd), uintptr(unsafe.Pointer(pid)))
	return uint32(ret)
}

// attachThreadInput 附加或分离两个线程的输入处理。
func attachThreadInput(idAttach, idAttachTo uint32, attach bool) {
	var attachFlag uintptr
	if attach {
		attachFlag = 1
	}
	procAttachThreadInput.Call(uintptr(idAttach), uintptr(idAttachTo), attachFlag)
}

// getCurrentThreadId 获取当前线程 ID。
func getCurrentThreadId() uint32 {
	ret, _, _ := procGetCurrentThreadId.Call()
	return uint32(ret)
}
