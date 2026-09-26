// Package config 加载应用配置（Viper，yaml + 环境变量 YF_ 前缀覆盖）。
package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// weakJWTSecrets 已知的弱 JWT 密钥（默认值精确匹配）。
var weakJWTSecrets = map[string]bool{
	"":          true,
	"change-me": true,
}

// weakJWTSubstrings 弱密钥词根（小写子串匹配）。覆盖连字符/下划线变体与项目默认值，
// 例如 yaofang-compose-dev-secret-change-me、yaofang-dev-secret-change-in-prod、
// CHANGE_ME/change.me/placeholder 等占位符。
var weakJWTSubstrings = []string{
	"change-me", "change_me", "change.me", "changeme",
	"placeholder", "please-change", "please_change",
	"yaofang-dev-secret", "yaofang-compose-dev-secret", "yaofang-secret",
	"your-secret", "example-secret", "sample-secret", "test-secret",
}

// isWeakJWTSecret 判断 JWT 密钥是否为弱/占位值（精确默认值或词根子串）。
func isWeakJWTSecret(s string) bool {
	if weakJWTSecrets[s] {
		return true
	}
	lower := strings.ToLower(strings.TrimSpace(s))
	if lower == "" {
		return true
	}
	for _, sub := range weakJWTSubstrings {
		if strings.Contains(lower, sub) {
			return true
		}
	}
	return false
}

// Config 应用配置根。
type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Auth      AuthConfig      `mapstructure:"auth"`
	Scheduler SchedulerConfig `mapstructure:"scheduler"`
	Stock     StockConfig     `mapstructure:"stock"`
	Log       LogConfig       `mapstructure:"log"`
}

// ServerConfig 服务配置。
type ServerConfig struct {
	Port             int      `mapstructure:"port"`
	Mode             string   `mapstructure:"mode"`
	CORSAllowOrigins []string `mapstructure:"cors_allow_origins"` // 前端跨域白名单（空=fail-closed 不下发 CORS 头；含 * 放行任意来源，生产应限定域名）
	// TrustedProxies 反向代理/负载均衡的 IP 或 CIDR 白名单，用于安全解析 X-Forwarded-For。
	// 为空表示不信任任何代理头（ClientIP 取直连地址），避免伪造 XFF 绕过登录限速。
	// 部署在 Nginx/网关之后时，须填写代理所在网段（如 172.16.0.0/12）。
	TrustedProxies []string `mapstructure:"trusted_proxies"`
}

// DatabaseConfig PostgreSQL 连接配置。
type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	Name            string        `mapstructure:"name"`
	SSLMode         string        `mapstructure:"sslmode"` // disable / require（云数据库建议 require）
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"`
}

// DSN 返回 gorm postgres DSN。
func (d DatabaseConfig) DSN() string {
	if d.SSLMode == "" {
		d.SSLMode = "disable"
	}
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=Asia/Shanghai",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode)
}

// LogConfig 日志配置（输出到 stdout/文件，供日志聚合采集）。
type LogConfig struct {
	Level  string `mapstructure:"level"`  // debug / info / warn / error
	Format string `mapstructure:"format"` // text / json（生产建议 json，便于聚合解析）
	File   string `mapstructure:"file"`   // 输出文件路径；空 = 写 stdout（容器内务必保持空）
}

// AuthConfig 鉴权配置。
type AuthConfig struct {
	JWTSecret string        `mapstructure:"jwt_secret"`
	TokenTTL  time.Duration `mapstructure:"token_ttl"`
}

// SchedulerConfig 定时任务配置。
type SchedulerConfig struct {
	Enabled           bool   `mapstructure:"enabled"`
	ExpiryWarningCron string `mapstructure:"expiry_warning_cron"`
	ExpiredLockCron   string `mapstructure:"expired_lock_cron"`
	StockWarningCron  string `mapstructure:"stock_warning_cron"`
}

// StockConfig 库存规则配置。
type StockConfig struct {
	DefaultExpiryWarningDays int `mapstructure:"default_expiry_warning_days"`
}

