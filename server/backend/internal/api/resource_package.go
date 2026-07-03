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

// ListResourcePackages 返回所有资源包的列表。
func ListResourcePackages(db *store.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := store.ListResourcePackages(db)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, list)
	}
}

// CreateResourcePackage 处理资源包（zip 文件）上传请求，保存文件并将信息写入数据库。
func CreateResourcePackage(db *store.DB, dataDir string) gin.HandlerFunc {
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

		filesDir := filepath.Join(dataDir, "files", "resource")
		if err := os.MkdirAll(filesDir, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		savePath := filepath.Join(filesDir, fmt.Sprintf("%s_%s%s", name, version, filepath.Ext(file.Filename)))
		if err := c.SaveUploadedFile(file, savePath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		v := &model.ResourcePackage{
			Name:     name,
			Version:  version,
			Filename: file.Filename,
			Filepath: savePath,
		}
		if err := store.CreateResourcePackage(db, v); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, v)
	}
}

// DeleteResourcePackage 删除指定 id 的资源包记录及其对应的本地文件。
func DeleteResourcePackage(db *store.DB, dataDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		path, err := store.DeleteResourcePackage(db, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		_ = os.Remove(path)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// SetLatestResourcePackage 将指定 id 的资源包设置为最新，同时取消其他资源包的最新标记。
func SetLatestResourcePackage(db *store.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		if err := store.SetLatestResourcePackage(db, id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// UpdateResourcePackageName 更新指定资源包的显示名称。
func UpdateResourcePackageName(db *store.DB) gin.HandlerFunc {
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
		if err := store.UpdateResourcePackageName(db, id, req.Name); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}
