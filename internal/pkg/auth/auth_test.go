package auth

import (
	"testing"
	"time"
)

func TestGenerateAndParse(t *testing.T) {
	m := NewManager("test-secret-please-change-32-characters", time.Hour)
	token, err := m.Generate(42, "alice", "爱丽丝", "pharmacist")
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
}

func TestParseRejectsWrongSecret(t *testing.T) {
	signer := NewManager("secret-number-one-32-characters-long", time.Hour)
	token, err := signer.Generate(1, "u", "n", "admin")
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
	token, err := m.Generate(1, "u", "n", "admin")
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
