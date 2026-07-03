package api

import (
	"example.com/appupdatemanager/server/internal/model"
	"example.com/appupdatemanager/server/internal/store"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ListClientVersions 返回所有客户端程序版本的列表。
func ListClientVersions(db *store.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := store.ListClientVersions(db)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, list)
	}
}

// CreateClientVersion 处理客户端程序新版本的上传请求，保存文件并将版本信息写入数据库。
func CreateClientVersion(db *store.DB, dataDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		version := c.PostForm("version")
		if version == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "version required"})
			return
		}
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "file required"})
			return
		}

		filesDir := filepath.Join(dataDir, "files", "client")
		if err := os.MkdirAll(filesDir, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		savePath := filepath.Join(filesDir, fmt.Sprintf("client_%s%s", version, filepath.Ext(file.Filename)))
		if err := c.SaveUploadedFile(file, savePath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		v := &model.ClientVersion{
			Version:  version,
			Filename: file.Filename,
			Filepath: savePath,
		}
		if err := store.CreateClientVersion(db, v); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, v)
	}
}

// SetLatestClientVersion 将指定 id 的客户端版本设置为最新版本，同时取消其他版本的最新标记。
func SetLatestClientVersion(db *store.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		if err := store.SetLatestClientVersion(db, id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}
