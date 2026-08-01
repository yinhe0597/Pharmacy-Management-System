// Package middleware 提供通用 Gin 中间件：请求ID、访问日志、panic 恢复、JWT 鉴权。
package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"yaofang/internal/pkg/auth"
	"yaofang/internal/pkg/errs"
)

// 上下文字段常量。
const (
	ctxKeyRequestID = "request_id"
	ctxKeyUserID    = "user_id"
	ctxKeyUserName  = "user_name"
	ctxKeyUserRole  = "user_role"
)

func randomID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// RequestID 生成/透传请求 ID。
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader("X-Request-ID")
		if rid == "" {
			rid = randomID()
		}
		c.Set(ctxKeyRequestID, rid)
		c.Header("X-Request-ID", rid)
		c.Next()
	}
}

// Logger 结构化访问日志。
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		c.Next()
		slog.Info("http_request",
			"method", c.Request.Method,
			"path", path,
			"status", c.Writer.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
			"client_ip", c.ClientIP(),
			"request_id", RequestIDFromCtx(c),
		)
	}
}

// Recover 捕获 panic，返回统一 500。
func Recover() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic_recovered", "err", r, "request_id", RequestIDFromCtx(c))
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"code":    9500,
					"message": "系统异常",
					"data":    nil,
				})
			}
		}()
		c.Next()
	}
}

// Auth JWT 鉴权中间件。
func Auth(jwt *auth.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		if tokenStr == "" {
			c.AbortWithStatusJSON(errs.ErrUnauthorized.HTTP, gin.H{
				"code": errs.ErrUnauthorized.Code, "message": errs.ErrUnauthorized.Message, "data": nil,
			})
			return
		}
		claims, err := jwt.Parse(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(errs.ErrUnauthorized.HTTP, gin.H{
				"code": errs.ErrUnauthorized.Code, "message": errs.ErrUnauthorized.Message, "data": nil,
			})
			return
		}
		c.Set(ctxKeyUserID, claims.UserID)
		c.Set(ctxKeyUserName, claims.Name)
		c.Set(ctxKeyUserRole, claims.Role)
		c.Next()
	}
}

// RequireRoles 角色白名单限制。
func RequireRoles(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(c *gin.Context) {
		role, _ := c.Get(ctxKeyUserRole)
		if !allowed[role.(string)] {
			c.AbortWithStatusJSON(errs.ErrForbidden.HTTP, gin.H{
				"code": errs.ErrForbidden.Code, "message": errs.ErrForbidden.Message, "data": nil,
			})
			return
		}
		c.Next()
	}
}

// RequestIDFromCtx 读取请求 ID。
func RequestIDFromCtx(c *gin.Context) string {
	if v, ok := c.Get(ctxKeyRequestID); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// UserIDFromCtx 读取当前用户 ID。
func UserIDFromCtx(c *gin.Context) int64 {
	if v, ok := c.Get(ctxKeyUserID); ok {
		if id, ok := v.(int64); ok {
			return id
		}
	}
	return 0
}

// UserNameFromCtx 读取当前用户姓名。
func UserNameFromCtx(c *gin.Context) string {
	if v, ok := c.Get(ctxKeyUserName); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// UserRoleFromCtx 读取当前用户角色。
func UserRoleFromCtx(c *gin.Context) string {
	if v, ok := c.Get(ctxKeyUserRole); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
