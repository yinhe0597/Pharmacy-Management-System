package repository

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"yaofang/internal/model"
)

// PurchaseOrderRepo 采购单仓储。
type PurchaseOrderRepo struct {
	db *gorm.DB
}

// NewPurchaseOrderRepo 构建采购单仓储。
func NewPurchaseOrderRepo(db *gorm.DB) *PurchaseOrderRepo { return &PurchaseOrderRepo{db: db} }

// Create 新建采购单。
func (r *PurchaseOrderRepo) Create(ctx context.Context, po *model.PurchaseOrder) error {
	return r.db.WithContext(ctx).Create(po).Error
}

// GetByID 查询采购单。
func (r *PurchaseOrderRepo) GetByID(ctx context.Context, id int64) (*model.PurchaseOrder, error) {
	var po model.PurchaseOrder
	if err := r.db.WithContext(ctx).First(&po, id).Error; err != nil {
		return nil, err
	}
	return &po, nil
}

// LockForUpdate 行级锁查询采购单。
func (r *PurchaseOrderRepo) LockForUpdate(ctx context.Context, id int64) (*model.PurchaseOrder, error) {
	var po model.PurchaseOrder
	err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&po, id).Error
	if err != nil {
		return nil, err
	}
	return &po, nil
}

// UpdateStatus 更新采购单状态。
func (r *PurchaseOrderRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	return r.db.WithContext(ctx).Model(&model.PurchaseOrder{}).Where("id = ?", id).
		Updates(map[string]any{"status": status, "received_at": time.Now()}).Error
}

// List 分页查询采购单。
func (r *PurchaseOrderRepo) List(ctx context.Context, supplierID int64, status string, offset, limit int) ([]model.PurchaseOrder, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.PurchaseOrder{})
	if supplierID > 0 {
		q = q.Where("supplier_id = ?", supplierID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.PurchaseOrder
	if err := q.Order("id DESC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// POItemRepo 采购明细仓储。
type POItemRepo struct {
	db *gorm.DB
}

// NewPOItemRepo 构建明细仓储。
func NewPOItemRepo(db *gorm.DB) *POItemRepo { return &POItemRepo{db: db} }

// CreateBatch 批量新建明细。
func (r *POItemRepo) CreateBatch(ctx context.Context, items []*model.PurchaseOrderItem) error {
	return r.db.WithContext(ctx).Create(items).Error
}

// GetByID 查询采购明细。
func (r *POItemRepo) GetByID(ctx context.Context, id int64) (*model.PurchaseOrderItem, error) {
	var it model.PurchaseOrderItem
	if err := r.db.WithContext(ctx).First(&it, id).Error; err != nil {
		return nil, err
	}
	return &it, nil
}

// ListByOrder 查询某采购单明细。
func (r *POItemRepo) ListByOrder(ctx context.Context, orderID int64) ([]model.PurchaseOrderItem, error) {
	var list []model.PurchaseOrderItem
	err := r.db.WithContext(ctx).Where("purchase_order_id = ?", orderID).Order("id ASC").Find(&list).Error
	return list, err
}

// UpdateReceived 累加已收数量。
func (r *POItemRepo) UpdateReceived(ctx context.Context, id, qty int64) error {
	return r.db.WithContext(ctx).Model(&model.PurchaseOrderItem{}).
		Where("id = ?", id).
		UpdateColumn("received_quantity", gorm.Expr("received_quantity + ?", qty)).Error
}

// PurchaseReceiptRepo 收货单仓储。
type PurchaseReceiptRepo struct {
	db *gorm.DB
}

// NewPurchaseReceiptRepo 构建收货单仓储。
func NewPurchaseReceiptRepo(db *gorm.DB) *PurchaseReceiptRepo { return &PurchaseReceiptRepo{db: db} }

// Create 新建收货单。
func (r *PurchaseReceiptRepo) Create(ctx context.Context, pr *model.PurchaseReceipt) error {
	return r.db.WithContext(ctx).Create(pr).Error
}

// GetByID 查询收货单。
func (r *PurchaseReceiptRepo) GetByID(ctx context.Context, id int64) (*model.PurchaseReceipt, error) {
	var pr model.PurchaseReceipt
	if err := r.db.WithContext(ctx).First(&pr, id).Error; err != nil {
		return nil, err
	}
	return &pr, nil
}

// LockForUpdate 行级锁查询收货单。
func (r *PurchaseReceiptRepo) LockForUpdate(ctx context.Context, id int64) (*model.PurchaseReceipt, error) {
	var pr model.PurchaseReceipt
	err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&pr, id).Error
	if err != nil {
		return nil, err
	}
	return &pr, nil
}

// UpdateStatus 更新收货单状态。
func (r *PurchaseReceiptRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	return r.db.WithContext(ctx).Model(&model.PurchaseReceipt{}).Where("id = ?", id).
		Updates(map[string]any{"status": status, "received_at": time.Now()}).Error
}

// List 分页查询收货单。
func (r *PurchaseReceiptRepo) List(ctx context.Context, supplierID int64, status string, offset, limit int) ([]model.PurchaseReceipt, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.PurchaseReceipt{})
	if supplierID > 0 {
		q = q.Where("supplier_id = ?", supplierID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.PurchaseReceipt
	if err := q.Order("id DESC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// ReceiptItemRepo 收货明细仓储。
type ReceiptItemRepo struct {
	db *gorm.DB
}

// NewReceiptItemRepo 构建收货明细仓储。
func NewReceiptItemRepo(db *gorm.DB) *ReceiptItemRepo { return &ReceiptItemRepo{db: db} }

// CreateBatch 批量新建收货明细。
func (r *ReceiptItemRepo) CreateBatch(ctx context.Context, items []*model.PurchaseReceiptItem) error {
	return r.db.WithContext(ctx).Create(items).Error
}

// ListByReceipt 查询某收货单明细。
func (r *ReceiptItemRepo) ListByReceipt(ctx context.Context, receiptID int64) ([]model.PurchaseReceiptItem, error) {
	var list []model.PurchaseReceiptItem
	err := r.db.WithContext(ctx).Where("receipt_id = ?", receiptID).Order("id ASC").Find(&list).Error
	return list, err
}

// HasQCFailed 判断是否存在质检不合格项。
func (r *ReceiptItemRepo) HasQCFailed(ctx context.Context, receiptID int64) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.PurchaseReceiptItem{}).
		Where("receipt_id = ? AND qc_result = 2", receiptID).Count(&n).Error
	return n > 0, err
}

// HasUninspected 判断是否存在未质检项（qc_result 未登记，非 1/2）。
func (r *ReceiptItemRepo) HasUninspected(ctx context.Context, receiptID int64) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.PurchaseReceiptItem{}).
		Where("receipt_id = ? AND qc_result NOT IN (1, 2)", receiptID).Count(&n).Error
	return n > 0, err
}
