package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// ServerProfile 表示一个服务器连接配置。
type ServerProfile struct {
	// Name 是该配置的显示名称，例如“测试服务器”、“生产服务器”。
	Name string `json:"name"`
	// Host 是服务器主机地址。
	Host string `json:"host"`
	// Port 是服务器端口。
	Port string `json:"port"`
}

// Config 保存客户端本地配置信息。
type Config struct {
	// ServerHost 与 ServerPort 保留用于兼容旧版配置文件，实际使用 Servers/SelectedServer。
	ServerHost     string          `json:"server_host"`     // ServerHost 是服务器主机地址。
	ServerPort     string          `json:"server_port"`     // ServerPort 是服务器端口。
	ClientName     string          `json:"client_name"`     // ClientName 是客户端名称，用于向服务器注册。
	ClientVersion  string          `json:"client_version"`  // ClientVersion 是客户端版本号。
	AutoStart      bool            `json:"auto_start"`      // AutoStart 表示是否开机自启动。
	Servers        []ServerProfile `json:"servers"`         // Servers 是服务器配置列表。
	SelectedServer int             `json:"selected_server"` // SelectedServer 是当前选中的服务器索引。
	MaxLogLines    int             `json:"max_log_lines"`   // MaxLogLines 是日志视图最大显示行数。
}

// Default 返回一个默认的客户端配置实例。
func Default() *Config {
	return &Config{
		ServerHost:     "127.0.0.1",
		ServerPort:     "8080",
		ClientName:     "client-001",
		ClientVersion:  "1.0.0",
		AutoStart:      false,
		Servers:        nil,
		SelectedServer: 0,
		MaxLogLines:    1000,
	}
}

// Dir 返回客户端配置目录的路径。
func Dir() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "."
	}
	return filepath.Join(configDir, "appUpdateManager")
}

// Path 返回客户端配置文件 client.json 的完整路径。
func Path() string {
	return filepath.Join(Dir(), "client.json")
}

// Load 从磁盘读取客户端配置，如果配置文件不存在则返回默认配置。
// 若读取到旧版配置（只有 ServerHost/ServerPort，没有 Servers），会自动迁移。
func Load() (*Config, error) {
	path := Path()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := Default()
			cfg.ensureServers()
			return cfg, nil
		}
		return nil, err
	}
	cfg := Default()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	cfg.migrate()
	cfg.ensureServers()
	return cfg, nil
}

// Save 将当前配置写入磁盘配置文件。
func (c *Config) Save() error {
	if err := os.MkdirAll(Dir(), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(Path(), data, 0644)
}

// CurrentServer 返回当前选中的服务器配置指针，若未选中则返回默认配置。
func (c *Config) CurrentServer() *ServerProfile {
	c.ensureServers()
	if c.SelectedServer < 0 || c.SelectedServer >= len(c.Servers) {
		c.SelectedServer = 0
	}
	return &c.Servers[c.SelectedServer]
}

// ServerURL 根据当前选中的服务器返回 HTTP 基础 URL。
func (c *Config) ServerURL() string {
	s := c.CurrentServer()
	return "http://" + s.Host + ":" + s.Port
}

// WSURL 根据当前选中的服务器返回 WebSocket 连接 URL。
func (c *Config) WSURL() string {
	s := c.CurrentServer()
	return "ws://" + s.Host + ":" + s.Port + "/ws"
}

// SelectServer 按索引选中服务器。
func (c *Config) SelectServer(index int) bool {
	c.ensureServers()
	if index < 0 || index >= len(c.Servers) {
		return false
	}
	c.SelectedServer = index
	return true
}

// AddServer 添加一个新的服务器配置。
func (c *Config) AddServer(profile ServerProfile) {
	c.Servers = append(c.Servers, profile)
	if len(c.Servers) == 1 {
		c.SelectedServer = 0
	}
}

// UpdateServer 更新指定索引的服务器配置。
func (c *Config) UpdateServer(index int, profile ServerProfile) bool {
	if index < 0 || index >= len(c.Servers) {
		return false
	}
	c.Servers[index] = profile
	return true
}

// RemoveServer 删除指定索引的服务器配置，删除后若选中索引越界则重置为 0。
func (c *Config) RemoveServer(index int) bool {
	if index < 0 || index >= len(c.Servers) {
		return false
	}
	c.Servers = append(c.Servers[:index], c.Servers[index+1:]...)
	if c.SelectedServer >= len(c.Servers) {
		c.SelectedServer = 0
	}
	if len(c.Servers) == 0 {
		c.SelectedServer = 0
	}
	return true
}

// migrate 把旧版配置文件中的 ServerHost/ServerPort 迁移到 Servers 列表。
func (c *Config) migrate() {
	if len(c.Servers) > 0 {
		return
	}
	c.Servers = []ServerProfile{
		{
			Name: "默认服务器",
			Host: c.ServerHost,
			Port: c.ServerPort,
		},
	}
	c.SelectedServer = 0
}

// ensureServers 保证 Servers 至少有一个有效配置。
func (c *Config) ensureServers() {
	if len(c.Servers) == 0 {
		c.Servers = []ServerProfile{
			{
				Name: "默认服务器",
				Host: c.ServerHost,
				Port: c.ServerPort,
			},
		}
		c.SelectedServer = 0
	}
	if c.SelectedServer < 0 || c.SelectedServer >= len(c.Servers) {
		c.SelectedServer = 0
	}
	if c.MaxLogLines <= 0 {
		c.MaxLogLines = 1000
	}
}
