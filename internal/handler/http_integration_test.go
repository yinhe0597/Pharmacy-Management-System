//go:build integration

package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"gorm.io/gorm"

	"yaofang/internal/config"
	"yaofang/internal/model"
	"yaofang/internal/server"
)

// HTTP 层集成测试（docs/07 §5 承诺的 handler 层用例）：
// 走真实 Gin 路由 + 真实 PostgreSQL，覆盖鉴权、RBAC、限速、审计中间件、错误码映射、
// release 模式收紧（Swagger/CORS）与探针端点。
//
// 运行：go test -tags=integration ./internal/handler/

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// getenvPort 读取端口类环境变量（供 PG 多版本兼容性测试指定不同端口）。
func getenvPort(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			return p
		}
	}
	return def
}

// newTestEngine 构建一个「测试密钥 + 测试库」的真实 Gin 引擎（不监听端口）。
func newTestEngine(t *testing.T, mode string) (*gorm.DB, http.Handler) {
	t.Helper()
	// 配置校验要求强 JWT 密钥（弱密钥黑名单会拒绝 test-secret 等词根）；
	// 测试不依赖本地 configs/config.yaml
	t.Setenv("YF_AUTH_JWT_SECRET", "handler-http-integration-0f3a9c7d5b284e61")
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}
	cfg.Database.Host = getenv("YF_TEST_DB_HOST", "127.0.0.1")
	cfg.Database.Port = getenvPort("YF_TEST_DB_PORT", 5432)
	cfg.Database.User = getenv("YF_TEST_DB_USER", "yaofang")
	cfg.Database.Password = getenv("YF_TEST_DB_PASSWORD", "yaofang123")
	cfg.Database.Name = getenv("YF_TEST_DB_NAME", "yaofang")
	cfg.Server.Mode = mode
	// 不依赖本地 configs/config.yaml 的 CORS 白名单，锁定 fail-closed 语义
	cfg.Server.CORSAllowOrigins = nil

	db, err := server.OpenDB(&cfg.Database)
	if err != nil {
		t.Fatalf("连接数据库失败: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db, server.NewApp(cfg, db).Engine()
}

type envelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// doJSON 发起一次 JSON 请求并解析统一响应信封。
func doJSON(t *testing.T, h http.Handler, method, path, token string, body any) (int, envelope) {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("序列化请求体失败: %v", err)
		}
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	var env envelope
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	return w.Code, env
}

// login 以指定账号登录并返回 token。
func login(t *testing.T, h http.Handler, username, password string) string {
	t.Helper()
	status, env := doJSON(t, h, http.MethodPost, "/api/v1/auth/login", "",
		map[string]string{"username": username, "password": password})
	if status != http.StatusOK || env.Code != 0 {
		t.Fatalf("登录失败(%s): http=%d code=%d msg=%s", username, status, env.Code, env.Message)
	}
	var data struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(env.Data, &data); err != nil || data.Token == "" {
		t.Fatalf("登录响应缺少 token: %v", err)
	}
	return data.Token
}

// TestHealthEndpoints 探针与版本端点（无鉴权、必须可达）。
func TestHealthEndpoints(t *testing.T) {
	_, h := newTestEngine(t, "debug")
	for _, path := range []string{"/healthz", "/readyz", "/version"} {
		status, _ := doJSON(t, h, http.MethodGet, path, "", nil)
		if status != http.StatusOK {
			t.Fatalf("%s 期望 200，实际 %d", path, status)
		}
	}
}

