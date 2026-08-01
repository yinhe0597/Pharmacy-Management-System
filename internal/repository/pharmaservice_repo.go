package repository

import (
	"context"

	"gorm.io/gorm"

	"yaofang/internal/model"
)

// ConsultationRepo 用药咨询仓储。
type ConsultationRepo struct {
	db *gorm.DB
}

// NewConsultationRepo 构建咨询仓储。
func NewConsultationRepo(db *gorm.DB) *ConsultationRepo { return &ConsultationRepo{db: db} }

// Create 新建咨询。
func (r *ConsultationRepo) Create(ctx context.Context, c *model.Consultation) error {
	return r.db.WithContext(ctx).Create(c).Error
}

// GetByID 查询咨询。
func (r *ConsultationRepo) GetByID(ctx context.Context, id int64) (*model.Consultation, error) {
	var c model.Consultation
	if err := r.db.WithContext(ctx).First(&c, id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// Update 更新咨询。
func (r *ConsultationRepo) Update(ctx context.Context, c *model.Consultation) error {
	return r.db.WithContext(ctx).Model(c).Updates(c).Error
}

// Delete 删除咨询。
func (r *ConsultationRepo) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.Consultation{}, id).Error
}

// List 分页查询咨询。
func (r *ConsultationRepo) List(ctx context.Context, keyword string, offset, limit int) ([]model.Consultation, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.Consultation{})
	if keyword != "" {
		q = q.Where("patient_name ILIKE ? OR question ILIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Consultation
	if err := q.Order("id DESC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// AdverseReactionRepo 不良反应仓储。
type AdverseReactionRepo struct {
	db *gorm.DB
}

// NewAdverseReactionRepo 构建不良反应仓储。
func NewAdverseReactionRepo(db *gorm.DB) *AdverseReactionRepo { return &AdverseReactionRepo{db: db} }

// Create 新建记录。
func (r *AdverseReactionRepo) Create(ctx context.Context, a *model.AdverseReaction) error {
	return r.db.WithContext(ctx).Create(a).Error
}

// GetByID 查询记录。
func (r *AdverseReactionRepo) GetByID(ctx context.Context, id int64) (*model.AdverseReaction, error) {
	var a model.AdverseReaction
	if err := r.db.WithContext(ctx).First(&a, id).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// Update 更新记录。
func (r *AdverseReactionRepo) Update(ctx context.Context, a *model.AdverseReaction) error {
	return r.db.WithContext(ctx).Model(a).Updates(a).Error
}

// Delete 删除记录。
func (r *AdverseReactionRepo) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.AdverseReaction{}, id).Error
}

// List 分页查询记录。
func (r *AdverseReactionRepo) List(ctx context.Context, drugID int64, keyword string, offset, limit int) ([]model.AdverseReaction, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.AdverseReaction{})
	if drugID > 0 {
		q = q.Where("drug_id = ?", drugID)
	}
	if keyword != "" {
		q = q.Where("patient_name ILIKE ? OR reporter ILIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.AdverseReaction
	if err := q.Order("id DESC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// MedicationGuidanceRepo 用药指导仓储。
type MedicationGuidanceRepo struct {
	db *gorm.DB
}

// NewMedicationGuidanceRepo 构建指导仓储。
func NewMedicationGuidanceRepo(db *gorm.DB) *MedicationGuidanceRepo {
	return &MedicationGuidanceRepo{db: db}
}

// Create 新建指导。
func (r *MedicationGuidanceRepo) Create(ctx context.Context, m *model.MedicationGuidance) error {
	return r.db.WithContext(ctx).Create(m).Error
}

// GetByID 查询指导。
func (r *MedicationGuidanceRepo) GetByID(ctx context.Context, id int64) (*model.MedicationGuidance, error) {
	var m model.MedicationGuidance
	if err := r.db.WithContext(ctx).First(&m, id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// List 分页查询指导。
func (r *MedicationGuidanceRepo) List(ctx context.Context, keyword string, offset, limit int) ([]model.MedicationGuidance, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.MedicationGuidance{})
	if keyword != "" {
		q = q.Where("patient_name ILIKE ? OR content ILIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.MedicationGuidance
	if err := q.Order("id DESC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// Update 更新指导。
func (r *MedicationGuidanceRepo) Update(ctx context.Context, m *model.MedicationGuidance) error {
	return r.db.WithContext(ctx).Model(m).Updates(m).Error
}

// Delete 删除指导。
func (r *MedicationGuidanceRepo) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.MedicationGuidance{}, id).Error
}
