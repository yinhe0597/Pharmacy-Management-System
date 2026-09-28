//go:build integration

// 双护士角色 RBAC 集成用例（docs/18 §七 验收标准）。
// 需真实 PostgreSQL（CI 提供），本地可 go vet -tags=integration 做编译校验。
package service_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"

	"yaofang/internal/config"
	"yaofang/internal/model"
	"yaofang/internal/pkg/auth"
	"yaofang/internal/server"
)

// seedRBACUsers 预置测试用户（Auth 中间件会复查签发用户状态，token 中的用户必须真实存在且启用）。
func seedRBACUsers(t *testing.T, db *gorm.DB) {
	t.Helper()
	users := []model.User{
		{ID: 1001, Username: "rbac_clinic_nurse", Name: "跟诊护士", Role: "clinic_nurse", Status: 1},
		{ID: 1002, Username: "rbac_pharmacy_nurse", Name: "药房护士", Role: "pharmacy_nurse", Status: 1},
	}
	for i := range users {
		db.Where("id = ?", users[i].ID).Assign(users[i]).FirstOrCreate(&users[i])
	}
}

func doJSON(t *testing.T, router http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// TestNurseRoleRBAC 跟诊护士可建档患者但不可领用耗材；药房护士相反。
func TestNurseRoleRBAC(t *testing.T) {
	db := setupTestDB(t)
	seedRBACUsers(t, db)
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}
	app := server.NewApp(cfg, db)
	router := app.Engine()
	mgr := auth.NewManager(cfg.Auth.JWTSecret, time.Hour)

	clinicTok, _ := mgr.Generate(1001, "clinic_nurse", "跟诊护士", "clinic_nurse", 0)
	pharmTok, _ := mgr.Generate(1002, "pharmacy_nurse", "药房护士", "pharmacy_nurse", 0)

	// 跟诊护士：可建档患者（PatientAdmin）
	w := doJSON(t, router, http.MethodPost, "/api/v1/patients", clinicTok,
		`{"card_no":"RBAC-CN-001","name":"跟诊建档"}`)
	if w.Code == http.StatusForbidden {
		t.Fatalf("跟诊护士应可建档患者，got %d", w.Code)
	}
	// 药房护士：不可建档患者（403）
	w = doJSON(t, router, http.MethodPost, "/api/v1/patients", pharmTok,
		`{"card_no":"RBAC-PN-001","name":"药房建档案"}`)
	if w.Code != http.StatusForbidden {
		t.Fatalf("药房护士建档患者应 403，got %d", w.Code)
	}
	// 药房护士：可领用耗材（Pharmacy）
	w = doJSON(t, router, http.MethodPost, "/api/v1/inventory/requisition", pharmTok,
		`{"drug_id":1,"location_id":2,"quantity":1,"reason":"补发"}`)
	if w.Code == http.StatusForbidden {
		t.Fatalf("药房护士应可领用耗材，got %d", w.Code)
	}
	// 跟诊护士：不可领用耗材（403）
	w = doJSON(t, router, http.MethodPost, "/api/v1/inventory/requisition", clinicTok,
		`{"drug_id":1,"location_id":2,"quantity":1,"reason":"补发"}`)
	if w.Code != http.StatusForbidden {
		t.Fatalf("跟诊护士领用耗材应 403，got %d", w.Code)
	}
}
