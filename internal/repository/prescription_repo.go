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
func (r *PrescriptionRepo) UpdateStatus(ctx context.Context, p *model.Prescription, expectVersion int) (bool, error) {
	q := r.db.WithContext(ctx).Model(&model.Prescription{}).
		Where("id = ?", p.ID)
	if expectVersion >= 0 {
		q = q.Where("version = ?", expectVersion)
	}
	res := q.Updates(map[string]any{
		"status":                     p.Status,
		"version":                    gorm.Expr("version + 1"),
		"reviewed_at":                p.ReviewedAt,
		"dispensed_at":               p.DispensedAt,
		"auditor_id":                 p.AuditorID,
		"auditor_name":               p.AuditorName,
		"dispensing_pharmacist_id":   p.DispensingPharmacistID,
		"dispensing_pharmacist_name": p.DispensingPharmacistName,
		"checker_id":                 p.CheckerID,
		"checker_name":               p.CheckerName,
		"updated_at":                 time.Now(),
	})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// UpdateBase 更新处方基础字段（录入/修改阶段）。
func (r *PrescriptionRepo) UpdateBase(ctx context.Context, p *model.Prescription) error {
	return r.db.WithContext(ctx).Model(p).Omit("prescription_no", "created_at", "version").Updates(p).Error
}

// PrescriptionFilter 处方列表筛选。
type PrescriptionFilter struct {
	Status           string
	PatientID        int64
	PatientName      string
	PrescriptionType int
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
	return r.db.WithContext(ctx).Create(records).Error
}

// ListByPrescription 查询某处方发药记录。
func (r *DispenseRecordRepo) ListByPrescription(ctx context.Context, prescriptionID int64) ([]model.PrescriptionDispenseRecord, error) {
	var list []model.PrescriptionDispenseRecord
	err := r.db.WithContext(ctx).Where("prescription_id = ?", prescriptionID).Order("id ASC").Find(&list).Error
	return list, err
}

// ListByItem 查询某明细发药记录。
func (r *DispenseRecordRepo) ListByItem(ctx context.Context, itemID int64) ([]model.PrescriptionDispenseRecord, error) {
	var list []model.PrescriptionDispenseRecord
	err := r.db.WithContext(ctx).Where("item_id = ?", itemID).Order("id ASC").Find(&list).Error
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
