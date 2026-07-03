package model

import "time"

// User 表示 Web 控制台用户账户。
type User struct {
	// ID 用户唯一标识。
	ID int64 `json:"id"`
	// Username 登录用户名。
	Username string `json:"username"`
	// Password 明文密码，仅在从配置文件加载时使用，不会序列化到 JSON。
	Password string `json:"-"`
	// PasswordHash 经过哈希后的密码，存储于数据库，不会序列化到 JSON。
	PasswordHash string `json:"-"`
}

// SoftwareVersion 表示被管理软件的一个版本。
type SoftwareVersion struct {
	// ID 版本唯一标识。
	ID int64 `json:"id"`
	// Name 软件显示名称。
	Name string `json:"name"`
	// Version 软件版本号。
	Version string `json:"version"`
	// Filename 原始上传文件名。
	Filename string `json:"filename"`
	// Filepath 文件在服务器本地存储的绝对路径。
	Filepath string `json:"filepath"`
	// IsLatest 是否为当前最新版本。
	IsLatest bool `json:"is_latest"`
	// CreatedAt 版本创建时间。
	CreatedAt time.Time `json:"created_at"`
}

// ClientVersion 表示客户端程序自身的一个版本。
type ClientVersion struct {
	// ID 版本唯一标识。
	ID int64 `json:"id"`
	// Version 客户端版本号。
	Version string `json:"version"`
	// Filename 原始上传文件名。
	Filename string `json:"filename"`
	// Filepath 文件在服务器本地存储的绝对路径。
	Filepath string `json:"filepath"`
	// IsLatest 是否为当前最新版本。
	IsLatest bool `json:"is_latest"`
	// CreatedAt 版本创建时间。
	CreatedAt time.Time `json:"created_at"`
}

// ResourcePackage 表示一个资源包（zip 压缩包），用于随软件版本一起下发给客户端。
type ResourcePackage struct {
	// ID 资源包唯一标识。
	ID int64 `json:"id"`
	// Name 资源包显示名称。
	Name string `json:"name"`
	// Version 资源包版本号。
	Version string `json:"version"`
	// Filename 原始上传文件名。
	Filename string `json:"filename"`
	// Filepath 文件在服务器本地存储的绝对路径。
	Filepath string `json:"filepath"`
	// IsLatest 是否为当前最新资源包。
	IsLatest bool `json:"is_latest"`
	// CreatedAt 资源包创建时间。
	CreatedAt time.Time `json:"created_at"`
}

// Client 表示一个已注册或已连接的客户端。
type Client struct {
	// ID 客户端唯一标识。
	ID int64 `json:"id"`
	// Name 客户端名称（由客户端上报）。
	Name string `json:"name"`
	// ClientVersion 客户端程序当前版本。
	ClientVersion string `json:"client_version"`
	// SoftwareVersion 客户端上被管理软件的当前版本。
	SoftwareVersion string `json:"software_version"`
	// Status 客户端在线状态，如 online。
	Status string `json:"status"`
	// IsRunning 被管理软件是否正在运行。
	IsRunning bool `json:"is_running"`
	// IP 客户端 IP 地址。
	IP string `json:"ip"`
	// OSVersion 客户端操作系统版本。
	OSVersion string `json:"os_version"`
	// Memory 客户端内存信息。
	Memory string `json:"memory"`
	// CPU 客户端 CPU 信息。
	CPU string `json:"cpu"`
	// ProcessRuntime 被管理软件进程已运行时长（秒）。
	ProcessRuntime int64 `json:"process_runtime"`
	// LastSeen 最近一次心跳时间。
	LastSeen time.Time `json:"last_seen"`
	// CreatedAt 客户端首次注册时间。
	CreatedAt time.Time `json:"created_at"`
}

// ClientCommand 表示服务器向客户端下发的命令。
type ClientCommand struct {
	// ID 命令唯一标识。
	ID int64 `json:"id"`
	// ClientID 目标客户端 ID。
	ClientID int64 `json:"client_id"`
	// CommandType 命令类型，如 update_software、start、stop 等。
	CommandType string `json:"command_type"`
	// Payload 命令的 JSON 载荷。
	Payload string `json:"payload"`
	// Status 命令状态，如 pending、sent。
	Status string `json:"status"`
	// CreatedAt 命令创建时间。
	CreatedAt time.Time `json:"created_at"`
}

// HeartbeatData 表示客户端心跳消息中上报的数据。
type HeartbeatData struct {
	// Name 客户端名称。
	Name string `json:"name"`
	// ClientVersion 客户端当前版本。
	ClientVersion string `json:"client_version"`
	// SoftwareVersion 被管理软件当前版本。
	SoftwareVersion string `json:"software_version"`
	// IsRunning 被管理软件是否正在运行。
	IsRunning bool `json:"is_running"`
	// IP 客户端 IP 地址。
	IP string `json:"ip"`
	// OSVersion 客户端操作系统版本。
	OSVersion string `json:"os_version"`
	// Memory 客户端内存信息。
	Memory string `json:"memory"`
	// CPU 客户端 CPU 信息。
	CPU string `json:"cpu"`
	// ProcessRuntime 被管理软件进程已运行时长（秒）。
	ProcessRuntime int64 `json:"process_runtime"`
}

// WSMessage 是 WebSocket 通信的通用消息信封。
type WSMessage struct {
	// Type 消息类型，如 register、heartbeat、command。
	Type string `json:"type"`
	// Data 消息的具体载荷。
	Data interface{} `json:"data"`
}

// CommandPayload 是下发给客户端的命令载荷。
type CommandPayload struct {
	// Command 命令类型，如 update_software、update_self、start、stop、restart。
	Command string `json:"command"`
	// Version 目标版本号，仅在更新类命令中使用。
	Version string `json:"version,omitempty"`
	// DownloadURL 升级文件下载地址，仅在更新类命令中使用。
	DownloadURL string `json:"download_url,omitempty"`
}
