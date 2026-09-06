package config

import (
	"strings"
	"testing"
	"time"
)

func TestIsWeakJWTSecret(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"空串", "", true},
		{"默认值", "change-me", true},
		{"占位 CHANGE_ME", "CHANGE_ME", true},
		{"样例占位", "CHANGE_ME_32chars+", true},
		{"k8s 模板占位", "CHANGE_ME_32chars_minimum_random_string", true},
		{"小写占位变体", "please_change_me_now", true},
		{"强密钥", "k8sJ3@9fNz!qL2mX7vR5tY8wB1cD6eG4", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isWeakJWTSecret(c.in); got != c.want {
				t.Fatalf("isWeakJWTSecret(%q) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

// TestValidateRejectsWeakSecret 弱/占位 JWT 密钥在任何模式下均应拒绝启动。
func TestValidateRejectsWeakSecret(t *testing.T) {
	for _, mode := range []string{"debug", "release"} {
		for _, sec := range []string{"", "change-me", "CHANGE_ME", "CHANGE_ME_32chars+"} {
			c := &Config{Server: ServerConfig{Mode: mode}, Auth: AuthConfig{JWTSecret: sec}}
			if err := c.validate(); err == nil {
				t.Fatalf("mode=%s secret=%q 应拒绝启动", mode, sec)
			}
		}
	}
}

// TestValidateRejectsShortSecret 长度不足 32 的密钥在任何模式下均应拒绝启动。
func TestValidateRejectsShortSecret(t *testing.T) {
	c := &Config{Auth: AuthConfig{JWTSecret: "short-key-12345"}}
	if err := c.validate(); err == nil {
		t.Fatal("长度 <32 的密钥应拒绝启动")
	}
}

func TestNormalizeFallbacks(t *testing.T) {
	c := &Config{}
	c.Database.MaxOpenConns = 0
	c.Database.MaxIdleConns = -1
	c.Database.ConnMaxLifetime = 0
	c.Database.ConnMaxIdleTime = -time.Second
	c.Log.Level = "verbose"
	c.Log.Format = "xml"
	c.normalize()

	if c.Database.MaxOpenConns != 20 {
		t.Errorf("MaxOpenConns = %d, want 20", c.Database.MaxOpenConns)
	}
	if c.Database.MaxIdleConns != 0 {
		t.Errorf("MaxIdleConns = %d, want 0", c.Database.MaxIdleConns)
	}
	if c.Database.ConnMaxLifetime != time.Hour {
		t.Errorf("ConnMaxLifetime = %v, want 1h", c.Database.ConnMaxLifetime)
	}
	if c.Database.ConnMaxIdleTime != 30*time.Minute {
		t.Errorf("ConnMaxIdleTime = %v, want 30m", c.Database.ConnMaxIdleTime)
	}
	if c.Database.SSLMode != "disable" {
		t.Errorf("SSLMode = %q, want disable", c.Database.SSLMode)
	}
	if c.Log.Level != "info" {
		t.Errorf("Log.Level = %q, want info", c.Log.Level)
	}
	if c.Log.Format != "text" {
		t.Errorf("Log.Format = %q, want text", c.Log.Format)
	}
}

// TestNormalizeIdleCappedByOpen 空闲连接数不得超过最大打开数。
func TestNormalizeIdleCappedByOpen(t *testing.T) {
	c := &Config{}
	c.Database.MaxOpenConns = 5
	c.Database.MaxIdleConns = 50
	c.Database.ConnMaxLifetime = time.Hour
	c.Database.ConnMaxIdleTime = time.Minute
	c.normalize()
	if c.Database.MaxIdleConns != 5 {
		t.Fatalf("MaxIdleConns = %d, want 5（被 MaxOpenConns 钳制）", c.Database.MaxIdleConns)
	}
}

func TestDSNSSLMode(t *testing.T) {
	d := DatabaseConfig{Host: "h", Port: 5432, User: "u", Password: "p", Name: "n"}
	if !strings.Contains(d.DSN(), "sslmode=disable") {
		t.Error("未配置 sslmode 时 DSN 应默认 disable")
	}
	d.SSLMode = "require"
	if !strings.Contains(d.DSN(), "sslmode=require") {
		t.Error("sslmode=require 应透传至 DSN")
	}
}