// TestLoginAndAuthFailures 登录成功/失败与未认证访问。
func TestLoginAndAuthFailures(t *testing.T) {
	_, h := newTestEngine(t, "debug")

	// 错误口令 → 400（业务码非 0，不泄露账号是否存在）
	status, env := doJSON(t, h, http.MethodPost, "/api/v1/auth/login", "",
		map[string]string{"username": "admin", "password": "definitely-wrong"})
	if status != http.StatusBadRequest || env.Code == 0 {
		t.Fatalf("错误口令应返回 400 且业务码非 0，实际 http=%d code=%d", status, env.Code)
	}

	// 缺少 token → 401
	status, env = doJSON(t, h, http.MethodGet, "/api/v1/drugs", "", nil)
	if status != http.StatusUnauthorized || env.Code != 9002 {
		t.Fatalf("未认证访问应返回 401/9002，实际 http=%d code=%d", status, env.Code)
	}

	// 伪造 token → 401
	status, _ = doJSON(t, h, http.MethodGet, "/api/v1/drugs", "not-a-jwt", nil)
	if status != http.StatusUnauthorized {
		t.Fatalf("伪造 token 应返回 401，实际 %d", status)
	}

	// 正确口令 → 200 + token
	if tok := login(t, h, "admin", "admin123"); tok == "" {
		t.Fatal("登录应返回 token")
	}
}

// TestLoginRateLimit 登录限速：同一「IP+用户名」超过阈值返回 429。
func TestLoginRateLimit(t *testing.T) {
	_, h := newTestEngine(t, "debug")
	user := fmt.Sprintf("ratelimit-probe-%d", time.Now().UnixNano())

	limited := false
	for i := 0; i < 8; i++ {
		status, env := doJSON(t, h, http.MethodPost, "/api/v1/auth/login", "",
			map[string]string{"username": user, "password": "wrong-password"})
		if status == http.StatusTooManyRequests {
			if env.Code != 9029 {
				t.Fatalf("限速响应业务码应为 9029，实际 %d", env.Code)
			}
			limited = true
			break
		}
	}
	if !limited {
		t.Fatal("连续 8 次失败登录应触发 429 限速")
	}
}

// TestRBACByRole 角色矩阵：药品写操作仅 DrugAdmin 组可执行。
func TestRBACByRole(t *testing.T) {
	_, h := newTestEngine(t, "debug")

	// 药房护士（pharmacy_nurse）不属于 DrugAdmin → 403
	nurseTok := login(t, h, "pharmacy_nurse", "admin123")
	status, env := doJSON(t, h, http.MethodPost, "/api/v1/drugs", nurseTok, map[string]any{
		"code": "RBAC-NURSE-1", "generic_name": "越权测试药品", "dosage_form": "片剂",
	})
	if status != http.StatusForbidden || env.Code != 9003 {
		t.Fatalf("护士创建药品应 403/9003，实际 http=%d code=%d", status, env.Code)
	}

	// 管理员（admin ∈ DrugAdmin）→ 200
	adminTok := login(t, h, "admin", "admin123")
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%100000000)
	status, env = doJSON(t, h, http.MethodPost, "/api/v1/drugs", adminTok, map[string]any{
		"code": "RBAC-ADMIN-" + suffix, "generic_name": "HTTP 测试药品 " + suffix,
		"dosage_form": "片剂", "specification": "0.25g-" + suffix, "manufacturer": "HTTP 测试药厂 " + suffix,
		"base_unit": "盒", "split_unit": "片", "pack_size": 12,
		"is_split_allowed": true, "retail_price": 1200, "purchase_price": 800,
	})
	if status != http.StatusOK || env.Code != 0 {
		t.Fatalf("管理员创建药品应成功，实际 http=%d code=%d msg=%s", status, env.Code, env.Message)
	}
	var drug struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(env.Data, &drug); err != nil || drug.ID == 0 {
		t.Fatalf("创建药品响应缺少 id: %v", err)
	}

	// 参数校验：缺必填字段 → 400
	status, env = doJSON(t, h, http.MethodPost, "/api/v1/drugs", adminTok, map[string]any{"code": ""})
	if status != http.StatusBadRequest || env.Code == 0 {
		t.Fatalf("缺必填字段应 400，实际 http=%d code=%d", status, env.Code)
	}
}

