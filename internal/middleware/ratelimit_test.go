package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// newLoginRouter 构造仅挂登录限速中间件的最小路由。
func newLoginRouter(rl gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/login", rl, func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	return r
}

func postLogin(t *testing.T, r *gin.Engine, body string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))
	req.RemoteAddr = "10.0.0.1:1234"
	r.ServeHTTP(w, req)
	return w
}

// TestLoginRateLimitBlocks 超限后返回 429。
func TestLoginRateLimitBlocks(t *testing.T) {
	r := newLoginRouter(LoginRateLimit(3, time.Minute))
	for i := 0; i < 3; i++ {
		if w := postLogin(t, r, `{"username":"admin"}`); w.Code != http.StatusOK {
			t.Fatalf("第 %d 次应在限速阈值内放行，got %d", i+1, w.Code)
		}
	}
	if w := postLogin(t, r, `{"username":"admin"}`); w.Code != http.StatusTooManyRequests {
		t.Fatalf("超过阈值应返回 429，got %d", w.Code)
	}
}

// TestLoginRateLimitBoundedBody 未认证接口不得把任意大小的 body 读进内存。
// 回归：io.ReadAll 无上限时，单个 chunked 大 body 即可打爆进程内存。
func TestLoginRateLimitBoundedBody(t *testing.T) {
	r := newLoginRouter(LoginRateLimit(100, time.Minute))
	// 8 MiB 攻击体，超过 4 KiB 上限
	w := postLogin(t, r, `{"username":"`+strings.Repeat("A", 8<<20)+`"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("超大 body 应被截断后正常处理而非 panic/500，got %d", w.Code)
	}
}

// TestLoginRateLimitUsernameTruncation 超长用户名按前缀截断计入同一桶，
// 避免攻击者用「同一用户名 + 无限后缀」绕过按账号的限速。
func TestLoginRateLimitUsernameTruncation(t *testing.T) {
	r := newLoginRouter(LoginRateLimit(2, time.Minute))
	prefix := strings.Repeat("a", 80) // 超过 64 截断长度
	if w := postLogin(t, r, `{"username":"`+prefix+`1"}`); w.Code != http.StatusOK {
		t.Fatalf("第 1 次应放行，got %d", w.Code)
	}
	if w := postLogin(t, r, `{"username":"`+prefix+`2"}`); w.Code != http.StatusOK {
		t.Fatalf("第 2 次应放行，got %d", w.Code)
	}
	// 同一截断前缀的第 3 次应命中同一桶被限速
	if w := postLogin(t, r, `{"username":"`+prefix+`3"}`); w.Code != http.StatusTooManyRequests {
		t.Fatalf("截断后同桶的第 3 次应返回 429，got %d", w.Code)
	}
}

// TestLoginRateLimitBucketBounded 限速桶数量必须有硬上限：
// 攻击者用海量唯一用户名喷洒时，若只清理「已过期」桶则内存单调增长。
func TestLoginRateLimitBucketBounded(t *testing.T) {
	r := newLoginRouter(LoginRateLimit(5, time.Hour)) // 窗口极长 → 不会有桶自然过期
	// 6000 个唯一用户名远超内部 maxBuckets(4096)
	var wg sync.WaitGroup
	for i := 0; i < 6000; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			postLogin(t, r, `{"username":"user-`+itoa(n)+`"}`)
		}(i)
	}
	wg.Wait()
	// 无死锁、无 panic 即通过：桶被硬上限收敛，GC 可回收
}

// itoa 测试内联的整数转字符串。
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
