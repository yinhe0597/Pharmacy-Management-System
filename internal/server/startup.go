package server

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"yaofang/internal/model"
)

// SeedDefaultPasswordHash 种子账号默认口令（admin123）的 bcrypt 哈希，
// 与 migrations/000002_seed.up.sql、000010_seed_user_roles.up.sql、000026_refine_nurse_roles.up.sql 一致。
const SeedDefaultPasswordHash = "$2a$10$N12O3.KDfwSbtezh1s8jUuyu5.8ubDOx0L.YWUpxan9ny4Q/vACCu"

// allowDefaultPasswordsEnv 演示/本地环境豁免开关。
const allowDefaultPasswordsEnv = "YF_AUTH_ALLOW_DEFAULT_PASSWORDS"

// ListDefaultPasswordUsers 返回仍在用默认口令（admin123）且处于启用状态的账号名。
func (a *App) ListDefaultPasswordUsers(ctx context.Context) ([]string, error) {
	var usernames []string
	err := a.db.WithContext(ctx).Model(&model.User{}).
		Where("status = ? AND password_hash = ?", 1, SeedDefaultPasswordHash).
		Order("username ASC").
		Pluck("username", &usernames).Error
	if err != nil {
		return nil, fmt.Errorf("检查默认口令账号失败: %w", err)
	}
	return usernames, nil
}

// AllowDefaultPasswords 是否显式豁免默认口令检查（仅限演示/本地环境）。
func AllowDefaultPasswords() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(allowDefaultPasswordsEnv))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// GuardSeedDefaultPasswords 启动期口令安全检查：
//   - release 模式：若仍存在使用默认口令 admin123 的启用账号，直接返回错误阻止启动
//     （可用 YF_AUTH_ALLOW_DEFAULT_PASSWORDS=true 临时豁免，仅限演示环境）；
//   - 其他模式：仅告警，便于本地开发与联调（种子账号可直接登录）。
//
// 设计动机与弱 JWT 密钥拦截一致：默认凭据不得静默上线。
func (a *App) GuardSeedDefaultPasswords(ctx context.Context) error {
	users, err := a.ListDefaultPasswordUsers(ctx)
	if err != nil {
		return err
	}
	if len(users) == 0 {
		return nil
	}
	msg := fmt.Sprintf("检测到 %d 个启用账号仍在使用默认口令 admin123：%s",
		len(users), strings.Join(users, ", "))

	if a.cfg.Server.Mode == "release" && !AllowDefaultPasswords() {
		return fmt.Errorf("%s；请在登录后立即修改这些账号的口令，或临时设置 %s=true 豁免（仅限演示环境）",
			msg, allowDefaultPasswordsEnv)
	}
	slog.Warn("security_warning_default_passwords",
		"count", len(users), "usernames", strings.Join(users, ","),
		"hint", "生产环境（server.mode=release）必须修改默认口令，否则服务将拒绝启动")
	return nil
}
