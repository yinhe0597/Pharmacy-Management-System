package repository

import (
	"context"

	"gorm.io/gorm"

	"yaofang/internal/model"
)

// DrugRepo 药品主数据仓储。
type DrugRepo struct {
	db *gorm.DB
}

// NewDrugRepo 创建药品仓储。
func NewDrugRepo(db *gorm.DB) *DrugRepo { return &DrugRepo{db: db} }

// Create 新建药品。
func (r *DrugRepo) Create(ctx context.Context, d *model.Drug) error {
	return r.db.WithContext(ctx).Create(d).Error
}

// GetByID 按 ID 查询。
func (r *DrugRepo) GetByID(ctx context.Context, id int64) (*model.Drug, error) {
	var d model.Drug
	if err := r.db.WithContext(ctx).First(&d, id).Error; err != nil {
		return nil, err
	}
	return &d, nil
}

// GetByCode 按编码查询（含软删）。
func (r *DrugRepo) GetByCode(ctx context.Context, code string) (*model.Drug, error) {
	var d model.Drug
	if err := r.db.WithContext(ctx).Unscoped().Where("code = ?", code).First(&d).Error; err != nil {
		return nil, err
	}
	return &d, nil
}

// FindByUnique 按一品一规一商唯一键查询（用于创建前冲突校验）。
func (r *DrugRepo) FindByUnique(ctx context.Context, genericName, specification, manufacturer, dosageForm string) (*model.Drug, error) {
	var d model.Drug
	if err := r.db.WithContext(ctx).
		Where("generic_name = ? AND specification = ? AND manufacturer = ? AND dosage_form = ?",
			genericName, specification, manufacturer, dosageForm).
		First(&d).Error; err != nil {
		return nil, err
	}
	return &d, nil
}

// Update 更新药品。
func (r *DrugRepo) Update(ctx context.Context, d *model.Drug) error {
	return r.db.WithContext(ctx).Model(d).Omit("code", "created_at").Updates(d).Error
}

// Delete 软删除药品（调用方先做冻结检查）。
func (r *DrugRepo) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.Drug{}, id).Error
}

// DrugListFilter 药品列表筛选条件。
type DrugListFilter struct {
	Keyword            string
	CategoryID         int64
	AntibioticLevel    int
	SpecialControlType int
	Status             int // 0=不筛
	IncludeFrozen      bool
}

// List 分页查询药品。
func (r *DrugRepo) List(ctx context.Context, f DrugListFilter, offset, limit int) ([]model.Drug, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.Drug{})
	if f.Keyword != "" {
		q = q.Where("generic_name ILIKE ? OR brand_name ILIKE ? OR py_code ILIKE ? OR code ILIKE ?",
			"%"+f.Keyword+"%", "%"+f.Keyword+"%", "%"+f.Keyword+"%", "%"+f.Keyword+"%")
	}
	if f.CategoryID > 0 {
		q = q.Where("category_id = ?", f.CategoryID)
	}
	if f.AntibioticLevel > 0 {
		q = q.Where("antibiotic_level = ?", f.AntibioticLevel)
	}
	if f.SpecialControlType > 0 {
		q = q.Where("special_control_type = ?", f.SpecialControlType)
	}
	if f.Status > 0 {
		q = q.Where("status = ?", f.Status)
	}
	if !f.IncludeFrozen {
		q = q.Where("is_frozen = FALSE")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Drug
	if err := q.Order("id ASC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// HasDownstream 判断药品是否存在下游单据（库存/采购/处方），用于冻结检查。
func (r *DrugRepo) HasDownstream(ctx context.Context, drugID int64) (bool, error) {
	var n int64
	if err := r.db.WithContext(ctx).Model(&model.Inventory{}).Where("drug_id = ?", drugID).Count(&n).Error; err != nil {
		return false, err
	}
	if n > 0 {
		return true, nil
	}
	if err := r.db.WithContext(ctx).Model(&model.PurchaseOrderItem{}).Where("drug_id = ?", drugID).Count(&n).Error; err != nil {
		return false, err
	}
	if n > 0 {
		return true, nil
	}
	if err := r.db.WithContext(ctx).Model(&model.PrescriptionItem{}).Where("drug_id = ?", drugID).Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

// ---- 分类 ----

// CategoryRepo 药品分类仓储。
type CategoryRepo struct {
	db *gorm.DB
}

// NewCategoryRepo 创建分类仓储。
func NewCategoryRepo(db *gorm.DB) *CategoryRepo { return &CategoryRepo{db: db} }

// Create 新建分类。
func (r *CategoryRepo) Create(ctx context.Context, c *model.DrugCategory) error {
	return r.db.WithContext(ctx).Create(c).Error
}

// GetByID 查询分类。
func (r *CategoryRepo) GetByID(ctx context.Context, id int64) (*model.DrugCategory, error) {
	var c model.DrugCategory
	if err := r.db.WithContext(ctx).First(&c, id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// Update 更新分类。
func (r *CategoryRepo) Update(ctx context.Context, c *model.DrugCategory) error {
	return r.db.WithContext(ctx).Model(c).Omit("code", "created_at").Updates(c).Error
}

// Delete 删除分类（调用方检查是否有药品引用）。
func (r *CategoryRepo) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.DrugCategory{}, id).Error
}

// List 查询分类列表。
func (r *CategoryRepo) List(ctx context.Context) ([]model.DrugCategory, error) {
	var list []model.DrugCategory
	err := r.db.WithContext(ctx).Order("sort_order ASC, id ASC").Find(&list).Error
	return list, err
}

// CountDrugsByCategory 统计分类下的药品数。
func (r *CategoryRepo) CountDrugsByCategory(ctx context.Context, categoryID int64) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.Drug{}).Where("category_id = ?", categoryID).Count(&n).Error
	return n, err
}

// ---- 配伍禁忌 ----

// InteractionRepo 配伍禁忌仓储。
type InteractionRepo struct {
	db *gorm.DB
}

// NewInteractionRepo 创建禁忌仓储。
func NewInteractionRepo(db *gorm.DB) *InteractionRepo { return &InteractionRepo{db: db} }

// Create 新建禁忌。
func (r *InteractionRepo) Create(ctx context.Context, i *model.DrugInteraction) error {
	return r.db.WithContext(ctx).Create(i).Error
}

// GetByID 查询禁忌。
func (r *InteractionRepo) GetByID(ctx context.Context, id int64) (*model.DrugInteraction, error) {
	var i model.DrugInteraction
	if err := r.db.WithContext(ctx).First(&i, id).Error; err != nil {
		return nil, err
	}
	return &i, nil
}

// Update 更新禁忌。
func (r *InteractionRepo) Update(ctx context.Context, i *model.DrugInteraction) error {
	return r.db.WithContext(ctx).Model(i).Updates(i).Error
}

// Delete 删除禁忌。
func (r *InteractionRepo) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.DrugInteraction{}, id).Error
}

// List 分页查询禁忌。
func (r *InteractionRepo) List(ctx context.Context, keyword string, offset, limit int) ([]model.DrugInteraction, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.DrugInteraction{})
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.DrugInteraction
	if err := q.Order("id ASC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// ListByDrugIDs 查询与给定药品集合相关的全部禁忌（用于处方审核），返回 A-B 双向规范化后的集合。
func (r *InteractionRepo) ListByDrugIDs(ctx context.Context, drugIDs []int64) ([]model.DrugInteraction, error) {
	var list []model.DrugInteraction
	err := r.db.WithContext(ctx).
		Where("drug_a_id IN ? OR drug_b_id IN ?", drugIDs, drugIDs).
		Find(&list).Error
	return list, err
}
