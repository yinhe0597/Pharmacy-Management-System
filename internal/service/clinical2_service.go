// Package service 二期就诊模块服务（docs/20 S2-S4）：就诊/病历/合并结算。
package service

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"yaofang/internal/domain/enum"
	"yaofang/internal/model"
	"yaofang/internal/pkg/errs"
	"yaofang/internal/pkg/seq"
	"yaofang/internal/service/port"
)

// ===================== 就诊 =====================

// VisitInput 挂号/分诊输入。
type VisitInput struct {
	PatientID  int64  `json:"patient_id" binding:"required"`
	Department string `json:"department"`
	DoctorID   *int64 `json:"doctor_id"`
	DoctorName string `json:"doctor_name"`
	VisitType  string `json:"visit_type"` // 默认 outpatient
	Remarks    string `json:"remarks"`
}

// VisitFilter 就诊列表筛选。
type VisitFilter struct {
	PatientID int64     `form:"patient_id"`
	DoctorID  int64     `form:"doctor_id"`
	Status    string    `form:"status"`
	Start     time.Time `form:"start"`
	End       time.Time `form:"end"`
}

// VisitService 就诊登记服务。
type VisitService struct {
	db *gorm.DB
}

func NewVisitService(db *gorm.DB) *VisitService { return &VisitService{db: db} }

// Register 挂号/分诊：校验患者存在，生成就诊号，初始状态 waiting。
func (s *VisitService) Register(ctx context.Context, in VisitInput, operator string) (*model.Visit, error) {
	if in.VisitType == "" {
		in.VisitType = enum.VisitTypeOutpatient
	}
	switch in.VisitType {
	case enum.VisitTypeOutpatient, enum.VisitTypeInpatient, enum.VisitTypeRefill:
	default:
		return nil, errs.ErrBadRequest
	}
	var pat model.Patient
	if err := s.db.WithContext(ctx).First(&pat, in.PatientID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrPatientNotFound
		}
		return nil, err
	}
	v := model.Visit{
		VisitNo:      seq.Next("V"),
		PatientID:    pat.ID,
		PatientName:  pat.Name,
		Department:   in.Department,
		DoctorID:     in.DoctorID,
		DoctorName:   in.DoctorName,
		VisitType:    in.VisitType,
		Status:       enum.VisitStatusWaiting,
		RegisteredBy: operator,
		RegisteredAt: time.Now(),
	}
	if err := s.db.WithContext(ctx).Create(&v).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

