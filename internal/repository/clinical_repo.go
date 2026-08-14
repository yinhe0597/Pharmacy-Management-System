package repository

import (
	"context"

	"gorm.io/gorm"

	"yaofang/internal/model"
)

// ---- 诊疗项目 ----

type ClinicalServiceRepo struct{ db *gorm.DB }

func NewClinicalServiceRepo(db *gorm.DB) *ClinicalServiceRepo { return &ClinicalServiceRepo{db: db} }

func (r *ClinicalServiceRepo) Create(ctx context.Context, s *model.ClinicalService) error {
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *ClinicalServiceRepo) GetByID(ctx context.Context, id int64) (*model.ClinicalService, error) {
	var s model.ClinicalService
	if err := r.db.WithContext(ctx).First(&s, id).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *ClinicalServiceRepo) Update(ctx context.Context, s *model.ClinicalService) error {
	return r.db.WithContext(ctx).Model(s).Select("*").Updates(s).Error
}

func (r *ClinicalServiceRepo) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.ClinicalService{}, id).Error
}

func (r *ClinicalServiceRepo) List(ctx context.Context, keyword string, offset, limit int) ([]model.ClinicalService, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.ClinicalService{})
	if keyword != "" {
		q = q.Where("name ILIKE ? OR code ILIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.ClinicalService
	if err := q.Order("id ASC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// ---- 计费记录 ----

type ChargeRecordRepo struct{ db *gorm.DB }

func NewChargeRecordRepo(db *gorm.DB) *ChargeRecordRepo { return &ChargeRecordRepo{db: db} }

func (r *ChargeRecordRepo) Create(ctx context.Context, cr *model.ChargeRecord) error {
	return r.db.WithContext(ctx).Create(cr).Error
}

// CreateBatch 批量写入计费记录。
func (r *ChargeRecordRepo) CreateBatch(ctx context.Context, crs []*model.ChargeRecord) error {
	if len(crs) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(crs).Error
}

// CountByRef 统计某来源单据已产生的计费记录数（用于幂等去重）。
func (r *ChargeRecordRepo) CountByRef(ctx context.Context, refType string, refID int64) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.ChargeRecord{}).
		Where("ref_type = ? AND ref_id = ? AND amount > 0", refType, refID).
		Count(&n).Error
	return n, err
}

func (r *ChargeRecordRepo) List(ctx context.Context, keyword string, offset, limit int) ([]model.ChargeRecord, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.ChargeRecord{})
	if keyword != "" {
		q = q.Where("patient_name ILIKE ? OR item_name ILIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.ChargeRecord
	if err := q.Order("id DESC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}
