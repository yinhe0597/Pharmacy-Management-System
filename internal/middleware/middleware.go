// Package middleware 提供通用 Gin 中间件：请求ID、访问日志、panic 恢复、JWT 鉴权。
package middleware

import (
	"context"
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
	ctxKeyUserName  = "user_name" // 姓名（展示用）
	ctxKeyUsername  = "username"  // 登录名（唯一标识，审计用）
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

// Logger 结构化访问日志（健康检查探测不记录，避免探针刷屏日志/告警）。
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		if isProbePath(c.Request.URL.Path) {
			c.Next()
			return
		}
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

// isProbePath 健康检查探活路径。
func isProbePath(path string) bool {
	return path == "/healthz" || path == "/readyz"
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

// UserState 用户当前权威状态（由 server 层查库注入）。
type UserState struct {
	Username string // 当前登录名（以库为准，避免改名后审计归属失真）
	Name     string
	Role     string
	TokenVer int64 // 当前口令版本；与 claims.Ver 不一致即视为已吊销
}

// UserStateChecker 用户状态复查器（由 server 层注入仓储实现，中间件不直接依赖 DB）。
// 返回当前状态与错误；错误非 nil 或状态为 nil 表示拒绝访问（账号停用/删除/不存在）。
// 每请求复查，使 Token TTL 内的降权/改角色/改口令立即生效（不依赖签发时的旧快照）。
type UserStateChecker func(ctx context.Context, userID int64) (*UserState, error)

// Auth JWT 鉴权中间件。
// stateCheck 非 nil 时，对每个请求复查签发用户当前状态与角色（token TTL 内停用/删除/降权立即生效）。
func Auth(jwt *auth.Manager, stateCheck UserStateChecker) gin.HandlerFunc {
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
		// 身份信息一律以库中当前值为准：token 内的 role/name 可能在签发后即已过期，
		// 直接使用会让降权/改名无法即时生效、并使操作日志归属失真。
		role, username, name := claims.Role, claims.Username, claims.Name
		if stateCheck != nil {
			st, cerr := stateCheck(c.Request.Context(), claims.UserID)
			if cerr != nil || st == nil || st.TokenVer != claims.Ver {
				c.AbortWithStatusJSON(errs.ErrUnauthorized.HTTP, gin.H{
					"code": errs.ErrUnauthorized.Code, "message": errs.ErrUnauthorized.Message, "data": nil,
				})
				return
			}
			role, username, name = st.Role, st.Username, st.Name
		}
		c.Set(ctxKeyUserID, claims.UserID)
		c.Set(ctxKeyUserName, name)
		c.Set(ctxKeyUsername, username)
		c.Set(ctxKeyUserRole, role)
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
		rs, _ := role.(string) // 未挂载 Auth 时为 nil，安全断言避免 panic
		if !allowed[rs] {
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

// UserNameFromCtx 读取当前用户姓名（展示用）。
func UserNameFromCtx(c *gin.Context) string {
	if v, ok := c.Get(ctxKeyUserName); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// UsernameFromCtx 读取当前用户登录名（唯一标识，审计日志用）。
func UsernameFromCtx(c *gin.Context) string {
	if v, ok := c.Get(ctxKeyUsername); ok {
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
