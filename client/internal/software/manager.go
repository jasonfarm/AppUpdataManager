package software

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// Manager 负责管理所控软件 exe 的版本目录和进程生命周期。
type Manager struct {
	mu          sync.RWMutex // mu 保护以下字段的并发访问。
	currentVer  string       // currentVer 是当前运行或最后一次启动的软件版本。
	execPath    string       // execPath 是当前启动的可执行文件路径。
	cmd         *exec.Cmd    // cmd 是当前运行的子进程命令对象。
	startTime   time.Time    // startTime 记录当前进程启动时间。
	versionsDir string       // versionsDir 是存放所有版本子目录的根目录。
}

// NewManager 创建一个指定版本根目录的软件管理器实例。
func NewManager(versionsDir string) *Manager {
	return &Manager{
		versionsDir: versionsDir,
	}
}

// CurrentVersion 返回当前运行或最后一次启动的软件版本号。
func (m *Manager) CurrentVersion() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentVer
}

// IsRunning 返回当前管理的软件进程是否正在运行。
func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.cmd == nil || m.cmd.Process == nil {
		return false
	}
	return m.cmd.ProcessState == nil || !m.cmd.ProcessState.Exited()
}

// Runtime 返回当前软件进程已运行的秒数，如果未启动则返回 0。
func (m *Manager) Runtime() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.startTime.IsZero() {
		return 0
	}
	return int64(time.Since(m.startTime).Seconds())
}

// Start 在指定版本目录下启动指定的可执行文件，并记录版本、路径和启动时间。
func (m *Manager) Start(version, filename string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cmd != nil && m.cmd.Process != nil && m.cmd.ProcessState == nil {
		return fmt.Errorf("software already running")
	}
	execPath := filepath.Join(m.versionsDir, version, filename)
	if _, err := os.Stat(execPath); os.IsNotExist(err) {
		return fmt.Errorf("executable not found: %s", execPath)
	}
	m.execPath = execPath
	m.currentVer = version
	cmd := exec.Command(execPath)
	cmd.Dir = filepath.Dir(execPath)
	if err := cmd.Start(); err != nil {
		return err
	}
	m.cmd = cmd
	m.startTime = time.Now()
	return nil
}

// Stop 停止当前正在运行的软件进程，并清理相关状态。
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cmd == nil || m.cmd.Process == nil {
		return nil
	}
	if err := m.cmd.Process.Kill(); err != nil {
		return err
	}
	_, _ = m.cmd.Process.Wait()
	m.cmd = nil
	m.startTime = time.Time{}
	return nil
}

// Restart 先停止当前运行的软件，然后以指定版本和文件名重新启动。
func (m *Manager) Restart(version, filename string) error {
	if err := m.Stop(); err != nil {
		return err
	}
	return m.Start(version, filename)
}

// EnsureVersion 确保指定版本的目录存在，必要时创建目录，并返回该目录路径。
func (m *Manager) EnsureVersion(version string) (string, error) {
	dir := filepath.Join(m.versionsDir, version)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

// ExecutablePath 返回指定版本和文件名的可执行文件完整路径。
func (m *Manager) ExecutablePath(version, filename string) string {
	return filepath.Join(m.versionsDir, version, filename)
}

// VersionsDir 返回存放所有版本子目录的根目录路径。
func (m *Manager) VersionsDir() string {
	return m.versionsDir
}