// Load 加载配置。优先读取 configs/config.yaml，环境变量以 YF_ 前缀覆盖（点号转下划线）。
func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("configs")
	v.AddConfigPath(".")
	v.SetEnvPrefix("YF")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// 默认值
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.mode", "debug")
	v.SetDefault("database.host", "127.0.0.1")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.user", "yaofang")
	v.SetDefault("database.password", "")
	v.SetDefault("database.name", "yaofang")
	v.SetDefault("database.max_open_conns", 20)
	v.SetDefault("database.max_idle_conns", 10)
	v.SetDefault("database.conn_max_lifetime", "1h")
	v.SetDefault("database.conn_max_idle_time", "30m")
	v.SetDefault("database.sslmode", "disable")
	v.SetDefault("auth.jwt_secret", "change-me")
	v.SetDefault("auth.token_ttl", "720h")
	v.SetDefault("scheduler.enabled", true)
	v.SetDefault("scheduler.expiry_warning_cron", "0 7 * * *")
	v.SetDefault("scheduler.expired_lock_cron", "0 0 * * *")
	v.SetDefault("scheduler.stock_warning_cron", "0 7 * * *")
	v.SetDefault("stock.default_expiry_warning_days", 90)
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "text")
	v.SetDefault("log.file", "")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("读取配置文件失败: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}
	if cfg.Database.Password == "" {
		cfg.Database.Password = os.Getenv("YF_DATABASE_PASSWORD")
	}
	// 切片类配置经环境变量注入时 Viper 不会按逗号拆分，这里显式解析（逗号/空白分隔）。
	if raw := os.Getenv("YF_SERVER_TRUSTED_PROXIES"); raw != "" {
		cfg.Server.TrustedProxies = splitList(raw)
	}
	if raw := os.Getenv("YF_SERVER_CORS_ALLOW_ORIGINS"); raw != "" {
		cfg.Server.CORSAllowOrigins = splitList(raw)
	}
	cfg.normalize()
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// splitList 按逗号/空白切分环境变量注入的列表，去除空项。
func splitList(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ' ' || r == '\t' || r == '\n' })
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// validate 校验安全关键配置。与主线 59c3c71 的启动校验合并、统一收敛到配置层：
// JWT 密钥为空/默认/占位值或长度不足 32 位时一律拒绝启动（HS256 对称签名，弱钥即可离线伪造任意 token）。
func (c *Config) validate() error {
	if isWeakJWTSecret(c.Auth.JWTSecret) {
		return fmt.Errorf("auth.jwt_secret 为默认/占位值：请在 configs/config.yaml 设置 ≥32 位随机密钥，或使用环境变量 YF_AUTH_JWT_SECRET（建议 openssl rand -base64 48）")
	}
	if len(c.Auth.JWTSecret) < 32 {
		return fmt.Errorf("auth.jwt_secret 强度不足：当前 %d 字符 < 32（HS256 对称签名，密钥泄露即可离线伪造任意 token）", len(c.Auth.JWTSecret))
	}
	return nil
}

// normalize 对关键配置做钳制/兜底，避免非法值导致运行期异常（连接池、日志）。
func (c *Config) normalize() {
	// 连接池：非法值回退默认，并保证 idle ≤ open 约束成立。
	if c.Database.MaxOpenConns < 1 {
		c.Database.MaxOpenConns = 20
	}
	if c.Database.MaxIdleConns < 0 {
		c.Database.MaxIdleConns = 0
	}
	if c.Database.MaxIdleConns > c.Database.MaxOpenConns {
		c.Database.MaxIdleConns = c.Database.MaxOpenConns
	}
	if c.Database.ConnMaxLifetime <= 0 {
		c.Database.ConnMaxLifetime = time.Hour
	}
	if c.Database.ConnMaxIdleTime <= 0 {
		c.Database.ConnMaxIdleTime = 30 * time.Minute
	}
	if c.Database.SSLMode == "" {
		c.Database.SSLMode = "disable"
	}
	// 日志：非法值回退默认。
	switch c.Log.Level {
	case "debug", "info", "warn", "error":
	default:
		c.Log.Level = "info"
	}
	switch c.Log.Format {
	case "text", "json":
	default:
		c.Log.Format = "text"
	}
}
