package auth

import (
	"testing"
	"time"
)

func TestGenerateAndParse(t *testing.T) {
	m := NewManager("test-secret-please-change-32-characters", time.Hour)
	token, err := m.Generate(42, "alice", "爱丽丝", "pharmacist", 7)
	if err != nil {
		t.Fatalf("Generate 失败: %v", err)
	}
	claims, err := m.Parse(token)
	if err != nil {
		t.Fatalf("Parse 失败: %v", err)
	}
	if claims.UserID != 42 || claims.Username != "alice" || claims.Name != "爱丽丝" || claims.Role != "pharmacist" {
		t.Fatalf("Claims 不匹配: %+v", claims)
	}
	if claims.Ver != 7 {
		t.Fatalf("口令版本号应原样签入 claims，got %d want 7", claims.Ver)
	}
}

func TestParseRejectsWrongSecret(t *testing.T) {
	signer := NewManager("secret-number-one-32-characters-long", time.Hour)
	token, err := signer.Generate(1, "u", "n", "admin", 0)
	if err != nil {
		t.Fatalf("Generate 失败: %v", err)
	}
	verifier := NewManager("secret-number-two-32-characters-long", time.Hour)
	if _, err := verifier.Parse(token); err == nil {
		t.Fatal("使用不同密钥应解析失败")
	}
}

func TestParseRejectsExpired(t *testing.T) {
	m := NewManager("test-secret-please-change-32-characters", -time.Minute)
	token, err := m.Generate(1, "u", "n", "admin", 0)
	if err != nil {
		t.Fatalf("Generate 失败: %v", err)
	}
	if _, err := m.Parse(token); err == nil {
		t.Fatal("过期 token 应解析失败")
	}
}

func TestParseRejectsGarbage(t *testing.T) {
	m := NewManager("test-secret-please-change-32-characters", time.Hour)
	if _, err := m.Parse("not-a-jwt"); err == nil {
		t.Fatal("非法 token 应解析失败")
	}
}

// TestParseRejectsAlgNone alg=none 未签名 token 必须被拒。
func TestParseRejectsAlgNone(t *testing.T) {
	m := NewManager("test-secret-please-change-32-characters", time.Hour)
	// 手工构造 header.alg=none 的未签名 JWT
	raw := "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0." +
		"eyJ1c2VyX2lkIjoxLCJleHAiOjk5OTk5OTk5OTl9."
	if _, err := m.Parse(raw); err == nil {
		t.Fatal("alg=none 的未签名 token 应解析失败")
	}
}
