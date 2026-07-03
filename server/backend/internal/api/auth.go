package api

import (
	"example.com/appupdatemanager/server/config"
	"example.com/appupdatemanager/server/internal/middleware"
	"example.com/appupdatemanager/server/internal/store"
	"net/http"

	"github.com/gin-gonic/gin"
)

// LoginRequest 定义登录接口的请求体参数。
type LoginRequest struct {
	// Username 登录用户名。
	Username string `json:"username" binding:"required"`
	// Password 登录密码。
	Password string `json:"password" binding:"required"`
}

// Login 返回一个 gin 中间件，用于校验用户名和密码，登录成功后设置 cookie 并返回 JWT token。
func Login(cfg *config.Config, db *store.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		user, err := store.GetUserByUsername(db, req.Username)
		if err != nil || !cfg.ValidatePassword(db, req.Username, req.Password) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		token, err := middleware.GenerateJWT(user.Username, user.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
			return
		}
		c.SetCookie("token", token, 86400, "/", "", false, false)
		c.JSON(http.StatusOK, gin.H{"token": token, "username": user.Username})
	}
}

// Me 返回当前登录用户的基本信息（用户名与用户 ID）。
func Me(c *gin.Context) {
	username, _ := c.Get("username")
	userID, _ := c.Get("userID")
	c.JSON(http.StatusOK, gin.H{"username": username, "user_id": userID})
}
