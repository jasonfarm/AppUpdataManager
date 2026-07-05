// Package logger 提供客户端日志记录功能，将日志同时写入本地文件并保留在内存中供 UI 展示。
package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// Level 表示日志级别。
type Level string

const (
	// LevelInfo 是普通信息级别。
	LevelInfo Level = "INFO"
	// LevelWarn 是警告级别。
	LevelWarn Level = "WARN"
	// LevelError 是错误级别。
	LevelError Level = "ERROR"
)

// autoScrollResumeDelay 是手动滚动后恢复自动滚动的时间间隔。
const autoScrollResumeDelay = 10 * time.Minute

// Entry 表示一条日志记录。
type Entry struct {
	Time    time.Time // Time 是日志产生时间。
	Level   Level     // Level 是日志级别。
	Message string    // Message 是日志内容。
}

// Logger 将日志写入文件并保留最近的若干条记录，同时支持 UI 监听器实时刷新。
type Logger struct {
	mu         sync.RWMutex
	file       *os.File
	entries    []Entry
	listeners  []func(Entry)
	maxEntries int
}

// New 创建一个新的 Logger，日志会追加写入 filePath 指定的文件。
func New(filePath string) (*Logger, error) {
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	return &Logger{
		file:       f,
		maxEntries: 1000,
	}, nil
}

// Close 关闭日志文件。
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil {
		err := l.file.Close()
		l.file = nil
		return err
	}
	return nil
}

// SetMaxEntries 动态设置内存中保留的最大日志条数。
func (l *Logger) SetMaxEntries(n int) {
	if n <= 0 {
		n = 1000
	}
	l.mu.Lock()
	l.maxEntries = n
	if len(l.entries) > l.maxEntries {
		l.entries = l.entries[len(l.entries)-l.maxEntries:]
	}
	listeners := make([]func(Entry), len(l.listeners))
	copy(listeners, l.listeners)
	l.mu.Unlock()

	// 通知所有 view 重新刷新数据以应用新的上限。
	for _, listener := range listeners {
		listener(Entry{})
	}
}

// MaxEntries 返回当前内存中保留的最大日志条数。
func (l *Logger) MaxEntries() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.maxEntries
}

// AddListener 注册一个回调，每条新日志产生时都会调用该回调。
func (l *Logger) AddListener(listener func(Entry)) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.listeners = append(l.listeners, listener)
}

// Entries 返回当前保留在内存中的日志副本。
func (l *Logger) Entries() []Entry {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return append([]Entry{}, l.entries...)
}

func (l *Logger) log(level Level, msg string) {
	entry := Entry{
		Time:    time.Now(),
		Level:   level,
		Message: msg,
	}

	l.mu.Lock()
	l.entries = append(l.entries, entry)
	if len(l.entries) > l.maxEntries {
		l.entries = l.entries[len(l.entries)-l.maxEntries:]
	}
	listeners := make([]func(Entry), len(l.listeners))
	copy(listeners, l.listeners)
	file := l.file
	l.mu.Unlock()

	line := fmt.Sprintf("[%s] [%s] %s\n", entry.Time.Format("2006-01-02 15:04:05"), level, msg)
	if file != nil {
		_, _ = file.WriteString(line)
	}

	for _, listener := range listeners {
		listener(entry)
	}
}

// Info 记录一条 INFO 级别日志。
func (l *Logger) Info(msg string) {
	l.log(LevelInfo, msg)
}

// Infof 使用格式化字符串记录 INFO 级别日志。
func (l *Logger) Infof(format string, args ...interface{}) {
	l.log(LevelInfo, fmt.Sprintf(format, args...))
}

// Warn 记录一条 WARN 级别日志。
func (l *Logger) Warn(msg string) {
	l.log(LevelWarn, msg)
}

// Warnf 使用格式化字符串记录 WARN 级别日志。
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.log(LevelWarn, fmt.Sprintf(format, args...))
}

// Error 记录一条 ERROR 级别日志。
func (l *Logger) Error(msg string) {
	l.log(LevelError, msg)
}

// Errorf 使用格式化字符串记录 ERROR 级别日志。
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log(LevelError, fmt.Sprintf(format, args...))
}

// CreateView 创建一个 Fyne UI 视图，用于展示日志列表。
// 视图默认自动滚动到最新日志；用户手动滚动后会暂停自动滚动，10 分钟后恢复。
func (l *Logger) CreateView() fyne.CanvasObject {
	label := widget.NewLabel("")
	label.Wrapping = fyne.TextWrapBreak
	label.TextStyle = fyne.TextStyle{Monospace: true}

	scroll := container.NewScroll(label)
	scroll.SetMinSize(fyne.NewSize(580, 400))

	var (
		autoScroll = true
		pausedAt   time.Time
		mu         sync.Mutex
	)

	rebuildLabel := func() {
		l.mu.RLock()
		lines := make([]string, len(l.entries))
		for i, e := range l.entries {
			lines[i] = formatEntry(e)
		}
		l.mu.RUnlock()
		label.SetText(strings.Join(lines, "\n"))
	}

	scrollToBottom := func() {
		scroll.ScrollToBottom()
	}

	isNearBottom := func() bool {
		contentHeight := scroll.Content.MinSize().Height
		scrollHeight := scroll.Size().Height
		if contentHeight <= scrollHeight {
			return true
		}
		return scroll.Offset.Y >= contentHeight-scrollHeight-5
	}

	scroll.OnScrolled = func(pos fyne.Position) {
		mu.Lock()
		defer mu.Unlock()
		if isNearBottom() {
			autoScroll = true
		} else {
			autoScroll = false
			pausedAt = time.Now()
		}
	}

	l.AddListener(func(e Entry) {
		mu.Lock()
		shouldAuto := autoScroll
		if !autoScroll && !pausedAt.IsZero() && time.Since(pausedAt) >= autoScrollResumeDelay {
			autoScroll = true
			shouldAuto = true
		}
		mu.Unlock()

		rebuildLabel()

		if shouldAuto {
			scrollToBottom()
		}
	})

	rebuildLabel()
	scrollToBottom()

	return scroll
}

func formatEntry(e Entry) string {
	return fmt.Sprintf("[%s] [%s] %s", e.Time.Format("2006-01-02 15:04:05"), e.Level, e.Message)
}
