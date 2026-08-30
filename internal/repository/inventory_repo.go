package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"yaofang/internal/model"
)

// LocationRepo 库存地点仓储。
type LocationRepo struct {
	db *gorm.DB
}

// NewLocationRepo 创建地点仓储。
func NewLocationRepo(db *gorm.DB) *LocationRepo { return &LocationRepo{db: db} }

// Create 新建地点。
func (r *LocationRepo) Create(ctx context.Context, l *model.InventoryLocation) error {
	return r.db.WithContext(ctx).Create(l).Error
}

// GetByID 查询地点。
func (r *LocationRepo) GetByID(ctx context.Context, id int64) (*model.InventoryLocation, error) {
	var l model.InventoryLocation
	if err := r.db.WithContext(ctx).First(&l, id).Error; err != nil {
		return nil, err
	}
	return &l, nil
}

// Update 更新地点。
func (r *LocationRepo) Update(ctx context.Context, l *model.InventoryLocation) error {
	return r.db.WithContext(ctx).Model(l).Omit("code", "created_at").Updates(l).Error
}

// List 查询启用地点。
func (r *LocationRepo) List(ctx context.Context) ([]model.InventoryLocation, error) {
	var list []model.InventoryLocation
	err := r.db.WithContext(ctx).Where("is_active = TRUE").Order("type ASC, id ASC").Find(&list).Error
	return list, err
}

// StockSettingRepo 库存上下限设置仓储。
type StockSettingRepo struct {
	db *gorm.DB
}

// NewStockSettingRepo 构建设置仓储。
func NewStockSettingRepo(db *gorm.DB) *StockSettingRepo { return &StockSettingRepo{db: db} }

// Upsert 存在则更新，否则新建（按 drug+location 唯一）。
func (r *StockSettingRepo) Upsert(ctx context.Context, s *model.DrugStockSetting) error {
	var existing model.DrugStockSetting
	err := r.db.WithContext(ctx).Where("drug_id = ? AND location_id = ?", s.DrugID, s.LocationID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r.db.WithContext(ctx).Create(s).Error
	}
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Model(&existing).
		Updates(map[string]any{
			"min_quantity": s.MinQuantity, "max_quantity": s.MaxQuantity,
			"reorder_qty": s.ReorderQty, "is_enabled": s.IsEnabled,
		}).Error
}

// ListEnabled 查询启用状态的设置（含药品名称）。
func (r *StockSettingRepo) ListEnabled(ctx context.Context) ([]model.DrugStockSetting, error) {
	var list []model.DrugStockSetting
	err := r.db.WithContext(ctx).Where("is_enabled = TRUE").Find(&list).Error
	return list, err
}

// InventoryRepo 库存批次仓储（核心）。
type InventoryRepo struct {
	db *gorm.DB
}

// NewInventoryRepo 构建库存仓储。
func NewInventoryRepo(db *gorm.DB) *InventoryRepo { return &InventoryRepo{db: db} }

// GetByID 查询库存行。
func (r *InventoryRepo) GetByID(ctx context.Context, id int64) (*model.Inventory, error) {
	var inv model.Inventory
	if err := r.db.WithContext(ctx).First(&inv, id).Error; err != nil {
		return nil, err
	}
	return &inv, nil
}

// LockForUpdate 行级锁查询库存行（必须在事务内调用）。
func (r *InventoryRepo) LockForUpdate(ctx context.Context, id int64) (*model.Inventory, error) {
	var inv model.Inventory
	err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&inv, id).Error
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

