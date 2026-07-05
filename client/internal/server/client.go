package server

import (
	"bytes"
	"encoding/json"
	"example.com/appupdatemanager/client/internal/config"
	"example.com/appupdatemanager/client/internal/logger"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Message 表示 WebSocket 消息的通用结构，包含消息类型和相关数据。
type Message struct {
	Type string      `json:"type"` // Type 表示消息类型，如 register、heartbeat、command。
	Data interface{} `json:"data"` // Data 是消息携带的具体数据。
}

// HeartbeatData 是客户端向服务器发送心跳时携带的载荷数据。
type HeartbeatData struct {
	Name            string `json:"name"`             // Name 是客户端名称。
	ClientVersion   string `json:"client_version"`   // ClientVersion 是客户端版本号。
	SoftwareVersion string `json:"software_version"` // SoftwareVersion 是当前管理的软件版本。
	IsRunning       bool   `json:"is_running"`       // IsRunning 表示管理的软件是否正在运行。
	IP              string `json:"ip"`               // IP 是客户端本机 IP 地址。
	OSVersion       string `json:"os_version"`       // OSVersion 是操作系统版本信息。
	Memory          string `json:"memory"`           // Memory 是内存使用情况描述。
	CPU             string `json:"cpu"`              // CPU 是 CPU 型号信息。
	ProcessRuntime  int64  `json:"process_runtime"`  // ProcessRuntime 是软件运行时长，单位秒。
}

// CommandHandler 是处理服务器下发命令的回调函数类型。
type CommandHandler func(cmd string, payload map[string]string)

// Client 管理客户端与服务器之间的 WebSocket 连接、心跳和命令处理。
type Client struct {
	cfg            *config.Config  // cfg 是客户端配置。
	conn           *websocket.Conn // conn 是 WebSocket 连接。
	logger         *logger.Logger  // logger 用于记录服务器交互日志。
	handler        CommandHandler  // handler 是接收服务器命令的回调。
	stopCh         chan struct{}   // stopCh 用于通知连接相关的 goroutine 退出。
	wg             sync.WaitGroup  // wg 等待读写 goroutine 结束。
	mu             sync.RWMutex    // mu 保护状态字段的并发访问。
	softwareVer    string          // softwareVer 是当前软件版本。
	isRunning      bool            // isRunning 表示软件是否运行中。
	processRuntime int64           // processRuntime 是软件运行时长，单位秒。
	connected      bool            // connected 表示当前是否处于连接状态。
	closed         bool            // closed 表示 Client 是否已被关闭，关闭后不再重连。
}

// NewClient 创建一个新的服务器客户端实例。
func NewClient(cfg *config.Config, log *logger.Logger, handler CommandHandler) *Client {
	return &Client{
		cfg:     cfg,
		logger:  log,
		handler: handler,
		stopCh:  make(chan struct{}),
	}
}

// SetStatus 更新客户端下次心跳时要上报的软件版本、运行状态和运行时长。
func (c *Client) SetStatus(softwareVer string, isRunning bool, processRuntime int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.softwareVer = softwareVer
	c.isRunning = isRunning
	c.processRuntime = processRuntime
}

// Status 返回当前保存的软件版本、运行状态和运行时长。
func (c *Client) Status() (string, bool, int64) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.softwareVer, c.isRunning, c.processRuntime
}

// IsConnected 返回当前是否处于连接状态。
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// Run 启动与服务器的连接，并在断开后每 15 秒尝试重新连接，直到 Close 被调用。
func (c *Client) Run(sysInfo *HeartbeatData) {
	for {
		c.mu.RLock()
		closed := c.closed
		c.mu.RUnlock()
		if closed {
			return
		}

		if err := c.Connect(sysInfo); err != nil {
			time.Sleep(15 * time.Second)
			continue
		}

		// 等待读写 goroutine 结束（即连接断开）
		c.wg.Wait()
		c.setConnected(false)

		c.mu.Lock()
		c.stopCh = make(chan struct{})
		closed = c.closed
		c.mu.Unlock()
		if closed {
			return
		}

		if c.logger != nil {
			c.logger.Info("连接已断开，15 秒后尝试重连")
		}
		time.Sleep(15 * time.Second)
	}
}

// SetConfig 在运行时更新客户端使用的配置，下次连接或重连会使用新的服务器地址。
func (c *Client) SetConfig(cfg *config.Config) {
	c.mu.Lock()
	c.cfg = cfg
	c.mu.Unlock()
}

// Reset 重置客户端连接状态，允许在 Close 之后再次调用 Run。
func (c *Client) Reset() {
	c.mu.Lock()
	c.closed = false
	c.connected = false
	c.stopCh = make(chan struct{})
	c.mu.Unlock()
}

