package repository

import (
	"context"
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
// 必须用 ON CONFLICT 原子 upsert（与本文件 UpsertAddQuantity 同一范式）：
// 「先 First 后 Create」两步在并发首次配置时双方都读到 not-found、都走 Create，
// 一方撞 uq_stock_setting 报 23505；且 PostgreSQL 下唯一冲突会把事务置入
// aborted(25P02)，后续任何查询都失败，整次配置以 500 收场。
func (r *StockSettingRepo) Upsert(ctx context.Context, s *model.DrugStockSetting) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "drug_id"}, {Name: "location_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"min_quantity": s.MinQuantity,
			"max_quantity": s.MaxQuantity,
			"reorder_qty":  s.ReorderQty,
			"is_enabled":   s.IsEnabled,
			"updated_at":   gorm.Expr("NOW()"),
		}),
	}).Create(s).Error
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

// UpsertAddQuantity 原子 upsert：不存在则按 inv 新建，存在则 quantity += inv.Quantity。
// 依赖唯一约束 uq_inventory (drug_id, location_id, batch_no, expiry_date, is_split) 与
// ON CONFLICT，避免「先查后插」在并发下唯一键冲突——PostgreSQL 下冲突会使事务进入 aborted
// 状态（25P02），此时任何回退查询都会失败，导致并发入库整单失败而非累加。
// 返回落库后的最终数量（供流水 before/after 计算）。
func (r *InventoryRepo) UpsertAddQuantity(ctx context.Context, inv *model.Inventory) (int64, error) {
	res := r.db.WithContext(ctx).Clauses(
		clause.OnConflict{
			Columns: []clause.Column{
				{Name: "drug_id"}, {Name: "location_id"},
				{Name: "batch_no"}, {Name: "expiry_date"}, {Name: "is_split"},
			},
			DoUpdates: clause.Assignments(map[string]any{
				"quantity":   gorm.Expr("inventory.quantity + ?", inv.Quantity),
				"updated_at": gorm.Expr("NOW()"),
			}),
		},
		clause.Returning{},
	).Create(inv)
	if res.Error != nil {
		return 0, res.Error
	}
	return inv.Quantity, nil
}

// AddExpiredFromReceipt 收货时若该批次已过期，允许入库但状态直接锁定（收货侧校验在服务层完成）。
func (r *InventoryRepo) SetStatus(ctx context.Context, id int64, status int) error {
	return r.db.WithContext(ctx).Model(&model.Inventory{}).Where("id = ?", id).Update("status", status).Error
}

// Deduct 条件扣减数量：quantity >= qty 才扣，返回是否成功（防止负库存）。
// qty<=0 恒真于条件判断并会写成加法，直接返回 false 拒绝（防调用方漏校验）。
func (r *InventoryRepo) Deduct(ctx context.Context, id, qty int64) (bool, error) {
	if qty <= 0 {
		return false, nil
	}
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
	if qty <= 0 {
		return false, nil
	}
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
	Keyword    string // 药品通用名/商品名/编码/拼音码（前端库存搜索框，此前被静默丢弃）
	Status     int    // 0=全部
	NearExpiry bool
	BelowMin   bool // 与库存设置联动（在服务层处理）
}

// InventoryRow 库存列表行：在批次字段基础上补出药品名与库房名。
// 前端库存列表按 drug_name / location_name 渲染，纯 model.Inventory 无此两列会整列空白。
type InventoryRow struct {
	model.Inventory
	DrugName     string `json:"drug_name"`
	LocationName string `json:"location_name"`
}

