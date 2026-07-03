package middleware

import (
	"example.com/appupdatemanager/server/config"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// jwtSecret 是用于签发与校验 JWT 的 HMAC 密钥，生产环境应当替换为更安全的随机密钥。
const jwtSecret = "appUpdateManager-secret-key-change-in-production"

// Auth 返回一个 gin 中间件，用于校验请求中的 JWT token 或 session cookie，并将用户信息写入上下文。
func Auth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		claims, err := ParseJWT(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Set("username", claims.Username)
		c.Set("userID", claims.UserID)
		c.Next()
	}
}

// extractToken 从请求的 Authorization 头或 token cookie 中提取 JWT token。
func extractToken(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	cookie, err := c.Cookie("token")
	if err == nil {
		return cookie
	}
	return ""
}

// TokenClaims 定义 JWT 中携带的用户声明信息。
type TokenClaims struct {
	// Username 登录用户名。
	Username string `json:"username"`
	// UserID 用户在数据库中的唯一标识。
	UserID   int64     `json:"user_id"`
	// Exp token 的过期时间戳（Unix 秒）。
	Exp      int64     `json:"exp"`
}

// GenerateJWT 使用 HMAC-SHA256 为指定用户生成一个有效期为 24 小时的 JWT token。
func GenerateJWT(username string, userID int64) (string, error) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	claims := TokenClaims{
		Username: username,
		UserID:   userID,
		Exp:      time.Now().Add(24 * time.Hour).Unix(),
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(claimsJSON)
	sig := sign(header + "." + payload)
	return header + "." + payload + "." + sig, nil
}

// ParseJWT 解析并校验 JWT token 的格式、签名与过期时间，返回解析后的声明信息。
func ParseJWT(token string) (*TokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid token format")
	}
	expectedSig := sign(parts[0] + "." + parts[1])
	if !hmac.Equal([]byte(expectedSig), []byte(parts[2])) {
		return nil, errors.New("invalid signature")
	}
	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	var claims TokenClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, err
	}
	if time.Now().Unix() > claims.Exp {
		return nil, errors.New("token expired")
	}
	return &claims, nil
}

// sign 使用 jwtSecret 对输入数据进行 HMAC-SHA256 签名，并返回 Base64 URL 编码结果。
func sign(data string) string {
	h := hmac.New(sha256.New, []byte(jwtSecret))
	h.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}
