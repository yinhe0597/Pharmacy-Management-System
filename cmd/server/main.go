// 药房管理系统服务入口。
package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata" // 内嵌时区数据库：-trimpath 静态二进制/精简容器也能解析 DSN TimeZone

	"yaofang/internal/config"
	"yaofang/internal/scheduler"
	"yaofang/internal/server"
	"yaofang/internal/version"
)

// @title 药房管理系统 API
// @version 1.3.0
// @description 药房进销存与处方调配后端服务
// @host localhost:8080
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("加载配置失败", "err", err)
		os.Exit(1)
	}
	logger, err := newLogger(&cfg.Log)
	if err != nil {
		slog.Error("初始化日志失败", "err", err)
		os.Exit(1)
	}
	slog.SetDefault(logger)
	slog.Info("配置加载完成", "version", version.Version, "commit", version.Commit, "build_time", version.BuildTime)

	db, err := server.OpenDB(&cfg.Database)
	if err != nil {
		slog.Error("初始化数据库失败", "err", err)
		os.Exit(1)
	}

	app := server.NewApp(cfg, db)

	// 启动期口令安全检查：默认口令（admin123）不得静默上线（release 模式直接拒绝启动）
	if err := app.GuardSeedDefaultPasswords(context.Background()); err != nil {
		slog.Error("启动安全检查未通过", "err", err)
		os.Exit(1)
	}

	router := app.Engine()

	// 定时任务
	cron := scheduler.New(cfg, app.Inventory(), app.OperationLogs(), app.Notifications()).Start()
	defer cron.Stop()

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		slog.Info("服务启动", "addr", addr, "version", version.Version, "commit", version.Commit)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("服务异常退出", "err", err)
			os.Exit(1)
		}
	}()

	// 优雅退出：先标记关闭（/healthz 转 503 摘流量），再排空存量请求。
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("正在优雅关闭")
	app.BeginShutdown()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("关闭失败", "err", err)
	}
}

// newLogger 按配置构建 slog 处理器：
//   - format=json：结构化 JSON（推荐生产，供 Loki/ELK/Promtail 直接采集解析）；
//   - format=text：人类可读文本（本地开发）；
//   - file 非空：追加写入该文件（裸机部署配合 logrotate 轮转；容器内务必留空走 stdout）。
func newLogger(cfg *config.LogConfig) (*slog.Logger, error) {
	level := slog.LevelInfo
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	opts := &slog.HandlerOptions{Level: level}

	out := io.Writer(os.Stdout)
	if cfg.File != "" {
		f, err := os.OpenFile(cfg.File, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, fmt.Errorf("打开日志文件失败: %w", err)
		}
		out = f
	}

	var h slog.Handler
	if cfg.Format == "json" {
		h = slog.NewJSONHandler(out, opts)
	} else {
		h = slog.NewTextHandler(out, opts)
	}
	return slog.New(h).With("service", "yaofang"), nil
}