// List 分页查询库存（含药品与库房名称）。
func (r *InventoryRepo) List(ctx context.Context, f InventoryListFilter, offset, limit int) ([]InventoryRow, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.Inventory{}).
		Joins("LEFT JOIN drugs d ON d.id = inventory.drug_id AND d.deleted_at IS NULL").
		Joins("LEFT JOIN inventory_locations l ON l.id = inventory.location_id")
	if f.DrugID > 0 {
		q = q.Where("inventory.drug_id = ?", f.DrugID)
	}
	if f.LocationID > 0 {
		q = q.Where("inventory.location_id = ?", f.LocationID)
	}
	if f.BatchNo != "" {
		q = q.Where("inventory.batch_no ILIKE ?", "%"+f.BatchNo+"%")
	}
	if f.Keyword != "" {
		k := "%" + f.Keyword + "%"
		q = q.Where("d.generic_name ILIKE ? OR d.brand_name ILIKE ? OR d.py_code ILIKE ? OR d.code ILIKE ?", k, k, k, k)
	}
	if f.Status > 0 {
		q = q.Where("inventory.status = ?", f.Status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []InventoryRow
	if err := q.Select("inventory.*, d.generic_name AS drug_name, l.name AS location_name").
		Order("inventory.drug_id ASC, inventory.expiry_date ASC").
		Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// FindAvailableForDispenseUnit 按 FEFO 查找可发药批次（指定口径 isSplit）：
// 同口径内按效期升序、入库时间升序；优先拆零库存由调用方按需选择。排除已锁定与已过期批次。
//
// 效期比较用 CURRENT_DATE 而非传入时刻：expiry_date 是 DATE 列，PostgreSQL 会把它
// 按会话时区提升为当天 00:00 再与参数比较。若传 time.Now()（带时分秒），
// 「当天到期」的批次会因 00:00 < 当前时刻而被整日排除，无法发药；
// 同一参数传给 LockExpired 的 < 比较又会把当天批次判为已过期并锁定——
// 两条路径自相矛盾。域规则 rule.IsExpired 明确定义「效期早于今天才算过期」，
// 故此处按 date-only 比较，与 rule/预警口径一致。
func (r *InventoryRepo) FindAvailableForDispenseUnit(ctx context.Context, drugID, locationID int64, isSplit bool) ([]model.Inventory, error) {
	var list []model.Inventory
	err := r.db.WithContext(ctx).
		Where("drug_id = ? AND location_id = ? AND is_split = ? AND status = 1 AND quantity > reserved_quantity AND expiry_date >= CURRENT_DATE",
			drugID, locationID, isSplit).
		Order("expiry_date ASC, received_at ASC").
		Find(&list).Error
	return list, err
}

// FindAvailableForDispense 全口径可发药批次视图（供展示，FEFO 排序，优先拆零）。
// 效期口径同 FindAvailableForDispenseUnit（date-only）。
func (r *InventoryRepo) FindAvailableForDispense(ctx context.Context, drugID, locationID int64) ([]model.Inventory, error) {
	var list []model.Inventory
	err := r.db.WithContext(ctx).
		Where("drug_id = ? AND location_id = ? AND status = 1 AND quantity > reserved_quantity AND expiry_date >= CURRENT_DATE",
			drugID, locationID).
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
// 「已过期」= 效期早于今天（date-only），当天到期仍可发药，与 rule.IsExpired 一致；
// 若按 expiry_date < <带时分秒的时刻> 比较，当天到期批次会被误判过期并锁死。
func (r *InventoryRepo) LockExpired(ctx context.Context) (int64, error) {
	res := r.db.WithContext(ctx).Model(&model.Inventory{}).
		Where("status = 1 AND expiry_date < CURRENT_DATE").
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

// HasOpenByKey 判断是否存在未处理的同类预警（去重）。仅作快速跳过用；
// 真正的一致性保证由 CreateIfAbsent 的「部分唯一索引 + ON CONFLICT」承担（并发/重入安全）。
func (r *StockAlertRepo) HasOpenByKey(ctx context.Context, alertType string, drugID int64, batchNo string) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.StockAlert{}).
		Where("alert_type = ? AND drug_id = ? AND batch_no = ? AND status = 'open'", alertType, drugID, batchNo).
		Count(&n).Error
	return n > 0, err
}

// CreateIfAbsent 幂等写入预警：依赖迁移 000033 的部分唯一索引 uq_stock_alerts_open
// （alert_type + drug_id + location_id + batch_no 在 status='open' 下唯一）。
// 冲突时不报错也不新增，返回是否真正创建；调度器并发/重入下不会产生重复预警。
func (r *StockAlertRepo) CreateIfAbsent(ctx context.Context, a *model.StockAlert) (bool, error) {
	res := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "alert_type"}, {Name: "drug_id"},
			{Name: "location_id"}, {Name: "batch_no"},
		},
		// 部分唯一索引带谓词，冲突推断必须显式给出同一谓词
		TargetWhere: clause.Where{Exprs: []clause.Expression{
			clause.Eq{Column: clause.Column{Name: "status"}, Value: "open"},
		}},
		DoNothing: true,
	}).Create(a)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
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
// 返回是否实际发生流转：对不存在或已处置的预警必须返回 false，
// 否则调用方会把「未落库」当成「处置成功」上报（批量点击时静默丢单）。
func (r *StockAlertRepo) UpdateStatus(ctx context.Context, id int64, status string, resolvedBy int64, resolvedByName string) (bool, error) {
	now := time.Now()
	res := r.db.WithContext(ctx).Model(&model.StockAlert{}).
		Where("id = ? AND status = 'open'", id).
		Updates(map[string]any{
			"status":           status,
			"resolved_at":      now,
			"resolved_by":      resolvedBy,
			"resolved_by_name": resolvedByName,
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}
