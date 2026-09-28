package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"yaofang/internal/model"
)

// PrescriptionRepo 处方仓储。
type PrescriptionRepo struct {
	db *gorm.DB
}

// NewPrescriptionRepo 构建处方仓储。
func NewPrescriptionRepo(db *gorm.DB) *PrescriptionRepo { return &PrescriptionRepo{db: db} }

// Create 新建处方。
func (r *PrescriptionRepo) Create(ctx context.Context, p *model.Prescription) error {
	return r.db.WithContext(ctx).Create(p).Error
}

// GetByID 查询处方。
func (r *PrescriptionRepo) GetByID(ctx context.Context, id int64) (*model.Prescription, error) {
	var p model.Prescription
	if err := r.db.WithContext(ctx).First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// LockForUpdate 行级锁查询处方（状态流转必须锁行）。
func (r *PrescriptionRepo) LockForUpdate(ctx context.Context, id int64) (*model.Prescription, error) {
	var p model.Prescription
	err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&p, id).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// UpdateStatus 更新状态并自增版本号（乐观锁）。expectVersion < 0 表示不校验版本。
// 审核/调配/核对痕迹列仅在调用方回填了值时才写入：这些列在处方流转的不同阶段
// 由不同角色设置，无条件写入会把已记录的 reviewed_at / auditor_id 抹成零值。
func (r *PrescriptionRepo) UpdateStatus(ctx context.Context, p *model.Prescription, expectVersion int) (bool, error) {
	q := r.db.WithContext(ctx).Model(&model.Prescription{}).
		Where("id = ?", p.ID)
	if expectVersion >= 0 {
		q = q.Where("version = ?", expectVersion)
	}
	set := map[string]any{
		"status":  p.Status,
		"version": gorm.Expr("version + 1"),
	}
	if p.ReviewedAt != nil {
		set["reviewed_at"] = p.ReviewedAt
	}
	if p.DispensedAt != nil {
		set["dispensed_at"] = p.DispensedAt
	}
	if p.AuditorID != 0 {
		set["auditor_id"] = p.AuditorID
	}
	if p.AuditorName != "" {
		set["auditor_name"] = p.AuditorName
	}
	if p.DispensingPharmacistID != 0 {
		set["dispensing_pharmacist_id"] = p.DispensingPharmacistID
	}
	if p.DispensingPharmacistName != "" {
		set["dispensing_pharmacist_name"] = p.DispensingPharmacistName
	}
	if p.CheckerID != 0 {
		set["checker_id"] = p.CheckerID
	}
	if p.CheckerName != "" {
		set["checker_name"] = p.CheckerName
	}
	set["updated_at"] = time.Now()

	res := q.Updates(set)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// UpdateBase 更新处方基础字段（录入/修改阶段）。
// 使用 Select("*") 全量更新（含零值），修复 GORM 结构体更新跳过零值导致的
// 「布尔标记无法清除」问题（docs/15 H1）。
func (r *PrescriptionRepo) UpdateBase(ctx context.Context, p *model.Prescription) error {
	return r.db.WithContext(ctx).Model(p).
		Omit("id", "prescription_no", "created_at", "version").
		Select("*").Updates(p).Error
}

// PrescriptionFilter 处方列表筛选。
type PrescriptionFilter struct {
	Status           string
	PatientID        int64
	PatientName      string
	PrescriptionType int  // 0=不限, 1-5=具体类型
	SpecialOnly      bool // true=仅特殊类型（prescription_type > 0，麻精/毒性/放射性）
	Start            *time.Time
	End              *time.Time
}

// List 分页查询处方。
func (r *PrescriptionRepo) List(ctx context.Context, f PrescriptionFilter, offset, limit int) ([]model.Prescription, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.Prescription{})
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.PatientID > 0 {
		q = q.Where("patient_id = ?", f.PatientID)
	}
	if f.PatientName != "" {
		q = q.Where("patient_name ILIKE ?", "%"+f.PatientName+"%")
	}
	if f.PrescriptionType > 0 {
		q = q.Where("prescription_type = ?", f.PrescriptionType)
	}
	if f.SpecialOnly {
		q = q.Where("prescription_type > 0")
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
	var list []model.Prescription
	if err := q.Order("id DESC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// PrescriptionItemRepo 处方明细仓储。
type PrescriptionItemRepo struct {
	db *gorm.DB
}

// NewPrescriptionItemRepo 构建明细仓储。
func NewPrescriptionItemRepo(db *gorm.DB) *PrescriptionItemRepo { return &PrescriptionItemRepo{db: db} }

// CreateBatch 批量新建明细。
func (r *PrescriptionItemRepo) CreateBatch(ctx context.Context, items []*model.PrescriptionItem) error {
	// GORM 对空切片 Create 返回 ErrEmptySlice（会被当 500 上报）；空明细视为无事发生。
	if len(items) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(items).Error
}

// ListByPrescription 查询某处方明细。
func (r *PrescriptionItemRepo) ListByPrescription(ctx context.Context, prescriptionID int64) ([]model.PrescriptionItem, error) {
	var list []model.PrescriptionItem
	err := r.db.WithContext(ctx).Where("prescription_id = ?", prescriptionID).Order("line_no ASC").Find(&list).Error
	return list, err
}

// GetByID 查询明细。
func (r *PrescriptionItemRepo) GetByID(ctx context.Context, id int64) (*model.PrescriptionItem, error) {
	var it model.PrescriptionItem
	if err := r.db.WithContext(ctx).First(&it, id).Error; err != nil {
		return nil, err
	}
	return &it, nil
}

// UpdateDispensed 累加已发数量并更新状态。
func (r *PrescriptionItemRepo) UpdateDispensed(ctx context.Context, id, qty int64, status string) error {
	return r.db.WithContext(ctx).Model(&model.PrescriptionItem{}).Where("id = ?", id).
		Updates(map[string]any{
			"dispensed_quantity": gorm.Expr("dispensed_quantity + ?", qty),
			"status":             status,
		}).Error
}

// UpdateReturned 累加已退数量并更新状态。
func (r *PrescriptionItemRepo) UpdateReturned(ctx context.Context, id, qty int64, status string) error {
	return r.db.WithContext(ctx).Model(&model.PrescriptionItem{}).Where("id = ?", id).
		Updates(map[string]any{
			"returned_quantity": gorm.Expr("returned_quantity + ?", qty),
			"status":            status,
		}).Error
}

// DispenseRecordRepo 发药记录仓储。
type DispenseRecordRepo struct {
	db *gorm.DB
}

// NewDispenseRecordRepo 构建发药记录仓储。
func NewDispenseRecordRepo(db *gorm.DB) *DispenseRecordRepo { return &DispenseRecordRepo{db: db} }

// CreateBatch 批量新建发药记录。
func (r *DispenseRecordRepo) CreateBatch(ctx context.Context, records []*model.PrescriptionDispenseRecord) error {
	// 同上：空切片不得触发 GORM ErrEmptySlice。
	if len(records) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(records).Error
}

// ListByPrescription 查询某处方发药记录。
func (r *DispenseRecordRepo) ListByPrescription(ctx context.Context, prescriptionID int64) ([]model.PrescriptionDispenseRecord, error) {
	var list []model.PrescriptionDispenseRecord
	err := r.db.WithContext(ctx).Where("prescription_id = ?", prescriptionID).Order("id ASC").Find(&list).Error
	return list, err
}

// ListByItem 查询某明细发药记录。
// 排序：拆零记录优先（is_split DESC），再按 id 升序。
// 退药时整盒记录只能按整盒倍数退回，若先取到整盒记录，会出现「拆零余量足够却报
// 3013 整盒退药须为整盒倍数」的顺序敏感失败；拆零优先可让散片退回优先命中拆零记录。
func (r *DispenseRecordRepo) ListByItem(ctx context.Context, itemID int64) ([]model.PrescriptionDispenseRecord, error) {
	var list []model.PrescriptionDispenseRecord
	err := r.db.WithContext(ctx).Where("item_id = ?", itemID).
		Order("is_split DESC").Order("id ASC").Find(&list).Error
	return list, err
}

// UpdateReturnQty 累加某发药记录已退数量。
func (r *DispenseRecordRepo) UpdateReturnQty(ctx context.Context, id, qty int64) error {
	return r.db.WithContext(ctx).Model(&model.PrescriptionDispenseRecord{}).
		Where("id = ?", id).
		UpdateColumn("return_quantity", gorm.Expr("return_quantity + ?", qty)).Error
}

// PrescriptionAuditRepo 处方审计日志仓储。
type PrescriptionAuditRepo struct {
	db *gorm.DB
}

// NewPrescriptionAuditRepo 构建审计日志仓储。
func NewPrescriptionAuditRepo(db *gorm.DB) *PrescriptionAuditRepo {
	return &PrescriptionAuditRepo{db: db}
}

// Create 写入审计日志。
func (r *PrescriptionAuditRepo) Create(ctx context.Context, log *model.PrescriptionAuditLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

// ListByPrescription 查询某处方审计日志。
func (r *PrescriptionAuditRepo) ListByPrescription(ctx context.Context, prescriptionID int64) ([]model.PrescriptionAuditLog, error) {
	var list []model.PrescriptionAuditLog
	err := r.db.WithContext(ctx).Where("prescription_id = ?", prescriptionID).Order("id ASC").Find(&list).Error
	return list, err
}

// 包装 gorm 错误，供服务层判断。
func IsNotFound(err error) bool { return errors.Is(err, gorm.ErrRecordNotFound) }
