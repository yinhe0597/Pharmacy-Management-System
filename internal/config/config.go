// Package config 加载应用配置（Viper，yaml + 环境变量 YF_ 前缀覆盖）。
package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

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
	CORSAllowOrigins []string `mapstructure:"cors_allow_origins"` // 前端跨域白名单（空或 * 放行任意，生产限定域名）
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
	cfg.normalize()
	return &cfg, nil
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