// List 就诊列表（患者/医生/状态/日期窗口筛选）。
func (s *VisitService) List(ctx context.Context, f VisitFilter, page, pageSize int) ([]model.Visit, int64, error) {
	q := s.db.WithContext(ctx).Model(&model.Visit{})
	if f.PatientID > 0 {
		q = q.Where("patient_id = ?", f.PatientID)
	}
	if f.DoctorID > 0 {
		q = q.Where("doctor_id = ?", f.DoctorID)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if !f.Start.IsZero() {
		q = q.Where("registered_at >= ?", f.Start)
	}
	if !f.End.IsZero() {
		q = q.Where("registered_at <= ?", f.End)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Visit
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// Get 就诊详情。
func (s *VisitService) Get(ctx context.Context, id int64) (*model.Visit, error) {
	var v model.Visit
	if err := s.db.WithContext(ctx).First(&v, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrVisitNotFound
		}
		return nil, err
	}
	return &v, nil
}

// Start 接诊：waiting → visiting。
func (s *VisitService) Start(ctx context.Context, id int64) error {
	return s.transition(ctx, id, enum.VisitStatusWaiting, enum.VisitStatusVisiting, "visited_at")
}

// Finish 结束就诊：visiting → finished。
func (s *VisitService) Finish(ctx context.Context, id int64) error {
	return s.transition(ctx, id, enum.VisitStatusVisiting, enum.VisitStatusFinished, "finished_at")
}

// Cancel 退号：waiting → cancelled。
func (s *VisitService) Cancel(ctx context.Context, id int64) error {
	return s.transition(ctx, id, enum.VisitStatusWaiting, enum.VisitStatusCancelled, "")
}

func (s *VisitService) transition(ctx context.Context, id int64, from, to, tsCol string) error {
	now := time.Now()
	updates := map[string]interface{}{"status": to, "updated_at": now}
	if tsCol == "visited_at" {
		updates["visited_at"] = now
	} else if tsCol == "finished_at" {
		updates["finished_at"] = now
	}
	res := s.db.WithContext(ctx).Model(&model.Visit{}).
		Where("id = ? AND status = ?", id, from).
		Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		// 区分不存在与状态冲突
		var n int64
		s.db.WithContext(ctx).Model(&model.Visit{}).Where("id = ?", id).Count(&n)
		if n == 0 {
			return errs.ErrVisitNotFound
		}
		return errs.ErrVisitState
	}
	return nil
}

// ===================== 病历 =====================

// MedicalRecordInput 病历输入（一就诊一病历，可反复保存）。
type MedicalRecordInput struct {
	ChiefComplaint    string                     `json:"chief_complaint"`
	PresentIllness    string                     `json:"present_illness"`
	PastHistory       string                     `json:"past_history"`
	PhysicalExam      string                     `json:"physical_exam"`
	Temperature       *float64                   `json:"temperature"`
	SystolicPressure  *int                       `json:"systolic_pressure"`
	DiastolicPressure *int                       `json:"diastolic_pressure"`
	Pulse             *int                       `json:"pulse"`
	Diagnosis         string                     `json:"diagnosis"`
	DiagnosisCode     string                     `json:"diagnosis_code"`
	Diagnoses         []MedicalRecordDiagnosisIn `json:"diagnoses"`
}

// MedicalRecordDiagnosisIn 病历结构化诊断输入。
type MedicalRecordDiagnosisIn struct {
	DiagnosisCode string `json:"diagnosis_code" binding:"required"`
	DiagnosisName string `json:"diagnosis_name"`
	IsPrimary     bool   `json:"is_primary"`
	SortOrder     int    `json:"sort_order"`
}

// MedicalRecordService 病历服务。
type MedicalRecordService struct {
	db *gorm.DB
}

func NewMedicalRecordService(db *gorm.DB) *MedicalRecordService { return &MedicalRecordService{db: db} }

// Upsert 创建或更新就诊病历（含多诊断）。
func (s *MedicalRecordService) Upsert(ctx context.Context, visitID int64, in MedicalRecordInput, operator string) (*model.MedicalRecord, error) {
	var visit model.Visit
	if err := s.db.WithContext(ctx).First(&visit, visitID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrVisitNotFound
		}
		return nil, err
	}
	if len(in.Diagnoses) == 0 && in.DiagnosisCode != "" {
		in.Diagnoses = []MedicalRecordDiagnosisIn{{DiagnosisCode: in.DiagnosisCode, IsPrimary: true}}
	}

	rec := model.MedicalRecord{
		VisitID:           visitID,
		PatientID:         visit.PatientID,
		ChiefComplaint:    in.ChiefComplaint,
		PresentIllness:    in.PresentIllness,
		PastHistory:       in.PastHistory,
		PhysicalExam:      in.PhysicalExam,
		Temperature:       in.Temperature,
		SystolicPressure:  in.SystolicPressure,
		DiastolicPressure: in.DiastolicPressure,
		Pulse:             in.Pulse,
		Diagnosis:         in.Diagnosis,
		DiagnosisCode:     in.DiagnosisCode,
		CreatedBy:         operator,
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing model.MedicalRecord
		if err := tx.Where("visit_id = ?", visitID).First(&existing).Error; err == nil {
			rec.ID = existing.ID
			rec.CreatedAt = existing.CreatedAt
			if err := tx.Model(&existing).Select("*").Updates(&rec).Error; err != nil {
				return err
			}
		} else if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := tx.Create(&rec).Error; err != nil {
				return err
			}
		} else {
			return err
		}
		// 重建多诊断（全删全插）
		if err := tx.Where("medical_record_id = ?", rec.ID).Delete(&model.MedicalRecordDiagnosis{}).Error; err != nil {
			return err
		}
		for i, d := range in.Diagnoses {
			if err := tx.Create(&model.MedicalRecordDiagnosis{
				MedicalRecordID: rec.ID,
				DiagnosisCode:   d.DiagnosisCode,
				DiagnosisName:   d.DiagnosisName,
				IsPrimary:       d.IsPrimary,
				SortOrder:       d.SortOrder,
			}).Error; err != nil {
				return err
			}
			_ = i
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

// Get 按就诊取病历（含诊断）。
func (s *MedicalRecordService) Get(ctx context.Context, visitID int64) (*model.MedicalRecord, error) {
	var rec model.MedicalRecord
	if err := s.db.WithContext(ctx).Where("visit_id = ?", visitID).First(&rec).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, err
	}
	var ds []model.MedicalRecordDiagnosis
	if err := s.db.WithContext(ctx).Where("medical_record_id = ?", rec.ID).Order("sort_order ASC").Find(&ds).Error; err != nil {
		return nil, err
	}
	rec.Diagnoses = ds
	return &rec, nil
}

// ===================== 合并结算 =====================

// ChargeService 合并结算服务（docs/20 S4）。
type ChargeService struct {
	db     *gorm.DB
	pricer port.IPricingService
}

func NewChargeService(db *gorm.DB, pricer port.IPricingService) *ChargeService {
	return &ChargeService{db: db, pricer: pricer}
}

// Create 按就诊生成结算单（调用 CalculateBill 聚合费用行）。
func (s *ChargeService) Create(ctx context.Context, visitID int64, discount int64, operator string) (*model.Charge, error) {
	var visit model.Visit
	if err := s.db.WithContext(ctx).First(&visit, visitID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrVisitNotFound
		}
		return nil, err
	}
	if visit.Status != enum.VisitStatusFinished && visit.Status != enum.VisitStatusVisiting {
		return nil, errs.ErrVisitState
	}
	lines, err := s.pricer.CalculateBill(ctx, visitID)
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return nil, errs.ErrChargeEmpty
	}
	total := int64(0)
	for _, l := range lines {
		total += l.Amount
	}
	payable := total - discount
	if payable < 0 {
		return nil, errs.ErrBadRequest
	}
	charge := model.Charge{
		ChargeNo:       seq.Next("C"),
		VisitID:        visitID,
		PatientID:      visit.PatientID,
		PatientName:    visit.PatientName,
		TotalAmount:    total,
		DiscountAmount: discount,
		PayableAmount:  payable,
		Status:         enum.ChargeStatusPending,
		OperatorName:   operator,
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&charge).Error; err != nil {
			return err
		}
		for i, l := range lines {
			itemID := l.RefID
			if itemID == 0 {
				itemID = 0
			}
			if err := tx.Create(&model.ChargeItem{
				ChargeID:  charge.ID,
				VisitID:   &visitID,
				ItemType:  l.ItemType,
				ItemID:    &itemID,
				ItemName:  s.itemName(ctx, tx, l),
				Quantity:  int(l.Quantity),
				UnitPrice: l.UnitPrice,
				Amount:    l.Amount,
				SortOrder: i,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	charge.Items = nil
	return &charge, nil
}

// itemName 从快照源取名称（处方明细/计费记录）。
func (s *ChargeService) itemName(ctx context.Context, tx *gorm.DB, l port.PriceLine) string {
	switch l.ItemType {
	case enum.ChargeItemTypeRegistration:
		return "挂号费"
	case enum.ChargeItemTypeConsultation:
		return "诊查费"
	}
	if l.ItemType == enum.ChargeItemTypeDrug || l.ItemType == enum.ChargeItemTypeConsumable {
		var it model.PrescriptionItem
		if err := tx.WithContext(ctx).Where("drug_id = ?", l.RefID).Order("id DESC").First(&it).Error; err == nil && it.DrugName != "" {
			return it.DrugName
		}
	}
	var cr model.ChargeRecord
	if err := tx.WithContext(ctx).Where("item_id = ? AND item_type = ?", l.RefID, l.ItemType).Order("id DESC").First(&cr).Error; err == nil {
		return cr.ItemName
	}
	return l.ItemType
}

// ChargeFilter 结算单筛选。
type ChargeFilter struct {
	PatientID int64     `form:"patient_id"`
	VisitID   int64     `form:"visit_id"`
	Status    string    `form:"status"`
	Start     time.Time `form:"start"`
	End       time.Time `form:"end"`
}

// List 结算单列表。
func (s *ChargeService) List(ctx context.Context, f ChargeFilter, page, pageSize int) ([]model.Charge, int64, error) {
	q := s.db.WithContext(ctx).Model(&model.Charge{})
	if f.PatientID > 0 {
		q = q.Where("patient_id = ?", f.PatientID)
	}
	if f.VisitID > 0 {
		q = q.Where("visit_id = ?", f.VisitID)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if !f.Start.IsZero() {
		q = q.Where("created_at >= ?", f.Start)
	}
	if !f.End.IsZero() {
		q = q.Where("created_at <= ?", f.End)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Charge
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// Get 结算单详情（含明细）。
func (s *ChargeService) Get(ctx context.Context, id int64) (*model.Charge, error) {
	var c model.Charge
	if err := s.db.WithContext(ctx).First(&c, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrChargeNotFound
		}
		return nil, err
	}
	var items []model.ChargeItem
	if err := s.db.WithContext(ctx).Where("charge_id = ?", id).Order("sort_order ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	c.Items = items
	return &c, nil
}

// Pay 收费：pending → paid。
func (s *ChargeService) Pay(ctx context.Context, id int64, paidAmount int64, operator string) error {
	return s.chargeTransition(ctx, id, enum.ChargeStatusPending, enum.ChargeStatusPaid, paidAmount, operator, "paid_at")
}

// Refund 退费：paid → refunded（全额退，写负金额冲正建议走 charge_records 红冲，这里仅状态回退）。
func (s *ChargeService) Refund(ctx context.Context, id int64, operator string) error {
	return s.chargeTransition(ctx, id, enum.ChargeStatusPaid, enum.ChargeStatusRefunded, 0, operator, "refunded_at")
}

func (s *ChargeService) chargeTransition(ctx context.Context, id int64, from, to string, paid int64, operator, tsCol string) error {
	now := time.Now()
	updates := map[string]interface{}{"status": to, "updated_at": now, "operator_name": operator}
	if to == enum.ChargeStatusPaid {
		updates["paid_amount"] = paid
	}
	if tsCol == "paid_at" {
		updates["paid_at"] = now
	} else if tsCol == "refunded_at" {
		updates["refunded_at"] = now
	}
	res := s.db.WithContext(ctx).Model(&model.Charge{}).Where("id = ? AND status = ?", id, from).Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		var n int64
		s.db.WithContext(ctx).Model(&model.Charge{}).Where("id = ?", id).Count(&n)
		if n == 0 {
			return errs.ErrChargeNotFound
		}
		return errs.ErrChargeState
	}
	return nil
}
