// Package scheduler 提供定时任务：效期预警、过期锁定、库存下限预警、日志归档、预警通知。
package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"

	"yaofang/internal/config"
	"yaofang/internal/model"
	"yaofang/internal/service"
)

// Job 定时任务集合。
type Job struct {
	cfg      *config.Config
	inv      *service.InventoryService
	logSvc   *service.OperationLogService
	notifSvc *service.NotificationService
}

// New 构建定时任务。
func New(cfg *config.Config, inv *service.InventoryService, logSvc *service.OperationLogService, notifSvc *service.NotificationService) *Job {
	return &Job{cfg: cfg, inv: inv, logSvc: logSvc, notifSvc: notifSvc}
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
		n, err := j.inv.LockExpiredBatches(ctx, today())
		if err != nil {
			slog.Error("scheduler_expired_lock_failed", "err", err)
			return
		}
		// 锁定批次需立即处置：critical 广播（000036）
		if n > 0 && j.notifSvc != nil {
			msg := fmt.Sprintf("今日锁定过期批次 %d 个，该批次已禁止发药，请及时处理", n)
			if err := j.notifSvc.Broadcast(ctx, "过期批次已锁定", msg, model.NotifyLevelCritical, "inventory", nil); err != nil {
				slog.Error("scheduler_lock_notify_failed", "err", err)
			}
		}
	}); err != nil {
		slog.Error("scheduler_add_expired_lock_failed", "err", err)
	}

	if _, err := c.AddFunc(j.cfg.Scheduler.StockWarningCron, func() {
		ctx := context.Background()
		if err := j.inv.GenerateStockWarnings(ctx, today()); err != nil {
			slog.Error("scheduler_stock_warning_failed", "err", err)
			return
		}
		// 当日预警摘要广播（同日幂等，000036）
		if j.notifSvc != nil {
			if _, err := j.notifSvc.NotifyDailyAlertSummary(ctx, today()); err != nil {
				slog.Error("scheduler_alert_notify_failed", "err", err)
			}
		}
	}); err != nil {
		slog.Error("scheduler_add_stock_warning_failed", "err", err)
	}

	if _, err := c.AddFunc(j.cfg.Scheduler.LogArchiveCron, func() {
		ctx := context.Background()
		if j.logSvc == nil {
			return
		}
		days := j.logSvc.RetentionDays(ctx)
		if _, err := j.logSvc.CleanupOldLogs(ctx, days); err != nil {
			slog.Error("scheduler_log_archive_failed", "err", err)
		}
	}); err != nil {
		slog.Error("scheduler_add_log_archive_failed", "err", err)
	}

	c.Start()
	return c
}
