package ws

import (
	"example.com/appupdatemanager/server/internal/model"
	"example.com/appupdatemanager/server/internal/store"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// upgrader 用于将 HTTP 连接升级为 WebSocket 连接，允许任意来源。
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// ClientConn 表示一个与客户端建立的 WebSocket 连接。
type ClientConn struct {
	// Hub 所属连接管理中心。
	Hub *Hub
	// Name 客户端名称，由注册或心跳消息上报。
	Name string
	// Conn 底层 WebSocket 连接。
	Conn *websocket.Conn
	// Send 用于向客户端发送消息的通道。
	Send chan []byte
	// ClientID 客户端在数据库中的唯一标识。
	ClientID int64
}

// Hub 维护所有活跃的客户端 WebSocket 连接，并负责注册、注销与广播。
type Hub struct {
	// clients 以客户端名称为键保存所有连接。
	clients    map[string]*ClientConn
	// register 用于接收新连接注册请求的通道。
	register   chan *ClientConn
	// unregister 用于接收连接注销请求的通道。
	unregister chan *ClientConn
	// broadcast 用于向所有客户端广播消息的通道。
	broadcast  chan []byte
	// db 是数据库连接，用于持久化客户端状态与命令。
	db         *store.DB
	// mu 保护 clients 的读写并发安全。
	mu         sync.RWMutex
}

// NewHub 创建一个 Hub 实例，需传入数据库连接以持久化客户端信息。
func NewHub(db *store.DB) *Hub {
	return &Hub{
		clients:    make(map[string]*ClientConn),
		register:   make(chan *ClientConn),
		unregister: make(chan *ClientConn),
		broadcast:  make(chan []byte),
		db:         db,
	}
}

// Run 启动 Hub 的事件循环，处理客户端注册、注销与广播消息。
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.Name] = client
			h.mu.Unlock()
			log.Printf("client registered: %s", client.Name)
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.Name]; ok {
				delete(h.clients, client.Name)
				close(client.Send)
			}
			h.mu.Unlock()
			log.Printf("client unregistered: %s", client.Name)
		case message := <-h.broadcast:
			h.mu.RLock()
			for _, client := range h.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// SendToClient 根据客户端名称发送命令载荷，若目标不在线则不发送。
func (h *Hub) SendToClient(name string, payload model.CommandPayload) {
	msg := model.WSMessage{Type: "command", Data: payload}
	data, _ := json.Marshal(msg)
	h.mu.RLock()
	client, ok := h.clients[name]
	h.mu.RUnlock()
	if ok {
		select {
		case client.Send <- data:
		default:
			log.Printf("client %s send channel full", name)
		}
	}
}

// Serve 将 HTTP 请求升级为 WebSocket 连接，并启动读写 goroutine。
func Serve(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade error: %v", err)
		return
	}
	client := &ClientConn{Hub: hub, Conn: conn, Send: make(chan []byte, 256)}
	go client.writePump()
	go client.readPump()
}

// readPump 持续从 WebSocket 连接读取消息，解析后交给 handleClientMessage 处理。
func (c *ClientConn) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("websocket read error: %v", err)
			}
			break
		}
		handleClientMessage(c, message)
	}
}

// writePump 持续监听发送通道与心跳定时器，向客户端下发消息或 ping 帧。
func (c *ClientConn) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleClientMessage 根据 WebSocket 消息类型处理注册、心跳等消息，并下发待执行命令。
func handleClientMessage(c *ClientConn, data []byte) {
	var msg model.WSMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		log.Printf("invalid ws message: %v", err)
		return
	}
	switch msg.Type {
	case "register":
		var payload map[string]string
		payloadBytes, _ := json.Marshal(msg.Data)
		json.Unmarshal(payloadBytes, &payload)
		c.Name = payload["name"]
		if c.Name == "" {
			c.Name = "unknown"
		}
		c.Hub.register <- c
	case "heartbeat":
		var hb model.HeartbeatData
		payloadBytes, _ := json.Marshal(msg.Data)
		if err := json.Unmarshal(payloadBytes, &hb); err != nil {
			log.Printf("invalid heartbeat: %v", err)
			return
		}
		if c.Name == "" {
			c.Name = hb.Name
			c.Hub.register <- c
		}
		client := &model.Client{
			Name:            hb.Name,
			ClientVersion:   hb.ClientVersion,
			SoftwareVersion: hb.SoftwareVersion,
			Status:          "online",
			IsRunning:       hb.IsRunning,
			IP:              hb.IP,
			OSVersion:       hb.OSVersion,
			Memory:          hb.Memory,
			CPU:             hb.CPU,
			ProcessRuntime:  hb.ProcessRuntime,
			LastSeen:        time.Now(),
		}
		if err := store.UpsertClient(c.Hub.db, client); err != nil {
			log.Printf("upsert client error: %v", err)
			return
		}
		c.ClientID = client.ID
		// Send any pending commands
		commands, err := store.ListPendingCommands(c.Hub.db, client.ID)
		if err != nil {
			log.Printf("list pending commands error: %v", err)
			return
		}
		for _, cmd := range commands {
			c.Send <- []byte(cmd.Payload)
			store.UpdateCommandStatus(c.Hub.db, cmd.ID, "sent")
		}
	default:
		log.Printf("unknown message type: %s", msg.Type)
	}
}
