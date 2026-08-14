package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORS 跨域中间件（docs/16：前后端分离开发的前置条件）。
// allowOrigins 为空或含 "*" 时放行任意来源（开发默认）；生产环境应在配置中限定具体域名。
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
