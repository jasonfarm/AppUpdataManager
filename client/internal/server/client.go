package server

import (
	"bytes"
	"encoding/json"
	"example.com/appupdatemanager/client/internal/config"
	"fmt"
	"io"
	"net/http"
	"os"
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
	handler        CommandHandler  // handler 是接收服务器命令的回调。
	stopCh         chan struct{}   // stopCh 用于通知连接相关的 goroutine 退出。
	wg             sync.WaitGroup  // wg 等待读写 goroutine 结束。
	mu             sync.RWMutex    // mu 保护状态字段的并发访问。
	softwareVer    string          // softwareVer 是当前软件版本。
	isRunning      bool            // isRunning 表示软件是否运行中。
	processRuntime int64           // processRuntime 是软件运行时长，单位秒。
}

// NewClient 创建一个新的服务器客户端实例。
func NewClient(cfg *config.Config, handler CommandHandler) *Client {
	return &Client{
		cfg:     cfg,
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

// Connect 建立与服务器的 WebSocket 连接，发送注册消息并启动读写 goroutine。
func (c *Client) Connect(sysInfo *HeartbeatData) error {
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}
	conn, _, err := dialer.Dial(c.cfg.WSURL(), nil)
	if err != nil {
		return err
	}
	c.conn = conn

	// Send register message
	if err := c.send(Message{Type: "register", Data: map[string]string{"name": c.cfg.ClientName}}); err != nil {
		conn.Close()
		return err
	}

	c.wg.Add(2)
	go c.readLoop()
	go c.heartbeatLoop(sysInfo)
	return nil
}

// Close 关闭 WebSocket 连接，通知后台 goroutine 退出并等待它们结束。
func (c *Client) Close() error {
	close(c.stopCh)
	if c.conn != nil {
		c.conn.Close()
	}
	c.wg.Wait()
	return nil
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
				// reconnect handled by caller
				return
			}
		}
		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}
		if msg.Type == "command" {
			payloadBytes, _ := json.Marshal(msg.Data)
			var payload map[string]string
			json.Unmarshal(payloadBytes, &payload)
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
				return
			}
		}
	}
}

// DownloadFile 从服务器下载文件并保存到指定本地路径。
func DownloadFile(serverURL, relativeURL, savePath string) error {
	url := serverURL + relativeURL
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %s", resp.Status)
	}
	out, err := os.Create(savePath)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}

// ReportStatus 通过 HTTP POST 向服务器发送一次性的状态报告。
func ReportStatus(serverURL string, status *HeartbeatData) error {
	data, err := json.Marshal(status)
	if err != nil {
		return err
	}
	url := serverURL + "/api/clients/status"
	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("report status failed: %s", string(body))
	}
	return nil
}
