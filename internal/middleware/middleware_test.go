package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

	adminTok, _ := mgr.Generate(1, "admin", "管理员", "admin")
	pharmTok, _ := mgr.Generate(2, "pharm", "药师", "pharmacist")
	nurseTok, _ := mgr.Generate(3, "pharmacy_nurse", "药房护士", "pharmacy_nurse")

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
	tok, err := mgr.Generate(7, "doctor", "医生", "doctor")
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
