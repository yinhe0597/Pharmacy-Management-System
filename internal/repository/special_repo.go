package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"yaofang/internal/model"
)

// SpecialDrugLedgerRepo 麻精药品专账仓储。
type SpecialDrugLedgerRepo struct {
	db *gorm.DB
}

// NewSpecialDrugLedgerRepo 构建专账仓储。
func NewSpecialDrugLedgerRepo(db *gorm.DB) *SpecialDrugLedgerRepo {
	return &SpecialDrugLedgerRepo{db: db}
}

// Create 写入专账记录。
func (r *SpecialDrugLedgerRepo) Create(ctx context.Context, l *model.SpecialDrugLedger) error {
	return r.db.WithContext(ctx).Create(l).Error
}

// LedgerFilter 专账筛选。
type LedgerFilter struct {
	DrugID  int64
	LogType string
	Start   *time.Time
	End     *time.Time
}

// List 分页查询专账。
func (r *SpecialDrugLedgerRepo) List(ctx context.Context, f LedgerFilter, offset, limit int) ([]model.SpecialDrugLedger, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.SpecialDrugLedger{})
	if f.DrugID > 0 {
		q = q.Where("drug_id = ?", f.DrugID)
	}
	if f.LogType != "" {
		q = q.Where("log_type = ?", f.LogType)
	}
	if f.Start != nil {
		q = q.Where("created_at >= ?", f.Start)
	}
	if f.End != nil {
		q = q.Where("created_at <= ?", f.End)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.SpecialDrugLedger
	if err := q.Order("id DESC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// AmpouleReturnRepo 空安瓿回收仓储。
type AmpouleReturnRepo struct {
	db *gorm.DB
}

// NewAmpouleReturnRepo 构建回收仓储。
func NewAmpouleReturnRepo(db *gorm.DB) *AmpouleReturnRepo { return &AmpouleReturnRepo{db: db} }

// Create 新建回收记录。
func (r *AmpouleReturnRepo) Create(ctx context.Context, a *model.AmpouleReturn) error {
	return r.db.WithContext(ctx).Create(a).Error
}

// GetByID 查询回收记录。
func (r *AmpouleReturnRepo) GetByID(ctx context.Context, id int64) (*model.AmpouleReturn, error) {
	var a model.AmpouleReturn
	if err := r.db.WithContext(ctx).First(&a, id).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// Verify 核对空安瓿回收：状态置 verified 并落库核对人（审计字段，五专可追溯）。
// 仅允许 pending → verified，重复核对返回 false（幂等，不覆盖首次核对人）。
func (r *AmpouleReturnRepo) Verify(ctx context.Context, id int64, verifiedBy string) (bool, error) {
	res := r.db.WithContext(ctx).Model(&model.AmpouleReturn{}).
		Where("id = ? AND status = ?", id, "pending").
		Updates(map[string]any{"status": "verified", "verified_by": verifiedBy})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// UpdateStatus 更新状态（pending→verified，不记录核对人；仅供内部/兼容路径使用）。
func (r *AmpouleReturnRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	return r.db.WithContext(ctx).Model(&model.AmpouleReturn{}).Where("id = ?", id).Update("status", status).Error
}

// List 分页查询回收记录。
func (r *AmpouleReturnRepo) List(ctx context.Context, drugID int64, status string, offset, limit int) ([]model.AmpouleReturn, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.AmpouleReturn{})
	if drugID > 0 {
		q = q.Where("drug_id = ?", drugID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.AmpouleReturn
	if err := q.Order("id DESC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}
