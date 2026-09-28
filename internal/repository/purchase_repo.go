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

// PurchaseOrderRow 采购单列表行：补出供应商名（前端采购列表按 supplier_name 渲染）。
type PurchaseOrderRow struct {
	model.PurchaseOrder
	SupplierName string `json:"supplier_name"`
}

// List 分页查询采购单（含供应商名称）。
func (r *PurchaseOrderRepo) List(ctx context.Context, supplierID int64, status string, offset, limit int) ([]PurchaseOrderRow, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.PurchaseOrder{}).
		Joins("LEFT JOIN suppliers s ON s.id = purchase_orders.supplier_id AND s.deleted_at IS NULL")
	if supplierID > 0 {
		q = q.Where("purchase_orders.supplier_id = ?", supplierID)
	}
	if status != "" {
		q = q.Where("purchase_orders.status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []PurchaseOrderRow
	if err := q.Select("purchase_orders.*, s.name AS supplier_name").
		Order("purchase_orders.id DESC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
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
	if len(items) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(items).Error
}

// POItemRow 采购明细行：补出药品名（前端收货质检弹窗按 drug_name 渲染）。
type POItemRow struct {
	model.PurchaseOrderItem
	DrugName string `json:"drug_name"`
}

// GetByID 查询采购明细。
func (r *POItemRepo) GetByID(ctx context.Context, id int64) (*model.PurchaseOrderItem, error) {
	var it model.PurchaseOrderItem
	if err := r.db.WithContext(ctx).First(&it, id).Error; err != nil {
		return nil, err
	}
	return &it, nil
}

// ListByOrder 查询某采购单明细（含药品名）。
func (r *POItemRepo) ListByOrder(ctx context.Context, orderID int64) ([]POItemRow, error) {
	var list []POItemRow
	err := r.db.WithContext(ctx).Model(&model.PurchaseOrderItem{}).
		Joins("LEFT JOIN drugs d ON d.id = purchase_order_items.drug_id AND d.deleted_at IS NULL").
		Where("purchase_order_items.purchase_order_id = ?", orderID).
		Select("purchase_order_items.*, d.generic_name AS drug_name").
		Order("purchase_order_items.id ASC").Find(&list).Error
	return list, err
}

// UpdateReceived 累加已收数量（无条件）。
//
// Deprecated: 无并发保护，仅用于确有行锁或单写者场景；入库累加请用
// UpdateReceivedWithinLimit（条件更新双保险），否则并发完成多张收货单会超收。
func (r *POItemRepo) UpdateReceived(ctx context.Context, id, qty int64) error {
	return r.db.WithContext(ctx).Model(&model.PurchaseOrderItem{}).
		Where("id = ?", id).
		UpdateColumn("received_quantity", gorm.Expr("received_quantity + ?", qty)).Error
}

// UpdateReceivedWithinLimit 条件累加已收数量：仅当「已收 + 本次 <= 订购量」时才累加，
// 返回是否累加成功（false = 会超收，调用方应返回 ErrReceiveExceeded）。
//
// 该判断与累加在同一条 UPDATE 内完成（原子），避免「无锁读校验 + 盲累加」导致的
// 并发超收（两张收货单同时完成时都读到旧值、都通过校验）。
func (r *POItemRepo) UpdateReceivedWithinLimit(ctx context.Context, id, qty int64) (bool, error) {
	if qty <= 0 {
		return false, nil
	}
	res := r.db.WithContext(ctx).Model(&model.PurchaseOrderItem{}).
		Where("id = ? AND received_quantity + ? <= quantity", id, qty).
		UpdateColumn("received_quantity", gorm.Expr("received_quantity + ?", qty))
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
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
	// GORM 对空切片 Create 返回 ErrEmptySlice（会被当 500 上报）；空明细视为无事发生。
	if len(items) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(items).Error
}

// ReceiptItemRow 收货明细行：补出药品名（前端收货质检弹窗按 drug_name 渲染）。
type ReceiptItemRow struct {
	model.PurchaseReceiptItem
	DrugName string `json:"drug_name"`
}

// ListByReceipt 查询某收货单明细（含药品名）。
func (r *ReceiptItemRepo) ListByReceipt(ctx context.Context, receiptID int64) ([]ReceiptItemRow, error) {
	var list []ReceiptItemRow
	err := r.db.WithContext(ctx).Model(&model.PurchaseReceiptItem{}).
		Joins("LEFT JOIN drugs d ON d.id = purchase_receipt_items.drug_id AND d.deleted_at IS NULL").
		Where("purchase_receipt_items.receipt_id = ?", receiptID).
		Select("purchase_receipt_items.*, d.generic_name AS drug_name").
		Order("purchase_receipt_items.id ASC").Find(&list).Error
	return list, err
}

// SumPendingByOrderItem 统计该采购单明细上「待质检收货单」已申报的数量合计。
// 收货单创建时 received_quantity 尚未累加（完成入库时才累加），因此创建收货单的
// 余额校验必须扣除在途（pending_quality）数量，否则可开出多张合计超订购量的收货单。
func (r *ReceiptItemRepo) SumPendingByOrderItem(ctx context.Context, orderItemID int64) (int64, error) {
	var sum int64
	err := r.db.WithContext(ctx).Model(&model.PurchaseReceiptItem{}).
		Joins("JOIN purchase_receipts pr ON pr.id = purchase_receipt_items.receipt_id").
		Where("purchase_receipt_items.order_item_id = ? AND pr.status = ?", orderItemID, "pending_quality").
		Select("COALESCE(SUM(purchase_receipt_items.received_quantity), 0)").
		Scan(&sum).Error
	return sum, err
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
