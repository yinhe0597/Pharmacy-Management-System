package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"yaofang/internal/pkg/auth"
)

func newTestRouter() (*gin.Engine, *auth.Manager) {
	gin.SetMode(gin.TestMode)
	mgr := auth.NewManager("test-secret-for-rbac", time.Hour)
	r := gin.New()
	// 受保护路由：仅 admin/pharmacist 可访问
	r.GET("/protected", Auth(mgr, nil), RequireRoles("admin", "pharmacist"),
		func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	return r, mgr
}

func doRequest(t *testing.T, r *gin.Engine, token string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	r.ServeHTTP(w, req)
	return w
}

// TestRequireRoles 角色白名单中间件：允许的角色放行，其余 403。
func TestRequireRoles(t *testing.T) {
	r, mgr := newTestRouter()

	adminTok, _ := mgr.Generate(1, "admin", "管理员", "admin", 0)
	pharmTok, _ := mgr.Generate(2, "pharm", "药师", "pharmacist", 0)
	nurseTok, _ := mgr.Generate(3, "pharmacy_nurse", "药房护士", "pharmacy_nurse", 0)

	cases := []struct {
		name     string
		token    string
		wantCode int
	}{
		{"admin 放行", adminTok, http.StatusOK},
		{"pharmacist 放行", pharmTok, http.StatusOK},
		{"nurse 拒绝", nurseTok, http.StatusForbidden},
		{"无 token 拒绝", "", http.StatusUnauthorized},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := doRequest(t, r, c.token)
			if w.Code != c.wantCode {
				t.Fatalf("status = %d, want %d", w.Code, c.wantCode)
			}
		})
	}
}

// TestAuthInjectsRole 鉴权中间件注入用户角色到上下文（供 RequireRoles 使用）。
func TestAuthInjectsRole(t *testing.T) {
	r, mgr := newTestRouter()
	tok, err := mgr.Generate(7, "doctor", "医生", "doctor", 0)
	if err != nil {
		t.Fatalf("生成 token 失败: %v", err)
	}
	// doctor 不在白名单 → 403，但 Auth 已通过（非 401）
	w := doRequest(t, r, tok)
	if w.Code != http.StatusForbidden {
		t.Fatalf("doctor 应返回 403（鉴权通过但角色拒绝），got %d", w.Code)
	}
	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["code"] != float64(9003) {
		t.Fatalf("错误码应为 9003，got %v", body["code"])
	}
}

func TestIsProbePath(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"/healthz", true},
		{"/readyz", true},
		{"/version", false},
		{"/api/v1/auth/login", false},
		{"/healthz/sub", false},
	}
	for _, c := range cases {
		if got := isProbePath(c.path); got != c.want {
			t.Errorf("isProbePath(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

// TestLoggerSkipsProbes 探针路径不产生访问日志，业务路径正常记录。
func TestLoggerSkipsProbes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	defer slog.SetDefault(prev)

	r := gin.New()
	r.Use(Logger())
	h := func(c *gin.Context) { c.Status(http.StatusOK) }
	r.GET("/healthz", h)
	r.GET("/readyz", h)
	r.GET("/api/v1/ping", h)

	for _, p := range []string{"/healthz", "/readyz", "/api/v1/ping"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, p, nil))
		if w.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want 200", p, w.Code)
		}
	}
	out := buf.String()
	if strings.Contains(out, "/healthz") || strings.Contains(out, "/readyz") {
		t.Errorf("探针请求不应出现在访问日志中：\n%s", out)
	}
	if !strings.Contains(out, "/api/v1/ping") {
		t.Errorf("业务请求应记录访问日志：\n%s", out)
	}
}

// TestRequireRolesWithoutAuth 未挂载 Auth 时安全断言不 panic，直接 403。
func TestRequireRolesWithoutAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/protected", RequireRoles("admin"),
		func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/protected", nil))
	if w.Code != http.StatusForbidden {
		t.Fatalf("无 Auth 上下文应 403，got %d", w.Code)
	}
}

// TestAuthInjectsUsername 鉴权中间件同时注入登录名（审计）与姓名（展示）。
func TestAuthInjectsUsername(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mgr := auth.NewManager("test-secret-for-rbac", time.Hour)
	r := gin.New()
	r.GET("/me", Auth(mgr, nil), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"username": UsernameFromCtx(c),
			"name":     UserNameFromCtx(c),
		})
	})
	tok, err := mgr.Generate(1, "alice", "爱丽丝", "admin", 0)
	if err != nil {
		t.Fatalf("生成 token 失败: %v", err)
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if body["username"] != "alice" {
		t.Fatalf("username = %v, want alice", body["username"])
	}
	if body["name"] != "爱丽丝" {
		t.Fatalf("name = %v, want 爱丽丝", body["name"])
	}
}

// TestSecureHeaders API 响应必须带 nosniff / DENY / no-referrer。
func TestSecureHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SecureHeaders())
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if got := w.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q", got)
	}
	if got := w.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("X-Frame-Options = %q", got)
	}
	if got := w.Header().Get("Referrer-Policy"); got != "no-referrer" {
		t.Fatalf("Referrer-Policy = %q", got)
	}
}
