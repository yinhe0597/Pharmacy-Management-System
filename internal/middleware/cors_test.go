package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// newCORSRouter 构造带 CORS 中间件的测试路由。
func newCORSRouter(allowOrigins []string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS(allowOrigins))
	r.GET("/ping", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	return r
}

func doCORS(t *testing.T, r *gin.Engine, method, origin string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, "/ping", nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	r.ServeHTTP(w, req)
	return w
}

// TestCORSFailClosedOnEmptyWhitelist 白名单为空时必须 fail-closed：
// 不下发任何 CORS 响应头（生产同源反代场景的正解，避免默认放行任意来源）。
func TestCORSFailClosedOnEmptyWhitelist(t *testing.T) {
	r := newCORSRouter(nil)
	w := doCORS(t, r, http.MethodGet, "https://evil.example.com")
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("空白名单不应下发 Access-Control-Allow-Origin，实际为 %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Methods"); got != "" {
		t.Fatalf("空白名单不应下发 Access-Control-Allow-Methods，实际为 %q", got)
	}
}

// TestCORSWhitelistMatch 白名单精确匹配：命中放行、未命中不下发头。
func TestCORSWhitelistMatch(t *testing.T) {
	r := newCORSRouter([]string{"https://pharmacy.example.com"})

	hit := doCORS(t, r, http.MethodGet, "https://pharmacy.example.com")
	if got := hit.Header().Get("Access-Control-Allow-Origin"); got != "https://pharmacy.example.com" {
		t.Fatalf("白名单命中应回显来源，实际 %q", got)
	}
	miss := doCORS(t, r, http.MethodGet, "https://pharmacy.example.com.evil.com")
	if got := miss.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("白名单未命中不应下发头，实际 %q", got)
	}
}

// TestCORSWildcard 显式配置 "*" 时放行任意来源（含无 Origin 的请求返回 "*"）。
func TestCORSWildcard(t *testing.T) {
	r := newCORSRouter([]string{"*"})

	withOrigin := doCORS(t, r, http.MethodGet, "https://any.example.com")
	if got := withOrigin.Header().Get("Access-Control-Allow-Origin"); got != "https://any.example.com" {
		t.Fatalf("通配配置应回显来源，实际 %q", got)
	}
	withoutOrigin := doCORS(t, r, http.MethodGet, "")
	if got := withoutOrigin.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("通配配置无 Origin 时应返回 *，实际 %q", got)
	}
}

// TestCORSPreflight 预检请求直接 204，不进入业务处理。
func TestCORSPreflight(t *testing.T) {
	r := newCORSRouter([]string{"https://pharmacy.example.com"})
	w := doCORS(t, r, http.MethodOptions, "https://pharmacy.example.com")
	if w.Code != http.StatusNoContent {
		t.Fatalf("预检应返回 204，实际 %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Headers"); got == "" {
		t.Fatal("预检响应应包含 Access-Control-Allow-Headers")
	}
}
