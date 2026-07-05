//go:build windows

package shortcut

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"golang.org/x/sys/windows"
)

// FOLDERID_Desktop GUID: B4BFCC3A-DB2C-424C-B029-7FE99A87C641
var folderIDDesktop = windows.KNOWNFOLDERID{0xB4BFCC3A, 0xDB2C, 0x424C, [8]byte{0xB0, 0x29, 0x7F, 0xE9, 0x9A, 0x87, 0xC6, 0x41}}

var (
	shell32                    = syscall.NewLazyDLL("shell32.dll")
	procSHGetKnownFolderPath   = shell32.NewProc("SHGetKnownFolderPath")
	procCoTaskMemFree          = syscall.NewLazyDLL("ole32.dll").NewProc("CoTaskMemFree")
)

// DesktopDir 返回当前用户的桌面目录路径。
func DesktopDir() (string, error) {
	var pathPtr *uint16
	ret, _, _ := procSHGetKnownFolderPath.Call(
		uintptr(unsafe.Pointer(&folderIDDesktop)),
		uintptr(uint32(0)), // KF_FLAG_DEFAULT
		uintptr(0),
		uintptr(unsafe.Pointer(&pathPtr)),
	)
	if ret != 0 {
		return "", syscall.Errno(ret)
	}
	defer procCoTaskMemFree.Call(uintptr(unsafe.Pointer(pathPtr)))
	return syscall.UTF16ToString((*[1 << 16]uint16)(unsafe.Pointer(pathPtr))[:]), nil
}

// CreateShortcut 在 shortcutPath 位置创建一个指向 targetPath 的 Windows 快捷方式。
func CreateShortcut(targetPath, shortcutPath string) error {
	_ = ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED)
	defer ole.CoUninitialize()

	unknown, err := oleutil.CreateObject("WScript.Shell")
	if err != nil {
		return fmt.Errorf("create WScript.Shell: %w", err)
	}
	defer unknown.Release()

	shell, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return fmt.Errorf("query IDispatch: %w", err)
	}
	defer shell.Release()

	shortcut, err := oleutil.CallMethod(shell, "CreateShortcut", shortcutPath)
	if err != nil {
		return fmt.Errorf("create shortcut: %w", err)
	}
	defer shortcut.ToIDispatch().Release()

	if _, err := oleutil.PutProperty(shortcut.ToIDispatch(), "TargetPath", targetPath); err != nil {
		return fmt.Errorf("set target path: %w", err)
	}
	workingDir := filepath.Dir(targetPath)
	if _, err := oleutil.PutProperty(shortcut.ToIDispatch(), "WorkingDirectory", workingDir); err != nil {
		return fmt.Errorf("set working directory: %w", err)
	}
	if _, err := oleutil.CallMethod(shortcut.ToIDispatch(), "Save"); err != nil {
		return fmt.Errorf("save shortcut: %w", err)
	}
	return nil
}

// ResolveShortcutTarget 解析 .lnk 快捷方式指向的真实目标路径。
func ResolveShortcutTarget(shortcutPath string) (string, error) {
	_ = ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED)
	defer ole.CoUninitialize()

	unknown, err := oleutil.CreateObject("WScript.Shell")
	if err != nil {
		return "", fmt.Errorf("create WScript.Shell: %w", err)
	}
	defer unknown.Release()

	shell, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return "", fmt.Errorf("query IDispatch: %w", err)
	}
	defer shell.Release()

	shortcut, err := oleutil.CallMethod(shell, "CreateShortcut", shortcutPath)
	if err != nil {
		return "", fmt.Errorf("open shortcut: %w", err)
	}
	defer shortcut.ToIDispatch().Release()

	target, err := oleutil.GetProperty(shortcut.ToIDispatch(), "TargetPath")
	if err != nil {
		return "", fmt.Errorf("get target path: %w", err)
	}
	return target.ToString(), nil
}

// EnsureSoftwareShortcut 在桌面创建当前软件版本的快捷方式。
// 快捷方式文件名使用 "软件名.lnk"，软件名由文件名去掉 .exe 后缀得到。
func EnsureSoftwareShortcut(softwareDir, filename string) error {
	desktop, err := DesktopDir()
	if err != nil {
		return err
	}
	filename = filepath.Base(filename)
	targetPath := filepath.Join(softwareDir, filename)
	if _, err := os.Stat(targetPath); err != nil {
		return fmt.Errorf("software executable not found: %w", err)
	}
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	shortcutPath := filepath.Join(desktop, name+".lnk")
	return CreateShortcut(targetPath, shortcutPath)
}

// RemoveOldVersionShortcuts 删除桌面上指向 softwareDir 中除 currentFilename 之外 .exe 的快捷方式。
func RemoveOldVersionShortcuts(softwareDir, currentFilename string) error {
	desktop, err := DesktopDir()
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(desktop)
	if err != nil {
		return err
	}
	currentTarget := filepath.Join(softwareDir, currentFilename)
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".lnk") {
			continue
		}
		shortcutPath := filepath.Join(desktop, entry.Name())
		target, err := ResolveShortcutTarget(shortcutPath)
		if err != nil {
			continue
		}
		if !strings.EqualFold(filepath.Dir(target), softwareDir) {
			continue
		}
		if !strings.EqualFold(filepath.Ext(target), ".exe") {
			continue
		}
		if strings.EqualFold(target, currentTarget) {
			continue
		}
		_ = os.Remove(shortcutPath)
	}
	return nil
}
