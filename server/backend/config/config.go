package config

import (
	"example.com/appupdatemanager/server/internal/model"
	"example.com/appupdatemanager/server/internal/store"
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// Config 保存应用级别的配置信息，包含从账户文件加载的用户列表。
type Config struct {
	// Users 是从 accounts.txt 解析出的账户列表。
	Users []model.User
}

// Load 从指定路径的账户文件读取配置，文件格式为每行 username:password，支持 # 注释与空行。
func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open accounts file: %w", err)
	}
	defer f.Close()

	cfg := &Config{}
	scanner := bufio.NewScanner(f)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid account line %d: %s", lineNo, line)
		}
		cfg.Users = append(cfg.Users, model.User{
			Username: strings.TrimSpace(parts[0]),
			Password: strings.TrimSpace(parts[1]),
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// SyncUsers 将配置文件中的账户同步到数据库，使用 bcrypt 对明文密码进行哈希后保存。
func (c *Config) SyncUsers(db *store.DB) error {
	for _, u := range c.Users {
		hash, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		u.PasswordHash = string(hash)
		u.Password = ""
		if err := store.UpsertUser(db, &u); err != nil {
			return err
		}
	}
	return nil
}

// ValidatePassword 使用数据库中的密码哈希验证用户名与密码是否匹配。
func (c *Config) ValidatePassword(db *store.DB, username, password string) bool {
	user, err := store.GetUserByUsername(db, username)
	if err != nil {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) == nil
}
