package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"yaofang/internal/model"
)

type OperationLogRepo struct{ db *gorm.DB }

func NewOperationLogRepo(db *gorm.DB) *OperationLogRepo { return &OperationLogRepo{db: db} }

func (r *OperationLogRepo) Create(ctx context.Context, log *model.OperationLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *OperationLogRepo) List(ctx context.Context, userID int64, action, resource, keyword string, start, end time.Time, offset, limit int) ([]model.OperationLog, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.OperationLog{})
	if userID > 0 {
		q = q.Where("user_id = ?", userID)
	}
	if action != "" {
		q = q.Where("action = ?", action)
	}
	if resource != "" {
		q = q.Where("resource = ?", resource)
	}
	if keyword != "" {
		q = q.Where("username ILIKE ? OR detail ILIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if !start.IsZero() {
		q = q.Where("created_at >= ?", start)
	}
	if !end.IsZero() {
		q = q.Where("created_at <= ?", end)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.OperationLog
	if err := q.Order("id DESC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// ArchiveBefore 归档早于 cutoff 的日志：单条语句原子搬运
// （DELETE ... RETURNING → INSERT 归档表），返回归档条数。
// 分批执行（batch 条/次），调度器循环调用直至返回 0（000035）。
func (r *OperationLogRepo) ArchiveBefore(ctx context.Context, cutoff time.Time, batch int) (int64, error) {
	res := r.db.WithContext(ctx).Exec(`
		WITH moved AS (
			DELETE FROM operation_logs
			WHERE id IN (
				SELECT id FROM operation_logs
				WHERE created_at < ? ORDER BY id LIMIT ?
			)
			RETURNING id, user_id, username, user_role, action, resource,
			          resource_id, method, path, ip, detail, created_at
		)
		INSERT INTO operation_logs_archive
			(id, user_id, username, user_role, action, resource,
			 resource_id, method, path, ip, detail, created_at, archived_at)
		SELECT id, user_id, username, user_role, action, resource,
		       resource_id, method, path, ip, detail, created_at, now()
		FROM moved`, cutoff, batch)
	return res.RowsAffected, res.Error
}
