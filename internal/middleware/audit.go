package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuditEntry 写操作审计条目（中间件层只描述事实，不依赖 model/service）。
type AuditEntry struct {
	UserID     int64
	Username   string
	UserRole   string
	Method     string
	Path       string // 实际请求路径（含具体 ID）
	Route      string // 路由模板（如 /api/v1/prescriptions/:id），便于聚合筛选
	ResourceID int64  // 路由中的 :id 参数（无则 0）
	IP         string
	Status     int // HTTP 状态码，用于区分成功/失败操作
}

// WriteAuditor 写操作审计落库接口（由 server 层实现并注入，避免中间件依赖 service/model）。
type WriteAuditor interface {
	Audit(ctx context.Context, e AuditEntry)
}

// auditSkipPrefixes 已由业务侧显式记录审计日志的路径前缀（避免重复记录）。
var auditSkipPrefixes = []string{"/api/v1/auth/"}

// AuditWrites 统一写操作审计：对已认证用户的非只读请求（POST/PUT/PATCH/DELETE）
// 落一条操作日志，覆盖此前仅登录/改密/建用户有审计的缺口（发药、盘点、调拨、红冲、
// 结算、系统设置、用户停用等敏感操作）。
//
// 注意：日志内容仅含方法/路径/状态码，不记录请求体，避免口令等敏感数据入库。
func AuditWrites(auditor WriteAuditor) gin.HandlerFunc {
	return func(c *gin.Context) {
		if auditor == nil || isReadOnlyMethod(c.Request.Method) {
			c.Next()
			return
		}
		userID := UserIDFromCtx(c) // 未认证请求（如登录）由业务侧自行记录
		c.Next()
		if userID == 0 {
			return
		}
		path := c.Request.URL.Path
		for _, prefix := range auditSkipPrefixes {
			if strings.HasPrefix(path, prefix) {
				return
			}
		}
		entry := AuditEntry{
			UserID: userID,
			// 记录唯一登录名（姓名可能重复，不利于审计定位到具体账号）
			Username: UsernameFromCtx(c),
			UserRole: UserRoleFromCtx(c),
			Method:   c.Request.Method,
			Path:     path,
			Route:    c.FullPath(),
			IP:       c.ClientIP(),
			Status:   c.Writer.Status(),
		}
		if rawID := c.Param("id"); rawID != "" {
			if id, err := strconv.ParseInt(rawID, 10, 64); err == nil {
				entry.ResourceID = id
			}
		}
		auditor.Audit(c.Request.Context(), entry)
	}
}

// isReadOnlyMethod 只读方法（GET/HEAD/OPTIONS）不产生审计日志。
func isReadOnlyMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

// AuditAction 由 HTTP 方法映射审计动作名（create/update/delete/other）。
func AuditAction(method string) string {
	switch method {
	case http.MethodPost:
		return "create"
	case http.MethodPut, http.MethodPatch:
		return "update"
	case http.MethodDelete:
		return "delete"
	default:
		return strings.ToLower(method)
	}
}

// AuditActionFor 结合路由模板给出更可读的动作名：
// 形如 /prescriptions/:id/dispense、/charges/:id/void、/purchase-receipts/:id/complete
// 的「子资源动作」直接取末段作为动作（dispense/void/complete…），便于按敏感操作筛选审计日志；
// 其余仍按方法映射为 create/update/delete。
func AuditActionFor(method, route string) string {
	if method == http.MethodPost && strings.Contains(route, ":") {
		if idx := strings.LastIndex(route, "/"); idx >= 0 && idx+1 < len(route) {
			last := route[idx+1:]
			if last != "" && !strings.HasPrefix(last, ":") && !strings.Contains(last, "*") {
				return last
			}
		}
	}
	return AuditAction(method)
}