// Connect 建立与服务器的 WebSocket 连接，发送注册消息并启动读写 goroutine。
func (c *Client) Connect(sysInfo *HeartbeatData) error {
	c.mu.Lock()
	if c.connected {
		c.mu.Unlock()
		return nil
	}
	c.mu.Unlock()

	if c.logger != nil {
		c.logger.Infof("正在连接服务器: %s", c.cfg.WSURL())
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}
	conn, _, err := dialer.Dial(c.cfg.WSURL(), nil)
	if err != nil {
		if c.logger != nil {
			c.logger.Errorf("连接服务器失败: %v", err)
		}
		return err
	}
	c.conn = conn

	// Send register message
	if err := c.send(Message{Type: "register", Data: map[string]string{"name": c.cfg.ClientName}}); err != nil {
		conn.Close()
		if c.logger != nil {
			c.logger.Errorf("发送注册消息失败: %v", err)
		}
		return err
	}
	if c.logger != nil {
		c.logger.Infof("已连接服务器并注册，客户端名称: %s", c.cfg.ClientName)
	}

	c.setConnected(true)
	c.wg.Add(2)
	go c.readLoop()
	go c.heartbeatLoop(sysInfo)
	return nil
}

// Close 关闭 WebSocket 连接，通知后台 goroutine 退出并等待它们结束。
func (c *Client) Close() error {
	c.mu.Lock()
	c.closed = true
	close(c.stopCh)
	if c.conn != nil {
		c.conn.Close()
	}
	c.mu.Unlock()
	c.wg.Wait()
	c.setConnected(false)
	if c.logger != nil {
		c.logger.Info("已关闭服务器连接")
	}
	return nil
}

func (c *Client) setConnected(v bool) {
	c.mu.Lock()
	c.connected = v
	c.mu.Unlock()
}

// send 将 Message 序列化为 JSON 并通过 WebSocket 发送。
func (c *Client) send(msg Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return c.conn.WriteMessage(websocket.TextMessage, data)
}

// readLoop 持续读取服务器下发的 WebSocket 消息，解析并调用命令回调处理命令消息。
func (c *Client) readLoop() {
	defer c.wg.Done()
	for {
		select {
		case <-c.stopCh:
			return
		default:
		}
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			select {
			case <-c.stopCh:
				return
			default:
				if c.logger != nil {
					c.logger.Errorf("读取服务器消息失败，连接断开: %v", err)
				}
				return
			}
		}
		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			if c.logger != nil {
				c.logger.Warnf("解析服务器消息失败: %v, 原始数据: %s", err, string(data))
			}
			continue
		}

		if c.logger != nil {
			c.logger.Infof("收到服务器消息，类型: %s", msg.Type)
		}

		if msg.Type == "command" {
			payloadBytes, _ := json.Marshal(msg.Data)
			var payload map[string]string
			json.Unmarshal(payloadBytes, &payload)
			if c.logger != nil {
				c.logger.Infof("收到服务器命令: %s, 参数: %s", payload["command"], string(payloadBytes))
			}
			if c.handler != nil {
				c.handler(payload["command"], payload)
			}
		}
	}
}

// heartbeatLoop 定期向服务器发送心跳消息，上报客户端和软件状态。
func (c *Client) heartbeatLoop(sysInfo *HeartbeatData) {
	defer c.wg.Done()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.mu.RLock()
			sysInfo.SoftwareVersion = c.softwareVer
			sysInfo.IsRunning = c.isRunning
			sysInfo.ProcessRuntime = c.processRuntime
			c.mu.RUnlock()
			sysInfo.ClientVersion = c.cfg.ClientVersion
			sysInfo.Name = c.cfg.ClientName
			if err := c.send(Message{Type: "heartbeat", Data: sysInfo}); err != nil {
				if c.logger != nil {
					c.logger.Errorf("发送心跳失败: %v", err)
				}
				return
			}
			if c.logger != nil {
				c.logger.Info("已发送心跳")
			}
		}
	}
}

// DownloadFile 从服务器下载文件并保存到指定本地路径。
func DownloadFile(log *logger.Logger, serverURL, relativeURL, savePath string) error {
	url := serverURL + relativeURL
	if log != nil {
		log.Infof("开始下载文件: %s -> %s", url, savePath)
	}
	resp, err := http.Get(url)
	if err != nil {
		if log != nil {
			log.Errorf("下载文件请求失败: %v", err)
		}
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if log != nil {
			log.Errorf("下载文件失败: %s", resp.Status)
		}
		return fmt.Errorf("download failed: %s", resp.Status)
	}
	out, err := os.Create(savePath)
	if err != nil {
		if log != nil {
			log.Errorf("创建本地文件失败: %v", err)
		}
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		if log != nil {
			log.Errorf("保存下载文件失败: %v", err)
		}
		return err
	}
	if log != nil {
		log.Infof("文件下载完成: %s", savePath)
	}
	return nil
}

