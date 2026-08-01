// 药房管理系统服务入口。
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"yaofang/internal/config"
	"yaofang/internal/scheduler"
	"yaofang/internal/server"
)

// @title 药房管理系统 API
// @version 1.0
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
	slog.SetDefault(newLogger())

	db, err := server.OpenDB(&cfg.Database)
	if err != nil {
		slog.Error("初始化数据库失败", "err", err)
		os.Exit(1)
	}

	app := server.NewApp(cfg, db)
	router := app.Engine()

	// 定时任务
	cron := scheduler.New(cfg, app.Inventory()).Start()
	defer cron.Stop()

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		slog.Info("服务启动", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("服务异常退出", "err", err)
			os.Exit(1)
		}
	}()

	// 优雅退出
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("正在优雅关闭")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("关闭失败", "err", err)
	}
}

func newLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
}
