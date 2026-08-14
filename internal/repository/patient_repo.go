package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"yaofang/internal/model"
)

// PatientRepo 患者档案与过敏史仓储。
type PatientRepo struct {
	db *gorm.DB
}

// NewPatientRepo 创建患者仓储。
func NewPatientRepo(db *gorm.DB) *PatientRepo { return &PatientRepo{db: db} }

// Create 新建患者。
func (r *PatientRepo) Create(ctx context.Context, p *model.Patient) error {
	return r.db.WithContext(ctx).Create(p).Error
}

// GetByID 按 ID 查询。
func (r *PatientRepo) GetByID(ctx context.Context, id int64) (*model.Patient, error) {
	var p model.Patient
	if err := r.db.WithContext(ctx).First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// GetByCardNo 按卡号查询。
func (r *PatientRepo) GetByCardNo(ctx context.Context, cardNo string) (*model.Patient, error) {
	var p model.Patient
	if err := r.db.WithContext(ctx).Where("card_no = ?", cardNo).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// Update 更新患者（Select("*") 全量更新，含零值，修复布尔/文本字段无法清除问题，docs/15 M5）。
func (r *PatientRepo) Update(ctx context.Context, p *model.Patient) error {
	return r.db.WithContext(ctx).Model(p).
		Omit("id", "created_at").
		Select("*").Updates(p).Error
}

// List 分页查询（按姓名/卡号/电话模糊搜索）。
func (r *PatientRepo) List(ctx context.Context, keyword string, offset, limit int) ([]model.Patient, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.Patient{})
	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("name ILIKE ? OR card_no ILIKE ? OR phone ILIKE ?", like, like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Patient
	if err := q.Order("id DESC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// ---- 过敏史 ----

// CreateAllergy 新增过敏记录。
func (r *PatientRepo) CreateAllergy(ctx context.Context, a *model.PatientAllergy) error {
	return r.db.WithContext(ctx).Create(a).Error
}

// ListAllergies 查询患者过敏史。
func (r *PatientRepo) ListAllergies(ctx context.Context, patientID int64) ([]model.PatientAllergy, error) {
	var list []model.PatientAllergy
	if err := r.db.WithContext(ctx).Where("patient_id = ?", patientID).Order("id ASC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// DeleteAllergy 删除过敏记录。
func (r *PatientRepo) DeleteAllergy(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.PatientAllergy{}, id).Error
}

// GetAllergy 按 ID 查询过敏记录。
func (r *PatientRepo) GetAllergy(ctx context.Context, id int64) (*model.PatientAllergy, error) {
	var a model.PatientAllergy
	if err := r.db.WithContext(ctx).First(&a, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, err
	}
	return &a, nil
}

// MedicationHistoryRow 患者用药史查询行（docs/18）。
type MedicationHistoryRow struct {
	DrugName  string    `json:"drug_name"`
	Dosage    string    `json:"dosage"`
	BeginDate time.Time `json:"begin_date"`
}

// MedicationHistory 查询患者用药史（已发药/已退药处方明细，docs/15 L6）。
func (r *PatientRepo) MedicationHistory(ctx context.Context, patientID int64) ([]MedicationHistoryRow, error) {
	rows := []MedicationHistoryRow{}
	err := r.db.WithContext(ctx).Raw(`
		SELECT pi.drug_name, COALESCE(pi.usage_text, '') AS dosage, p.created_at AS begin_date
		FROM prescription_dispense_records dr
		JOIN prescription_items pi ON pi.id = dr.item_id
		JOIN prescriptions p ON p.id = dr.prescription_id
		WHERE p.patient_id = ? AND p.status IN ('dispensed', 'returned')
		GROUP BY pi.drug_name, pi.usage_text, p.created_at, dr.prescription_id
		ORDER BY p.created_at DESC
	`, patientID).Scan(&rows).Error
	return rows, err
}