// DownloadFileWithProgress 从服务器下载文件并保存到指定本地路径，期间通过回调上报进度。
// onProgress 参数分别为：已下载字节数、总字节数（若服务器返回 Content-Length，否则为 -1）、人类可读速度。
func DownloadFileWithProgress(log *logger.Logger, serverURL, relativeURL, savePath string, onProgress func(downloaded, total int64, speed string)) error {
	url := serverURL + relativeURL
	if log != nil {
		log.Infof("开始下载文件: %s -> %s", url, savePath)
	}
	resp, err := http.Get(url)
	if err != nil {
		if log != nil {
			log.Errorf("下载文件请求失败: %v", err)
		}
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if log != nil {
			log.Errorf("下载文件失败: %s", resp.Status)
		}
		return fmt.Errorf("download failed: %s", resp.Status)
	}

	out, err := os.Create(savePath)
	if err != nil {
		if log != nil {
			log.Errorf("创建本地文件失败: %v", err)
		}
		return err
	}
	defer out.Close()

	total := resp.ContentLength
	pr := &progressReader{
		reader:     resp.Body,
		onProgress: onProgress,
		total:      total,
	}
	_, err = io.Copy(out, pr)
	if err != nil {
		if log != nil {
			log.Errorf("保存下载文件失败: %v", err)
		}
		return err
	}
	if onProgress != nil {
		onProgress(pr.downloaded, total, humanSpeed(pr.lastSpeed))
	}
	if log != nil {
		log.Infof("文件下载完成: %s", savePath)
	}
	return nil
}

// progressReader 是一个包装 io.Reader 以统计下载进度和速度的读取器。
type progressReader struct {
	reader     io.Reader
	onProgress func(downloaded, total int64, speed string)
	total      int64
	downloaded int64
	lastSpeed  float64
	lastTime   time.Time
	lastReport time.Time
}

func (p *progressReader) Read(buf []byte) (int, error) {
	n, err := p.reader.Read(buf)
	if n > 0 {
		p.downloaded += int64(n)
		now := time.Now()
		if p.lastTime.IsZero() {
			p.lastTime = now
			p.lastReport = now
		}
		elapsed := now.Sub(p.lastTime).Seconds()
		if elapsed > 0 {
			p.lastSpeed = float64(p.downloaded) / elapsed
		}
		if p.onProgress != nil && (now.Sub(p.lastReport) >= 500*time.Millisecond || err == io.EOF) {
			p.onProgress(p.downloaded, p.total, humanSpeed(p.lastSpeed))
			p.lastReport = now
		}
	}
	return n, err
}

// humanSpeed 将字节/秒转换为人类可读的字符串。
func humanSpeed(bps float64) string {
	switch {
	case bps >= 1024*1024*1024:
		return fmt.Sprintf("%.2f GB/s", bps/1024/1024/1024)
	case bps >= 1024*1024:
		return fmt.Sprintf("%.2f MB/s", bps/1024/1024)
	case bps >= 1024:
		return fmt.Sprintf("%.2f KB/s", bps/1024)
	default:
		return fmt.Sprintf("%.2f B/s", bps)
	}
}

// CommandFeedback 是客户端向服务端上报的命令执行反馈。
type CommandFeedback struct {
	CommandID int64  `json:"command_id"`
	Status    string `json:"status"`
	Progress  int    `json:"progress"`
	Speed     string `json:"speed"`
	Message   string `json:"message"`
}

// SendCommandFeedback 向服务端发送一条命令执行反馈消息。
func (c *Client) SendCommandFeedback(commandID string, status string, progress int, speed, message string) error {
	id, err := strconv.ParseInt(commandID, 10, 64)
	if err != nil {
		return err
	}
	return c.send(Message{
		Type: "command_feedback",
		Data: CommandFeedback{
			CommandID: id,
			Status:    status,
			Progress:  progress,
			Speed:     speed,
			Message:   message,
		},
	})
}

// ReportStatus 通过 HTTP POST 向服务器发送一次性的状态报告。
func ReportStatus(log *logger.Logger, serverURL string, status *HeartbeatData) error {
	data, err := json.Marshal(status)
	if err != nil {
		return err
	}
	url := serverURL + "/api/clients/status"
	if log != nil {
		log.Infof("正在上报状态到: %s", url)
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		if log != nil {
			log.Errorf("上报状态失败: %v", err)
		}
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if log != nil {
			log.Errorf("上报状态失败: %s, 响应: %s", resp.Status, string(body))
		}
		return fmt.Errorf("report status failed: %s", string(body))
	}
	if log != nil {
		log.Info("状态上报成功")
	}
	return nil
}
