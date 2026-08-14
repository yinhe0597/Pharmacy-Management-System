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
