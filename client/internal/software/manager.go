package software

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// State 记录当前运行或最后一次启动的软件版本与对应文件名，用于程序重启后恢复。
type State struct {
	Version  string `json:"version"`
	Filename string `json:"filename"`
}

// Manager 负责管理被控软件的单一目录和进程生命周期。
// 所有版本的软件可执行文件都存放在同一个目录中，通过文件名进行区分。
type Manager struct {
	mu              sync.RWMutex // mu 保护以下字段的并发访问。
	currentVer      string       // currentVer 是当前运行或最后一次启动的软件版本。
	currentFilename string       // currentFilename 是当前运行或最后一次启动的可执行文件名。
	execPath        string       // execPath 是当前启动的可执行文件路径。
	cmd             *exec.Cmd    // cmd 是当前运行的子进程命令对象。
	startTime       time.Time    // startTime 记录当前进程启动时间。
	softwareDir     string       // softwareDir 是存放所有被控软件文件的根目录。
}

// NewManager 创建一个指定软件根目录的管理器实例，并尝试加载之前保存的运行状态。
func NewManager(softwareDir string) *Manager {
	m := &Manager{
		softwareDir: softwareDir,
	}
	_ = m.loadState()
	return m
}

// CurrentVersion 返回当前运行或最后一次启动的软件版本号。
func (m *Manager) CurrentVersion() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentVer
}

// CurrentFilename 返回当前运行或最后一次启动的可执行文件名。
func (m *Manager) CurrentFilename() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentFilename
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

// Start 在软件目录下启动指定的可执行文件，并记录版本、路径和启动时间。
func (m *Manager) Start(version, filename string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cmd != nil && m.cmd.Process != nil && m.cmd.ProcessState == nil {
		return fmt.Errorf("software already running")
	}
	execPath := filepath.Join(m.softwareDir, filename)
	if _, err := os.Stat(execPath); os.IsNotExist(err) {
		return fmt.Errorf("executable not found: %s", execPath)
	}
	m.execPath = execPath
	m.currentVer = version
	m.currentFilename = filename
	cmd := exec.Command(execPath)
	cmd.Dir = m.softwareDir
	if err := cmd.Start(); err != nil {
		return err
	}
	m.cmd = cmd
	m.startTime = time.Now()
	_ = m.saveState()
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

// EnsureDir 确保软件根目录存在，必要时创建目录，并返回该目录路径。
func (m *Manager) EnsureDir() (string, error) {
	if err := os.MkdirAll(m.softwareDir, 0755); err != nil {
		return "", err
	}
	return m.softwareDir, nil
}

// EnsureVersion 保持向后兼容，忽略版本号，仅确保软件根目录存在。
func (m *Manager) EnsureVersion(_ string) (string, error) {
	return m.EnsureDir()
}

// ExecutablePath 返回指定文件名的可执行文件完整路径。
func (m *Manager) ExecutablePath(filename string) string {
	return filepath.Join(m.softwareDir, filename)
}

// SoftwareDir 返回存放所有被控软件文件的根目录路径。
func (m *Manager) SoftwareDir() string {
	return m.softwareDir
}

// VersionsDir 保持向后兼容，返回软件根目录路径。
func (m *Manager) VersionsDir() string {
	return m.softwareDir
}

// statePath 返回状态文件的路径。
func (m *Manager) statePath() string {
	return filepath.Join(m.softwareDir, ".current.json")
}

// saveState 将当前版本与文件名持久化到状态文件。
func (m *Manager) saveState() error {
	state := State{
		Version:  m.currentVer,
		Filename: m.currentFilename,
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.statePath(), data, 0644)
}

// loadState 从状态文件恢复之前保存的版本与文件名。
func (m *Manager) loadState() error {
	data, err := os.ReadFile(m.statePath())
	if err != nil {
		return err
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}
	m.currentVer = state.Version
	m.currentFilename = state.Filename
	return nil
}

// FindExecutableForVersion 在软件目录中查找与指定版本匹配的可执行文件。
// 优先查找文件名包含版本号的 .exe 文件，找不到则返回目录中第一个 .exe 文件。
func (m *Manager) FindExecutableForVersion(version string) (string, error) {
	entries, err := os.ReadDir(m.softwareDir)
	if err != nil {
		return "", err
	}
	var firstExe string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".exe") {
			continue
		}
		if firstExe == "" {
			firstExe = name
		}
		if version != "" && strings.Contains(strings.ToLower(name), strings.ToLower(version)) {
			return name, nil
		}
	}
	if firstExe != "" {
		return firstExe, nil
	}
	return "", fmt.Errorf("no executable found in %s", m.softwareDir)
}
