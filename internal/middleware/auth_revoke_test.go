package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"yaofang/internal/pkg/auth"
	"yaofang/internal/pkg/errs"
)

// stateCheckerFor 构造一个以「当前权威状态」工作的 stateCheck，模拟 server 层的查库实现。
func stateCheckerFor(st *UserState) UserStateChecker {
	return func(ctx context.Context, userID int64) (*UserState, error) {
		if st == nil {
			return nil, errs.ErrUnauthorized
		}
		return st, nil
	}
}

func newAuthRouter(check UserStateChecker, mgr *auth.Manager) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/me", Auth(mgr, check), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"username": UsernameFromCtx(c),
			"name":     UserNameFromCtx(c),
			"role":     UserRoleFromCtx(c),
		})
	})
	return r
}

func getMe(t *testing.T, r *gin.Engine, token string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	return w
}

// TestAuthRejectsRevokedTokenOnPasswordChange 口令轮换后，旧 token 必须立即失效。
// 回归：JWT 无服务端吊销机制，改密后存量 token 在 TTL（默认 720h=30 天）内仍可用；
// 事件响应中管理员改密也无法处置已泄露的令牌。
func TestAuthRejectsRevokedTokenOnPasswordChange(t *testing.T) {
	mgr := auth.NewManager("test-secret-for-rbac", 720*time.Hour)

	// 签发时口令版本 3
	tok, err := mgr.Generate(7, "alice", "爱丽丝", "pharmacist", 3)
	if err != nil {
		t.Fatalf("生成 token 失败: %v", err)
	}
	// 改密后库中版本变为 4
	after := &UserState{Username: "alice", Name: "爱丽丝", Role: "pharmacist", TokenVer: 4}
	r := newAuthRouter(stateCheckerFor(after), mgr)

	if w := getMe(t, r, tok); w.Code != http.StatusUnauthorized {
		t.Fatalf("口令轮换后旧 token 应返回 401，got %d", w.Code)
	}

	// 重新登录（版本 4）后应可用
	fresh, err := mgr.Generate(7, "alice", "爱丽丝", "pharmacist", 4)
	if err != nil {
		t.Fatalf("生成新 token 失败: %v", err)
	}
	if w := getMe(t, r, fresh); w.Code != http.StatusOK {
		t.Fatalf("口令版本一致的新 token 应放行，got %d", w.Code)
	}
}

// TestAuthUsesDatabaseIdentityNotTokenClaims 角色/姓名一律以库中当前值为准。
// 回归：此前 ctx 取 token 内的旧值，改名后操作日志 username/name 继续写旧值，审计归属失真；
// 且「停用后靠 token 里的 status」也无法生效。
func TestAuthUsesDatabaseIdentityNotTokenClaims(t *testing.T) {
	mgr := auth.NewManager("test-secret-for-rbac", time.Hour)
	// token 内是旧身份：oldname / admin
	tok, err := mgr.Generate(7, "oldname", "旧姓名", "admin", 1)
	if err != nil {
		t.Fatalf("生成 token 失败: %v", err)
	}
	st := &UserState{Username: "newname", Name: "新姓名", Role: "pharmacist", TokenVer: 1}
	r := newAuthRouter(stateCheckerFor(st), mgr)

	w := getMe(t, r, tok)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if body["username"] != "newname" {
		t.Fatalf("username 应取库中当前值 newname，got %v", body["username"])
	}
	if body["name"] != "新姓名" {
		t.Fatalf("name 应取库中当前值，got %v", body["name"])
	}
	if body["role"] != "pharmacist" {
		t.Fatalf("role 应取库中当前值（降权即时生效），got %v", body["role"])
	}
}

// TestAuthRejectsWhenStateCheckerRejects stateCheck 返回错误/空状态时必须 401。
func TestAuthRejectsWhenStateCheckerRejects(t *testing.T) {
	mgr := auth.NewManager("test-secret-for-rbac", time.Hour)
	tok, err := mgr.Generate(7, "alice", "爱丽丝", "pharmacist", 1)
	if err != nil {
		t.Fatalf("生成 token 失败: %v", err)
	}
	for _, st := range []*UserState{nil} {
		r := newAuthRouter(stateCheckerFor(st), mgr)
		if w := getMe(t, r, tok); w.Code != http.StatusUnauthorized {
			t.Fatalf("账号停用/删除时应 401，got %d", w.Code)
		}
	}
}
