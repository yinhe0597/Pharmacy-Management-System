package middleware

import "github.com/gin-gonic/gin"

// SecureHeaders 为 API 响应补充浏览器安全头。
//
// 生产入口是前端 Nginx（同源反代），HTML/CSP 由网关下发；本中间件覆盖
// 直连后端（调试、K8s 仅暴露 API、漏配网关）时的点击劫持与 MIME 嗅探风险。
// 不设置 CSP：JSON API 不是文档上下文，CSP 应加在托管 SPA 的 Nginx 上。
func SecureHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("X-XSS-Protection", "0")
		c.Next()
	}
}