// FindByKey 按唯一键查询库存行。
func (r *InventoryRepo) FindByKey(ctx context.Context, drugID, locationID int64, batchNo string, expiryDate time.Time, isSplit bool) (*model.Inventory, error) {
	var inv model.Inventory
	err := r.db.WithContext(ctx).
		Where("drug_id = ? AND location_id = ? AND batch_no = ? AND expiry_date = ? AND is_split = ?",
			drugID, locationID, batchNo, expiryDate, isSplit).
		First(&inv).Error
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

// Create 新建库存行。
func (r *InventoryRepo) Create(ctx context.Context, inv *model.Inventory) error {
	return r.db.WithContext(ctx).Create(inv).Error
}

// Add 给已存在库存行累加数量。
func (r *InventoryRepo) Add(ctx context.Context, id, qty int64) error {
	return r.db.WithContext(ctx).Model(&model.Inventory{}).
		Where("id = ?", id).
		UpdateColumn("quantity", gorm.Expr("quantity + ?", qty)).Error
}

// AddExpiredFromReceipt 收货时若该批次已过期，允许入库但状态直接锁定（收货侧校验在服务层完成）。
func (r *InventoryRepo) SetStatus(ctx context.Context, id int64, status int) error {
	return r.db.WithContext(ctx).Model(&model.Inventory{}).Where("id = ?", id).Update("status", status).Error
}

// Deduct 条件扣减数量：quantity >= qty 才扣，返回是否成功（防止负库存）。
func (r *InventoryRepo) Deduct(ctx context.Context, id, qty int64) (bool, error) {
	res := r.db.WithContext(ctx).Model(&model.Inventory{}).
		Where("id = ? AND quantity >= ?", id, qty).
		UpdateColumn("quantity", gorm.Expr("quantity - ?", qty))
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// DeductAvailable 可用量条件扣减：quantity - reserved_quantity >= qty 才扣，返回是否成功。
// 供领用/报损等「无行锁、无条件更新」出库路径使用，防止扣穿已预占库存
// （破坏 quantity >= reserved_quantity 不变量导致发药 Consume 失败）。
func (r *InventoryRepo) DeductAvailable(ctx context.Context, id, qty int64) (bool, error) {
	res := r.db.WithContext(ctx).Model(&model.Inventory{}).
		Where("id = ? AND quantity - reserved_quantity >= ?", id, qty).
		UpdateColumn("quantity", gorm.Expr("quantity - ?", qty))
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// Reserve 预占：可用量 quantity-reserved >= qty 才预占，返回是否成功。
func (r *InventoryRepo) Reserve(ctx context.Context, id, qty int64) (bool, error) {
	res := r.db.WithContext(ctx).Model(&model.Inventory{}).
		Where("id = ? AND quantity - reserved_quantity >= ?", id, qty).
		UpdateColumn("reserved_quantity", gorm.Expr("reserved_quantity + ?", qty))
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// ReleaseReserve 释放预占：reserved_quantity 足够才释放。
func (r *InventoryRepo) ReleaseReserve(ctx context.Context, id, qty int64) (bool, error) {
	res := r.db.WithContext(ctx).Model(&model.Inventory{}).
		Where("id = ? AND reserved_quantity >= ?", id, qty).
		UpdateColumn("reserved_quantity", gorm.Expr("reserved_quantity - ?", qty))
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// Consume 发药实扣：同时扣减 quantity 与 reserved_quantity（预占转实扣）。
func (r *InventoryRepo) Consume(ctx context.Context, id, qty int64) (bool, error) {
	res := r.db.WithContext(ctx).Model(&model.Inventory{}).
		Where("id = ? AND quantity >= ? AND reserved_quantity >= ?", id, qty, qty).
		Updates(map[string]any{
			"quantity":          gorm.Expr("quantity - ?", qty),
			"reserved_quantity": gorm.Expr("reserved_quantity - ?", qty),
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// InventoryListFilter 库存列表筛选。
type InventoryListFilter struct {
	DrugID     int64
	LocationID int64
	BatchNo    string
	Status     int // 0=全部
	NearExpiry bool
	BelowMin   bool // 与库存设置联动（在服务层处理）
}

// List 分页查询库存（含药品信息）。
func (r *InventoryRepo) List(ctx context.Context, f InventoryListFilter, offset, limit int) ([]model.Inventory, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.Inventory{})
	if f.DrugID > 0 {
		q = q.Where("drug_id = ?", f.DrugID)
	}
	if f.LocationID > 0 {
		q = q.Where("location_id = ?", f.LocationID)
	}
	if f.BatchNo != "" {
		q = q.Where("batch_no ILIKE ?", "%"+f.BatchNo+"%")
	}
	if f.Status > 0 {
		q = q.Where("status = ?", f.Status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Inventory
	if err := q.Order("drug_id ASC, expiry_date ASC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// FindAvailableForDispenseUnit 按 FEFO 查找可发药批次（指定口径 isSplit）：
// 同口径内按效期升序、入库时间升序；优先拆零库存由调用方按需选择。排除已锁定与已过期批次。
func (r *InventoryRepo) FindAvailableForDispenseUnit(ctx context.Context, drugID, locationID int64, isSplit bool, today time.Time) ([]model.Inventory, error) {
	var list []model.Inventory
	err := r.db.WithContext(ctx).
		Where("drug_id = ? AND location_id = ? AND is_split = ? AND status = 1 AND quantity > reserved_quantity AND expiry_date >= ?",
			drugID, locationID, isSplit, today).
		Order("expiry_date ASC, received_at ASC").
		Find(&list).Error
	return list, err
}

// FindAvailableForDispense 全口径可发药批次视图（供展示，FEFO 排序，优先拆零）。
func (r *InventoryRepo) FindAvailableForDispense(ctx context.Context, drugID, locationID int64, today time.Time) ([]model.Inventory, error) {
	var list []model.Inventory
	err := r.db.WithContext(ctx).
		Where("drug_id = ? AND location_id = ? AND status = 1 AND quantity > reserved_quantity AND expiry_date >= ?",
			drugID, locationID, today).
		Order("is_split DESC, expiry_date ASC, received_at ASC").
		Find(&list).Error
	return list, err
}

// FindExpiring 近效期/过期扫描：status=1 的全部批次（供预警任务）。
func (r *InventoryRepo) FindActive(ctx context.Context) ([]model.Inventory, error) {
	var list []model.Inventory
	err := r.db.WithContext(ctx).Where("status = 1 AND quantity > 0").Find(&list).Error
	return list, err
}

// LockExpired 将已过期批次状态置为 2（定时任务兜底）。
func (r *InventoryRepo) LockExpired(ctx context.Context, today time.Time) (int64, error) {
	res := r.db.WithContext(ctx).Model(&model.Inventory{}).
		Where("status = 1 AND expiry_date < ?", today).
		Update("status", 2)
	return res.RowsAffected, res.Error
}

// InventoryTransactionRepo 库存流水仓储。
type InventoryTransactionRepo struct {
	db *gorm.DB
}

// NewInventoryTransactionRepo 构建流水仓储。
func NewInventoryTransactionRepo(db *gorm.DB) *InventoryTransactionRepo {
	return &InventoryTransactionRepo{db: db}
}

// Create 写入流水。
func (r *InventoryTransactionRepo) Create(ctx context.Context, t *model.InventoryTransaction) error {
	return r.db.WithContext(ctx).Create(t).Error
}

// TxnListFilter 流水筛选。
type TxnListFilter struct {
	DrugID     int64
	LocationID int64
	TxnType    string
	RefType    string
	Start      *time.Time
	End        *time.Time
}

// List 分页查询流水。
func (r *InventoryTransactionRepo) List(ctx context.Context, f TxnListFilter, offset, limit int) ([]model.InventoryTransaction, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.InventoryTransaction{})
	if f.DrugID > 0 {
		q = q.Where("drug_id = ?", f.DrugID)
	}
	if f.LocationID > 0 {
		q = q.Where("location_id = ?", f.LocationID)
	}
	if f.TxnType != "" {
		q = q.Where("txn_type = ?", f.TxnType)
	}
	if f.RefType != "" {
		q = q.Where("ref_type = ?", f.RefType)
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
	var list []model.InventoryTransaction
	if err := q.Order("id DESC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// StockReservationRepo 库存预占仓储。
type StockReservationRepo struct {
	db *gorm.DB
}

// NewStockReservationRepo 构建预占仓储。
func NewStockReservationRepo(db *gorm.DB) *StockReservationRepo { return &StockReservationRepo{db: db} }

// Create 新建预占记录。
func (r *StockReservationRepo) Create(ctx context.Context, s *model.StockReservation) error {
	return r.db.WithContext(ctx).Create(s).Error
}

// ListActiveByRef 查询某单据的 active 预占记录。
func (r *StockReservationRepo) ListActiveByRef(ctx context.Context, refType string, refID int64) ([]model.StockReservation, error) {
	var list []model.StockReservation
	err := r.db.WithContext(ctx).
		Where("ref_type = ? AND ref_id = ? AND status = 'active'", refType, refID).
		Find(&list).Error
	return list, err
}

// UpdateStatus 更新预占状态（consumed/released）。
func (r *StockReservationRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	return r.db.WithContext(ctx).Model(&model.StockReservation{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// StockAlertRepo 预警仓储。
type StockAlertRepo struct {
	db *gorm.DB
}

// NewStockAlertRepo 构建预警仓储。
func NewStockAlertRepo(db *gorm.DB) *StockAlertRepo { return &StockAlertRepo{db: db} }

// Create 写入预警。
func (r *StockAlertRepo) Create(ctx context.Context, a *model.StockAlert) error {
	return r.db.WithContext(ctx).Create(a).Error
}

// HasOpenByKey 判断是否存在未处理的同类预警（去重）。
func (r *StockAlertRepo) HasOpenByKey(ctx context.Context, alertType string, drugID int64, batchNo string) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.StockAlert{}).
		Where("alert_type = ? AND drug_id = ? AND batch_no = ? AND status = 'open'", alertType, drugID, batchNo).
		Count(&n).Error
	return n > 0, err
}

// List 分页查询预警。
func (r *StockAlertRepo) List(ctx context.Context, alertType, status string, offset, limit int) ([]model.StockAlert, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.StockAlert{})
	if alertType != "" {
		q = q.Where("alert_type = ?", alertType)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.StockAlert
	if err := q.Order("id DESC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// UpdateStatus 将预警流转到目标状态（resolved/ignored），记录处理人与时间。
func (r *StockAlertRepo) UpdateStatus(ctx context.Context, id int64, status string, resolvedBy int64, resolvedByName string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.StockAlert{}).
		Where("id = ? AND status = 'open'", id).
		Updates(map[string]any{
			"status":           status,
			"resolved_at":      now,
			"resolved_by":      resolvedBy,
			"resolved_by_name": resolvedByName,
		}).Error
}
