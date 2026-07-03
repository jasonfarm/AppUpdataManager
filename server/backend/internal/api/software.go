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

// ListSoftware 返回所有被管理软件版本的列表。
func ListSoftware(db *store.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := store.ListSoftwareVersions(db)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, list)
	}
}

// CreateSoftware 处理被管理软件新版本的上传请求，保存文件并将版本信息写入数据库。
func CreateSoftware(db *store.DB, dataDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.PostForm("name")
		version := c.PostForm("version")
		if name == "" || version == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name and version required"})
			return
		}
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "file required"})
			return
		}

		filesDir := filepath.Join(dataDir, "files", "software")
		if err := os.MkdirAll(filesDir, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		savePath := filepath.Join(filesDir, fmt.Sprintf("%s_%s%s", name, version, filepath.Ext(file.Filename)))
		if err := c.SaveUploadedFile(file, savePath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		v := &model.SoftwareVersion{
			Name:     name,
			Version:  version,
			Filename: file.Filename,
			Filepath: savePath,
		}
		if err := store.CreateSoftwareVersion(db, v); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, v)
	}
}

// DeleteSoftware 删除指定 id 的软件版本记录及其对应的本地文件。
func DeleteSoftware(db *store.DB, dataDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		path, err := store.DeleteSoftwareVersion(db, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		_ = os.Remove(path)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// SetLatestSoftware 将指定 id 的软件版本设置为最新版本，同时取消其他版本的最新标记。
func SetLatestSoftware(db *store.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		if err := store.SetLatestSoftwareVersion(db, id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// UpdateSoftwareName 更新指定 id 软件版本的显示名称。
func UpdateSoftwareName(db *store.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		var req struct {
			// Name 新的软件显示名称。
			Name string `json:"name" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := store.UpdateSoftwareName(db, id, req.Name); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}
