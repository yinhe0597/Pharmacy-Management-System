package repository

import (
	"context"

	"gorm.io/gorm"

	"yaofang/internal/model"
)

// RequisitionOrderRepo 领用/补发登记单仓储。
type RequisitionOrderRepo struct {
	db *gorm.DB
}

// NewRequisitionOrderRepo 构建仓储。
func NewRequisitionOrderRepo(db *gorm.DB) *RequisitionOrderRepo { return &RequisitionOrderRepo{db: db} }

// Create 新建登记单。
func (r *RequisitionOrderRepo) Create(ctx context.Context, o *model.RequisitionOrder) error {
	return r.db.WithContext(ctx).Create(o).Error
}

// CreateItems 批量写明细。
func (r *RequisitionOrderRepo) CreateItems(ctx context.Context, items []*model.RequisitionOrderItem) error {
	if len(items) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(items).Error
}

// GetByID 查询登记单。
func (r *RequisitionOrderRepo) GetByID(ctx context.Context, id int64) (*model.RequisitionOrder, error) {
	var o model.RequisitionOrder
	if err := r.db.WithContext(ctx).First(&o, id).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

// ListItems 查询登记单明细。
func (r *RequisitionOrderRepo) ListItems(ctx context.Context, orderID int64) ([]model.RequisitionOrderItem, error) {
	var list []model.RequisitionOrderItem
	if err := r.db.WithContext(ctx).Where("requisition_order_id = ?", orderID).Order("id ASC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// List 分页查询登记单（可按库房/目的筛选）。
func (r *RequisitionOrderRepo) List(ctx context.Context, locationID int64, purpose string, offset, limit int) ([]model.RequisitionOrder, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.RequisitionOrder{})
	if locationID > 0 {
		q = q.Where("location_id = ?", locationID)
	}
	if purpose != "" {
		q = q.Where("purpose = ?", purpose)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.RequisitionOrder
	if err := q.Order("id DESC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}
