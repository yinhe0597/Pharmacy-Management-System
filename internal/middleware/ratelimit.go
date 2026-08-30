// Package middleware 登录限速：按「IP+用户名」滑动窗口计数，防暴力破解。
package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"yaofang/internal/pkg/errs"
)

// loginAttempt 限速请求体（仅取用户名参与限速键；宽松解析，不影响后续 Handler 绑定）。
type loginAttempt struct {
	Username string `json:"username"`
}

// LoginRateLimit 在 window 时间窗口内，同一「IP+用户名」最多允许 maxAttempts 次尝试，
// 超出返回 429（errs.ErrTooManyRequests）。内存计数，进程重启即清零（可满足单实例部署）。
func LoginRateLimit(maxAttempts int, window time.Duration) gin.HandlerFunc {
	type bucket struct {
		count int
		reset time.Time
	}
	var mu sync.Mutex
	buckets := make(map[string]*bucket)

	// 过期桶清理：避免长期运行内存无界增长（触发条件：桶数量翻倍于活跃窗口估算值）
	gcThreshold := 4 * maxAttempts

	return func(c *gin.Context) {
		var req loginAttempt
		// 读取请求体取用户名后立即还原，保证后续 Handler 的 ShouldBindJSON 正常绑定
		if raw, err := io.ReadAll(c.Request.Body); err == nil {
			_ = json.Unmarshal(raw, &req) // 宽松解析：body 异常时仍按 IP 限速
			c.Request.Body = io.NopCloser(bytes.NewReader(raw))
		}

		key := c.ClientIP() + "|" + strings.ToLower(req.Username)
		now := time.Now()

		mu.Lock()
		if len(buckets) > gcThreshold {
			for k, v := range buckets {
				if now.After(v.reset) {
					delete(buckets, k)
				}
			}
		}
		b, ok := buckets[key]
		if !ok || now.After(b.reset) {
			b = &bucket{count: 0, reset: now.Add(window)}
			buckets[key] = b
		}
		b.count++
		limited := b.count > maxAttempts
		mu.Unlock()

		if limited {
			c.AbortWithStatusJSON(errs.ErrTooManyRequests.HTTP, gin.H{
				"code":    errs.ErrTooManyRequests.Code,
				"message": errs.ErrTooManyRequests.Message,
				"data":    nil,
			})
			return
		}
		c.Next()
	}
}
