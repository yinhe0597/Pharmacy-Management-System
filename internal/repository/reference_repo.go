package repository

import (
	"context"

	"gorm.io/gorm"

	"yaofang/internal/model"
)

// ReferenceRepo 参考数据只读仓储（v1.3：ICD-10/集采/医保/耗材/非医保）。
type ReferenceRepo struct {
	db *gorm.DB
}

// NewReferenceRepo 创建参考数据仓储。
func NewReferenceRepo(db *gorm.DB) *ReferenceRepo { return &ReferenceRepo{db: db} }

// ListDiagnosisCodes 诊断编码模糊搜索（code/disease_name/py_code/chapter_name）。
func (r *ReferenceRepo) ListDiagnosisCodes(ctx context.Context, keyword string, offset, limit int) ([]model.DiagnosisCode, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.DiagnosisCode{}).Where("status = 1")
	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("(code ILIKE ? OR disease_name ILIKE ? OR py_code ILIKE ? OR chapter_name ILIKE ?)", like, like, like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.DiagnosisCode
	if err := q.Order("id ASC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// ListVBPDrugs 集采药品模糊搜索（generic_name/py_code/dosage_form）。
func (r *ReferenceRepo) ListVBPDrugs(ctx context.Context, keyword string, batch int, offset, limit int) ([]model.VBPDrug, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.VBPDrug{}).Where("status = 1")
	if batch > 0 {
		q = q.Where("vbp_batch = ?", batch)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("(generic_name ILIKE ? OR py_code ILIKE ? OR dosage_form ILIKE ?)", like, like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.VBPDrug
	if err := q.Order("id ASC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// ListNHSADrugs 医保药品模糊搜索（drug_name/py_code/dosage_form），可按甲乙类筛选。
func (r *ReferenceRepo) ListNHSADrugs(ctx context.Context, keyword, insuranceClass string, offset, limit int) ([]model.NHSADrug, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.NHSADrug{}).Where("status = 1")
	if insuranceClass != "" {
		q = q.Where("insurance_class = ?", insuranceClass)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("(drug_name ILIKE ? OR py_code ILIKE ? OR dosage_form ILIKE ?)", like, like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.NHSADrug
	if err := q.Order("id ASC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// ListMedicalConsumables 耗材目录模糊搜索（item_name/py_code/category）。
func (r *ReferenceRepo) ListMedicalConsumables(ctx context.Context, keyword, category string, offset, limit int) ([]model.MedicalConsumable, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.MedicalConsumable{}).Where("status = 1")
	if category != "" {
		q = q.Where("category = ?", category)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("(item_name ILIKE ? OR py_code ILIKE ? OR category ILIKE ?)", like, like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.MedicalConsumable
	if err := q.Order("id ASC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// FindNHSAByName 按药品名称精确匹配医保目录（同名多剂型时优先甲类）。
func (r *ReferenceRepo) FindNHSAByName(ctx context.Context, name string) (*model.NHSADrug, error) {
	var d model.NHSADrug
	err := r.db.WithContext(ctx).Where("drug_name = ? AND status = 1", name).
		Order("CASE insurance_class WHEN '甲类' THEN 0 ELSE 1 END, id ASC").
		First(&d).Error
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// FindVBPByName 按通用名精确匹配集采目录（优先最新批次）。
func (r *ReferenceRepo) FindVBPByName(ctx context.Context, name string) (*model.VBPDrug, error) {
	var d model.VBPDrug
	err := r.db.WithContext(ctx).Where("generic_name = ? AND status = 1", name).
		Order("vbp_batch DESC, id ASC").
		First(&d).Error
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// ListNonInsuranceDrugs 非医保药品模糊搜索（drug_name/py_code/category）。
func (r *ReferenceRepo) ListNonInsuranceDrugs(ctx context.Context, keyword, category string, offset, limit int) ([]model.NonInsuranceDrug, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.NonInsuranceDrug{}).Where("status = 1")
	if category != "" {
		q = q.Where("category = ?", category)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("(drug_name ILIKE ? OR py_code ILIKE ? OR category ILIKE ?)", like, like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.NonInsuranceDrug
	if err := q.Order("id ASC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}
