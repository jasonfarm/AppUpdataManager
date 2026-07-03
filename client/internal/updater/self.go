package updater

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// SelfUpdate 在 Windows 上执行客户端自更新：下载新 exe 后通过批处理替换当前可执行文件并重启。
func SelfUpdate(newExePath string) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("self update only implemented for windows")
	}
	currentExe, err := os.Executable()
	if err != nil {
		return err
	}
	currentExe, err = filepath.EvalSymlinks(currentExe)
	if err != nil {
		return err
	}

	batPath := filepath.Join(filepath.Dir(currentExe), "updater.bat")
	batContent := fmt.Sprintf(`@echo off
timeout /t 2 /nobreak >nul
move /Y "%s" "%s"
start "" "%s"
del "%%~f0"
`, newExePath, currentExe, currentExe)
	if err := os.WriteFile(batPath, []byte(batContent), 0755); err != nil {
		return err
	}
	cmd := exec.Command("cmd", "/C", batPath)
	if err := cmd.Start(); err != nil {
		return err
	}
	os.Exit(0)
	return nil
}

// DownloadPath 返回新客户端可执行文件应保存的临时路径（当前 exe 路径加 .new 后缀）。
func DownloadPath() (string, error) {
	currentExe, err := os.Executable()
	if err != nil {
		return "", err
	}
	currentExe, err = filepath.EvalSymlinks(currentExe)
	if err != nil {
		return "", err
	}
	return currentExe + ".new", nil
}
