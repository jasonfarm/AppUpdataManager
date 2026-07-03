package api

import (
	"encoding/json"
	"example.com/appupdatemanager/server/internal/model"
	"example.com/appupdatemanager/server/internal/store"
	"example.com/appupdatemanager/server/internal/ws"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ListClients 返回所有已注册客户端的列表。
func ListClients(db *store.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := store.ListClients(db)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, list)
	}
}

// GetClient 根据 URL 中的 id 返回单个客户端的详细信息。
func GetClient(db *store.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		client, err := store.GetClient(db, id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "client not found"})
			return
		}
		c.JSON(http.StatusOK, client)
	}
}

// UpdateClientSoftware 向指定客户端下发软件更新命令，可指定版本号，否则使用最新版本。
func UpdateClientSoftware(hub *ws.Hub, db *store.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		var req struct {
			// Version 目标软件版本号，为空时表示使用当前最新版本。
			Version string `json:"version"`
		}
		c.ShouldBindJSON(&req)

		client, err := store.GetClient(db, id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "client not found"})
			return
		}

		var version string
		var downloadURL string
		if req.Version != "" {
			// find specific version
			list, err := store.ListSoftwareVersions(db)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			found := false
			for _, v := range list {
				if v.Version == req.Version {
					version = v.Version
					downloadURL = fmt.Sprintf("/files/software/%s", filepathBase(v.Filepath))
					found = true
					break
				}
			}
			if !found {
				c.JSON(http.StatusBadRequest, gin.H{"error": "version not found"})
				return
			}
		} else {
			latest, err := store.GetLatestSoftwareVersion(db)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "no latest software version set"})
				return
			}
			version = latest.Version
			downloadURL = fmt.Sprintf("/files/software/%s", filepathBase(latest.Filepath))
		}

		payload := model.CommandPayload{
			Command:     "update_software",
			Version:     version,
			DownloadURL: downloadURL,
		}
		payloadBytes, _ := json.Marshal(payload)
		cmd := &model.ClientCommand{
			ClientID:    client.ID,
			CommandType: payload.Command,
			Payload:     string(payloadBytes),
			Status:      "pending",
		}
		if err := store.CreateCommand(db, cmd); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		hub.SendToClient(client.Name, payload)
		c.JSON(http.StatusOK, gin.H{"ok": true, "version": version})
	}
}

// UpdateClientResource 向指定客户端下发资源包更新命令，使用当前最新资源包。
func UpdateClientResource(hub *ws.Hub, db *store.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		client, err := store.GetClient(db, id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "client not found"})
			return
		}

		latest, err := store.GetLatestResourcePackage(db)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no latest resource package set"})
			return
		}

		payload := model.CommandPayload{
			Command:     "update_resource",
			Version:     latest.Version,
			DownloadURL: fmt.Sprintf("/files/resource/%s", filepathBase(latest.Filepath)),
		}
		payloadBytes, _ := json.Marshal(payload)
		cmd := &model.ClientCommand{
			ClientID:    client.ID,
			CommandType: payload.Command,
			Payload:     string(payloadBytes),
			Status:      "pending",
		}
		if err := store.CreateCommand(db, cmd); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		hub.SendToClient(client.Name, payload)
		c.JSON(http.StatusOK, gin.H{"ok": true, "version": latest.Version})
	}
}

// UpdateClientName 更新指定客户端的显示名称，并向在线客户端下发名称更新命令。
func UpdateClientName(hub *ws.Hub, db *store.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		var req struct {
			Name string `json:"name" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		client, err := store.GetClient(db, id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "client not found"})
			return
		}

		// 下发名称更新命令时使用旧名称路由，客户端收到后更新本地配置。
		payload := model.CommandPayload{
			Command: "update_name",
			Version: req.Name,
		}
		payloadBytes, _ := json.Marshal(payload)
		cmd := &model.ClientCommand{
			ClientID:    client.ID,
			CommandType: payload.Command,
			Payload:     string(payloadBytes),
			Status:      "pending",
		}
		if err := store.CreateCommand(db, cmd); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		hub.SendToClient(client.Name, payload)

		if err := store.UpdateClientName(db, id, req.Name); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// DeleteClient 删除指定 id 的客户端记录。
func DeleteClient(db *store.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		if err := store.DeleteClient(db, id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// UpdateClientSelf 向指定客户端下发客户端自身升级命令，使用当前最新客户端版本。
func UpdateClientSelf(hub *ws.Hub, db *store.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		client, err := store.GetClient(db, id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "client not found"})
			return
		}
		latest, err := store.GetLatestClientVersion(db)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no latest client version set"})
			return
		}
		payload := model.CommandPayload{
			Command:     "update_self",
			Version:     latest.Version,
			DownloadURL: fmt.Sprintf("/files/client/%s", filepathBase(latest.Filepath)),
		}
		payloadBytes, _ := json.Marshal(payload)
		cmd := &model.ClientCommand{
			ClientID:    client.ID,
			CommandType: payload.Command,
			Payload:     string(payloadBytes),
			Status:      "pending",
		}
		if err := store.CreateCommand(db, cmd); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		hub.SendToClient(client.Name, payload)
		c.JSON(http.StatusOK, gin.H{"ok": true, "version": latest.Version})
	}
}

// StartClientSoftware 向指定客户端下发启动被管理软件的命令。
func StartClientSoftware(hub *ws.Hub, db *store.DB) gin.HandlerFunc {
	return sendSimpleCommand(hub, db, "start")
}

// StopClientSoftware 向指定客户端下发停止被管理软件的命令。
func StopClientSoftware(hub *ws.Hub, db *store.DB) gin.HandlerFunc {
	return sendSimpleCommand(hub, db, "stop")
}

// RestartClientSoftware 向指定客户端下发重启被管理软件的命令。
func RestartClientSoftware(hub *ws.Hub, db *store.DB) gin.HandlerFunc {
	return sendSimpleCommand(hub, db, "restart")
}

// sendSimpleCommand 构造并下发简单的 start/stop/restart 控制命令，并将其状态保存为 pending。
func sendSimpleCommand(hub *ws.Hub, db *store.DB, command string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		client, err := store.GetClient(db, id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "client not found"})
			return
		}
		payload := model.CommandPayload{Command: command}
		payloadBytes, _ := json.Marshal(payload)
		cmd := &model.ClientCommand{
			ClientID:    client.ID,
			CommandType: command,
			Payload:     string(payloadBytes),
			Status:      "pending",
		}
		if err := store.CreateCommand(db, cmd); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		hub.SendToClient(client.Name, payload)
		c.JSON(http.StatusOK, gin.H{"ok": true, "command": command})
	}
}

// filepathBase 是一个小型辅助函数，返回路径中的最后一个文件名组件，避免导入 path/filepath 包。
func filepathBase(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[i+1:]
		}
	}
	return path
}