// TestWriteAuditMiddleware 写操作审计：已认证写请求必须落 operation_logs（不含请求体）。
func TestWriteAuditMiddleware(t *testing.T) {
	db, h := newTestEngine(t, "debug")
	adminTok := login(t, h, "admin", "admin123")
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%100000000)

	status, env := doJSON(t, h, http.MethodPost, "/api/v1/drugs", adminTok, map[string]any{
		"code": "AUDIT-" + suffix, "generic_name": "审计测试药品 " + suffix, "dosage_form": "胶囊剂",
		"specification": "0.5g-" + suffix, "manufacturer": "审计测试药厂 " + suffix,
		"base_unit": "盒", "split_unit": "粒", "pack_size": 24,
		"is_split_allowed": true, "retail_price": 2400, "purchase_price": 1800,
	})
	if status != http.StatusOK || env.Code != 0 {
		t.Fatalf("创建药品失败: http=%d code=%d", status, env.Code)
	}

	var log model.OperationLog
	err := db.Where("action = ? AND resource = ?", "create", "/api/v1/drugs").
		Order("id DESC").First(&log).Error
	if err != nil {
		t.Fatalf("写操作应写入 operation_logs: %v", err)
	}
	if log.Username != "admin" || log.Method != http.MethodPost || log.Detail != "status=200" {
		t.Fatalf("审计日志内容不符: username=%s method=%s detail=%s", log.Username, log.Method, log.Detail)
	}
	// 只读请求不产生审计日志（按上面这条日志之后的记录数判断）
	before := countAuditLogs(t, db)
	doJSON(t, h, http.MethodGet, "/api/v1/drugs", adminTok, nil)
	if after := countAuditLogs(t, db); after != before {
		t.Fatalf("GET 请求不应产生审计日志: before=%d after=%d", before, after)
	}
}

func countAuditLogs(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var n int64
	if err := db.Model(&model.OperationLog{}).Count(&n).Error; err != nil {
		t.Fatalf("统计审计日志失败: %v", err)
	}
	return n
}

// TestSwaggerAndCORSGatingByMode release 模式收紧：Swagger 不注册、空白名单不下发 CORS 头。
func TestSwaggerAndCORSGatingByMode(t *testing.T) {
	// debug：Swagger 可达
	_, debugEngine := newTestEngine(t, "debug")
	status, _ := doJSON(t, debugEngine, http.MethodGet, "/swagger/index.html", "", nil)
	if status != http.StatusOK {
		t.Fatalf("debug 模式 Swagger 应可达，实际 %d", status)
	}

	// release：Swagger 404、无 CORS 头
	_, releaseEngine := newTestEngine(t, "release")
	status, _ = doJSON(t, releaseEngine, http.MethodGet, "/swagger/index.html", "", nil)
	if status != http.StatusNotFound {
		t.Fatalf("release 模式 Swagger 应 404，实际 %d", status)
	}

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	w := httptest.NewRecorder()
	releaseEngine.ServeHTTP(w, req)
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("空白名单不应下发 CORS 头，实际 %q", got)
	}
	if got := w.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("release 模式应下发 X-Content-Type-Options=nosniff，实际 %q", got)
	}
	if got := w.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("release 模式应下发 X-Frame-Options=DENY，实际 %q", got)
	}
}

// TestPaginationClamp 分页参数越界被规整（page_size 上限 200，负页码回退 1）。
func TestPaginationClamp(t *testing.T) {
	_, h := newTestEngine(t, "debug")
	adminTok := login(t, h, "admin", "admin123")

	status, env := doJSON(t, h, http.MethodGet, "/api/v1/drugs?page=0&page_size=9999", adminTok, nil)
	if status != http.StatusOK || env.Code != 0 {
		t.Fatalf("分页查询失败: http=%d code=%d", status, env.Code)
	}
	var page struct {
		Page     int `json:"page"`
		PageSize int `json:"page_size"`
	}
	if err := json.Unmarshal(env.Data, &page); err != nil {
		t.Fatalf("解析分页响应失败: %v", err)
	}
	if page.Page != 1 || page.PageSize != 200 {
		t.Fatalf("分页应被规整为 page=1 page_size=200，实际 page=%d page_size=%d", page.Page, page.PageSize)
	}
}
