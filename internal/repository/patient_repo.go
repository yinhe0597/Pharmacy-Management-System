package repository

import (
	"context"
	"errors"

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

// Update 更新患者。
func (r *PatientRepo) Update(ctx context.Context, p *model.Patient) error {
	return r.db.WithContext(ctx).Model(p).Omit("created_at").Updates(p).Error
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
