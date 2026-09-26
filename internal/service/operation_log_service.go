package service

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"gorm.io/gorm"

	"yaofang/internal/model"
	"yaofang/internal/repository"
)

// OperationLogService 操作日志服务。
type OperationLogService struct {
	db *gorm.DB
}

func NewOperationLogService(db *gorm.DB) *OperationLogService {
	return &OperationLogService{db: db}
}

// Log 记录操作日志（非阻塞：失败不中断业务，但记录错误日志以便排查）。
func (s *OperationLogService) Log(ctx context.Context, log *model.OperationLog) {
	if err := repository.NewOperationLogRepo(s.db).Create(ctx, log); err != nil {
		slog.Error("operation_log_write_failed",
			"err", err, "action", log.Action, "resource", log.Resource, "path", log.Path)
	}
}

// List 分页查询操作日志（支持用户/动作/资源/关键字/时间窗口筛选）。
func (s *OperationLogService) List(ctx context.Context, userID int64, action, resource, keyword string, start, end time.Time, page, pageSize int) ([]model.OperationLog, int64, error) {
	return repository.NewOperationLogRepo(s.db).List(ctx, userID, action, resource, keyword, start, end, (page-1)*pageSize, pageSize)
}

// defaultLogRetentionDays 日志保留天数编译默认值（天）。
const defaultLogRetentionDays = 180

// archiveBatchSize 单次归档上限（条），避免长事务。
const archiveBatchSize = 5000

// RetentionDays 读取保留期设置（log_retention_days，000035）：缺失/非法/负数时回退 180 天。
func (s *OperationLogService) RetentionDays(ctx context.Context) int {
	var st model.SystemSetting
	if err := s.db.WithContext(ctx).First(&st, "key = ?", model.SettingLogRetentionDays).Error; err != nil {
		return defaultLogRetentionDays
	}
	n, err := parseRetentionDays(st.Value)
	if err != nil || n < 0 {
		return defaultLogRetentionDays
	}
	return n
}

// parseRetentionDays 解析保留天数字符串（纯函数，便于单测）。
func parseRetentionDays(v string) (int, error) {
	return strconv.Atoi(v)
}

// CleanupOldLogs 归档超保留期的日志：retentionDays<=0 表示不归档，返回归档总条数。
func (s *OperationLogService) CleanupOldLogs(ctx context.Context, retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		return 0, nil
	}
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	repo := repository.NewOperationLogRepo(s.db)
	var total int64
	for {
		n, err := repo.ArchiveBefore(ctx, cutoff, archiveBatchSize)
		if err != nil {
			return total, err
		}
		total += n
		if n < archiveBatchSize {
			break
		}
	}
	slog.Info("operation_log_archived", "retention_days", retentionDays, "archived", total)
	return total, nil
}
