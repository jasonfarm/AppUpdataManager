package service

import (
	"example.com/appupdatemanager/server/internal/model"
	"example.com/appupdatemanager/server/internal/store"
	"encoding/json"
)

// BuildCommandPayload 根据命令类型、目标版本与下载地址构建客户端命令的 JSON 载荷。
func BuildCommandPayload(command, version, downloadURL string) (string, error) {
	payload := model.CommandPayload{
		Command:     command,
		Version:     version,
		DownloadURL: downloadURL,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// SavePendingCommand 将一条待执行的客户端命令持久化到数据库。
func SavePendingCommand(db *store.DB, clientID int64, commandType, payload string) error {
	cmd := &model.ClientCommand{
		ClientID:    clientID,
		CommandType: commandType,
		Payload:     payload,
		Status:      "pending",
	}
	return store.CreateCommand(db, cmd)
}
