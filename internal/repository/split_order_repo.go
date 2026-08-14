package repository

import (
	"context"

	"gorm.io/gorm"

	"yaofang/internal/model"
)

// SplitOrderRepo 拆零操作单仓储。
type SplitOrderRepo struct {
	db *gorm.DB
}

// NewSplitOrderRepo 创建拆零操作单仓储。
func NewSplitOrderRepo(db *gorm.DB) *SplitOrderRepo { return &SplitOrderRepo{db: db} }

// Create 新建拆零操作单。
func (r *SplitOrderRepo) Create(ctx context.Context, o *model.SplitOrder) error {
	return r.db.WithContext(ctx).Create(o).Error
}

// GetByID 按 ID 查询。
func (r *SplitOrderRepo) GetByID(ctx context.Context, id int64) (*model.SplitOrder, error) {
	var o model.SplitOrder
	if err := r.db.WithContext(ctx).First(&o, id).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

// List 分页查询拆零操作单（可按药品/库房筛选）。
func (r *SplitOrderRepo) List(ctx context.Context, drugID, locationID int64, offset, limit int) ([]model.SplitOrder, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.SplitOrder{})
	if drugID > 0 {
		q = q.Where("drug_id = ?", drugID)
	}
	if locationID > 0 {
		q = q.Where("location_id = ?", locationID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.SplitOrder
	if err := q.Order("id DESC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}
