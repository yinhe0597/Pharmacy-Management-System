package repository

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"yaofang/internal/model"
)

// StocktakeRepo 盘点单仓储。
type StocktakeRepo struct {
	db *gorm.DB
}

// NewStocktakeRepo 构建盘点仓储。
func NewStocktakeRepo(db *gorm.DB) *StocktakeRepo { return &StocktakeRepo{db: db} }

// Create 新建盘点单。
func (r *StocktakeRepo) Create(ctx context.Context, s *model.Stocktake) error {
	return r.db.WithContext(ctx).Create(s).Error
}

// GetByID 查询盘点单。
func (r *StocktakeRepo) GetByID(ctx context.Context, id int64) (*model.Stocktake, error) {
	var s model.Stocktake
	if err := r.db.WithContext(ctx).First(&s, id).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

// LockForUpdate 行级锁查询盘点单。
func (r *StocktakeRepo) LockForUpdate(ctx context.Context, id int64) (*model.Stocktake, error) {
	var s model.Stocktake
	err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&s, id).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// UpdateStatus 更新盘点单状态。
func (r *StocktakeRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	updates := map[string]any{"status": status}
	if status == "counting" {
		updates["started_at"] = time.Now()
	}
	if status == "completed" || status == "cancelled" {
		updates["completed_at"] = time.Now()
	}
	return r.db.WithContext(ctx).Model(&model.Stocktake{}).Where("id = ?", id).Updates(updates).Error
}

// List 分页查询盘点单。
func (r *StocktakeRepo) List(ctx context.Context, status string, offset, limit int) ([]model.Stocktake, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.Stocktake{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Stocktake
	if err := q.Order("id DESC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// StocktakeItemRepo 盘点明细仓储。
type StocktakeItemRepo struct {
	db *gorm.DB
}

// NewStocktakeItemRepo 构建盘点明细仓储。
func NewStocktakeItemRepo(db *gorm.DB) *StocktakeItemRepo { return &StocktakeItemRepo{db: db} }

// GetByID 查询盘点明细。
func (r *StocktakeItemRepo) GetByID(ctx context.Context, id int64) (*model.StocktakeItem, error) {
	var it model.StocktakeItem
	if err := r.db.WithContext(ctx).First(&it, id).Error; err != nil {
		return nil, err
	}
	return &it, nil
}

// CreateBatch 批量新建盘点明细。
func (r *StocktakeItemRepo) CreateBatch(ctx context.Context, items []*model.StocktakeItem) error {
	return r.db.WithContext(ctx).Create(items).Error
}

// ListByStocktake 查询某盘点单明细。
func (r *StocktakeItemRepo) ListByStocktake(ctx context.Context, stocktakeID int64) ([]model.StocktakeItem, error) {
	var list []model.StocktakeItem
	err := r.db.WithContext(ctx).Where("stocktake_id = ?", stocktakeID).Order("id ASC").Find(&list).Error
	return list, err
}

// UpdateCounted 录入实盘数与差异（difference = 实盘 - 账面）。
func (r *StocktakeItemRepo) UpdateCounted(ctx context.Context, id, counted int64) error {
	return r.db.WithContext(ctx).Model(&model.StocktakeItem{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"counted_quantity": counted,
			"difference":       gorm.Expr("? - book_quantity", counted),
		}).Error
}

// MarkAdjusted 将盘点明细标记为已调整。
func (r *StocktakeItemRepo) MarkAdjusted(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Model(&model.StocktakeItem{}).Where("id = ?", id).Update("status", "adjusted").Error
}

// ListPending 查询未调整明细。
func (r *StocktakeItemRepo) ListPending(ctx context.Context, stocktakeID int64) ([]model.StocktakeItem, error) {
	var list []model.StocktakeItem
	err := r.db.WithContext(ctx).
		Where("stocktake_id = ? AND status != 'adjusted'", stocktakeID).
		Find(&list).Error
	return list, err
}
