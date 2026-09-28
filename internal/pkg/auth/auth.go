// Package auth 提供 JWT 签发与解析。
package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims JWT 载荷，含当前用户关键信息。
type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	// Ver 签发时该用户的口令版本号（users.token_version）。
	// JWT 无服务端吊销机制，改密后存量 token 仍可用到 TTL 结束；
	// 每请求与库中当前值比对，口令轮换自增后旧 token 立即全部失效。
	Ver int64 `json:"ver"`
	jwt.RegisteredClaims
}

// Manager JWT 管理器。
type Manager struct {
	secret []byte
	ttl    time.Duration
}

// NewManager 创建 JWT 管理器。
func NewManager(secret string, ttl time.Duration) *Manager {
	return &Manager{secret: []byte(secret), ttl: ttl}
}

// Generate 为用户签发 token（ver 为口令版本号）。
func (m *Manager) Generate(userID int64, username, name, role string, ver int64) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   userID,
		Username: username,
		Name:     name,
		Role:     role,
		Ver:      ver,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "yaofang",
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
}

// Parse 解析并校验 token，返回 Claims。
// 锁定签名方法为 HS256 并校验 issuer：HS256 密钥若被复用于其它服务/租户，
// 对方签发的 token 不应能通过本服务鉴权。
func (m *Manager) Parse(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		return m.secret, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer("yaofang"),
	)
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
