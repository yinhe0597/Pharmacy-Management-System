// Package scheduler 提供定时任务：效期预警、过期锁定、库存下限预警。
package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"

	"yaofang/internal/config"
	"yaofang/internal/service"
)

// Job 定时任务集合。
type Job struct {
	cfg *config.Config
	inv *service.InventoryService
}

// New 构建定时任务。
func New(cfg *config.Config, inv *service.InventoryService) *Job {
	return &Job{cfg: cfg, inv: inv}
}

// Start 启动 cron 定时任务（enabled=false 时为空 cron）。
func (j *Job) Start() *cron.Cron {
	c := cron.New()
	if !j.cfg.Scheduler.Enabled {
		return c
	}
	today := func() time.Time { return time.Now() }

	if _, err := c.AddFunc(j.cfg.Scheduler.ExpiryWarningCron, func() {
		ctx := context.Background()
		if err := j.inv.GenerateExpiryWarnings(ctx, today(), j.cfg.Stock.DefaultExpiryWarningDays); err != nil {
			slog.Error("scheduler_expiry_warning_failed", "err", err)
		}
	}); err != nil {
		slog.Error("scheduler_add_expiry_warning_failed", "err", err)
	}

	if _, err := c.AddFunc(j.cfg.Scheduler.ExpiredLockCron, func() {
		ctx := context.Background()
		if _, err := j.inv.LockExpiredBatches(ctx, today()); err != nil {
			slog.Error("scheduler_expired_lock_failed", "err", err)
		}
	}); err != nil {
		slog.Error("scheduler_add_expired_lock_failed", "err", err)
	}

	if _, err := c.AddFunc(j.cfg.Scheduler.StockWarningCron, func() {
		ctx := context.Background()
		if err := j.inv.GenerateStockWarnings(ctx, today()); err != nil {
			slog.Error("scheduler_stock_warning_failed", "err", err)
		}
	}); err != nil {
		slog.Error("scheduler_add_stock_warning_failed", "err", err)
	}

	c.Start()
	return c
}
