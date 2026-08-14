package service

import (
	"context"
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

// Log 记录操作日志（非阻塞，失败静默忽略）。
func (s *OperationLogService) Log(ctx context.Context, log *model.OperationLog) {
	_ = repository.NewOperationLogRepo(s.db).Create(ctx, log)
}

// List 分页查询操作日志（支持用户/动作/资源/关键字/时间窗口筛选）。
func (s *OperationLogService) List(ctx context.Context, userID int64, action, resource, keyword string, start, end time.Time, page, pageSize int) ([]model.OperationLog, int64, error) {
	return repository.NewOperationLogRepo(s.db).List(ctx, userID, action, resource, keyword, start, end, (page-1)*pageSize, pageSize)
}
