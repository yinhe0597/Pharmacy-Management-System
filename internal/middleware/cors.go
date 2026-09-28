package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORS 跨域中间件（docs/16：前后端分离开发的前置条件）。
// 语义（fail-closed）：
//   - allowOrigins 为空 → 不下发任何 CORS 响应头（生产同源 Nginx 反代场景的正解）；
//   - 含 "*" → 放行任意来源（仅开发/显式选择时使用）；
//   - 配置具体域名 → 仅白名单来源放行（Origin 精确匹配）。
//
// 鉴权走 Authorization 头（非 Cookie），因此无需 Access-Control-Allow-Credentials。
func CORS(allowOrigins []string) gin.HandlerFunc {
	origins := make(map[string]bool, len(allowOrigins))
	allowAll := false
	for _, o := range allowOrigins {
		if o == "*" {
			allowAll = true
		}
		origins[o] = true
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if allowAll || origins[origin] {
			allow := origin
			if allowAll && origin == "" {
				allow = "*"
			}
			c.Header("Access-Control-Allow-Origin", allow)
			// 回显具体 Origin 时必须带 Vary: Origin：否则前置 Nginx/CDN 的响应缓存
			// 会把为 A 域计算的 ACAO 命中并返回给 B 域。
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
			c.Header("Access-Control-Max-Age", "86400")
		}
		if c.Request.Method == http.MethodOptions {
			// 预检请求直接返回
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
