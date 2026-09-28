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

// maxLoginBodyBytes 登录请求体上限。本中间件挂在鉴权之前且完全公开，
// io.ReadAll 会把整个 body 读进内存；不限长时单个 chunked 大 body 即可打爆进程。
const maxLoginBodyBytes = 4 << 10 // 4 KiB

// maxKeyUsernameLen 限速键中的用户名截断长度：用户名由客户端完全控制，
// 不截断则攻击者可用超长/海量唯一用户名撑爆限速桶内存并放大审计表。
const maxKeyUsernameLen = 64

// LoginRateLimit 在 window 时间窗口内，同一「IP+用户名」最多允许 maxAttempts 次尝试，
// 超出返回 429（errs.ErrTooManyRequests）。内存计数，进程重启即清零（可满足单实例部署）。
func LoginRateLimit(maxAttempts int, window time.Duration) gin.HandlerFunc {
	type bucket struct {
		count int
		reset time.Time
	}
	var mu sync.Mutex
	buckets := make(map[string]*bucket)

	// 过期桶清理：避免长期运行内存无界增长。
	// 上限为 maxAttempts 的固定倍数：攻击者用唯一用户名喷洒时，
	// 桶数量会单调增长，故超过硬上限后强制淘汰最早到期的一批。
	const maxBuckets = 4096

	evict := func(now time.Time) {
		if len(buckets) <= maxBuckets {
			return
		}
		// 先清已过期桶（无代价）；仍超限则按到期时间淘汰最旧的一批。
		for k, v := range buckets {
			if now.After(v.reset) {
				delete(buckets, k)
			}
		}
		if len(buckets) <= maxBuckets {
			return
		}
		overflow := len(buckets) - maxBuckets
		for k, v := range buckets {
			if overflow == 0 {
				break
			}
			// 跳过最晚到期的桶，尽量保留正在计数的活跃桶
			latest := v
			skip := false
			for _, o := range buckets {
				if o.reset.After(latest.reset) {
					skip = true
					break
				}
			}
			if skip {
				continue
			}
			delete(buckets, k)
			overflow--
		}
	}

	return func(c *gin.Context) {
		var req loginAttempt
		// 读取请求体取用户名后立即还原，保证后续 Handler 的 ShouldBindJSON 正常绑定
		// （限定 4 KiB，防未认证接口被超大 body 打爆内存）
		if raw, err := io.ReadAll(io.LimitReader(c.Request.Body, maxLoginBodyBytes)); err == nil {
			_ = json.Unmarshal(raw, &req) // 宽松解析：body 异常时仍按 IP 限速
			c.Request.Body = io.NopCloser(bytes.NewReader(raw))
		}

		name := strings.ToLower(req.Username)
		if len(name) > maxKeyUsernameLen {
			name = name[:maxKeyUsernameLen]
		}
		key := c.ClientIP() + "|" + name
		now := time.Now()

		mu.Lock()
		evict(now)
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
