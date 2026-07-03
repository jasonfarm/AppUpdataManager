package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config 保存客户端本地配置信息。
type Config struct {
	ServerHost    string `json:"server_host"`    // ServerHost 是服务器主机地址。
	ServerPort    string `json:"server_port"`    // ServerPort 是服务器端口。
	ClientName    string `json:"client_name"`    // ClientName 是客户端名称，用于向服务器注册。
	ClientVersion string `json:"client_version"` // ClientVersion 是客户端版本号。
	AutoStart     bool   `json:"auto_start"`     // AutoStart 表示是否开机自启动。
}

// Default 返回一个默认的客户端配置实例。
func Default() *Config {
	return &Config{
		ServerHost:    "127.0.0.1",
		ServerPort:    "8080",
		ClientName:    "client-001",
		ClientVersion: "1.0.0",
		AutoStart:     false,
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
func Load() (*Config, error) {
	path := Path()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Default(), nil
		}
		return nil, err
	}
	cfg := Default()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
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

// ServerURL 根据服务器主机和端口组合返回 HTTP 基础 URL。
func (c *Config) ServerURL() string {
	return "http://" + c.ServerHost + ":" + c.ServerPort
}

// WSURL 根据服务器主机和端口组合返回 WebSocket 连接 URL。
func (c *Config) WSURL() string {
	return "ws://" + c.ServerHost + ":" + c.ServerPort + "/ws"
}
