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

func (r *InteractionRepo) ListByDrugIDs(ctx context.Context, drugIDs []int64) ([]model.DrugInteraction, error) {
	var list []model.DrugInteraction
	err := r.db.WithContext(ctx).
		Where("drug_a_id IN ? OR drug_b_id IN ?", drugIDs, drugIDs).
		Find(&list).Error
	return list, err
}

// ---- 药品成分 ----

// DrugIngredientRepo 药品成分映射仓储。
type DrugIngredientRepo struct {
	db *gorm.DB
}

// NewDrugIngredientRepo 创建成分仓储。
func NewDrugIngredientRepo(db *gorm.DB) *DrugIngredientRepo { return &DrugIngredientRepo{db: db} }

// Create 添加成分映射。
func (r *DrugIngredientRepo) Create(ctx context.Context, di *model.DrugIngredient) error {
	return r.db.WithContext(ctx).Create(di).Error
}

// Delete 删除成分映射。
func (r *DrugIngredientRepo) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.DrugIngredient{}, id).Error
}

// ListByDrugID 按药品 ID 查询成分。
func (r *DrugIngredientRepo) ListByDrugID(ctx context.Context, drugID int64) ([]model.DrugIngredient, error) {
	var list []model.DrugIngredient
	err := r.db.WithContext(ctx).Where("drug_id = ?", drugID).Find(&list).Error
	return list, err
}

// ListByDrugIDs 批量查询药品成分。
func (r *DrugIngredientRepo) ListByDrugIDs(ctx context.Context, drugIDs []int64) ([]model.DrugIngredient, error) {
	var list []model.DrugIngredient
	err := r.db.WithContext(ctx).Where("drug_id IN ?", drugIDs).Find(&list).Error
	return list, err
}

// ---- 成分相互作用 ----

// IngredientInteractionRepo 成分级交互规则仓储。
type IngredientInteractionRepo struct {
	db *gorm.DB
}

// NewIngredientInteractionRepo 创建成分交互仓储。
func NewIngredientInteractionRepo(db *gorm.DB) *IngredientInteractionRepo {
	return &IngredientInteractionRepo{db: db}
}

// Create 创建成分交互规则。
func (r *IngredientInteractionRepo) Create(ctx context.Context, ii *model.IngredientInteraction) error {
	return r.db.WithContext(ctx).Create(ii).Error
}

// GetAll 加载全部成分交互规则。
func (r *IngredientInteractionRepo) GetAll(ctx context.Context) ([]model.IngredientInteraction, error) {
	var list []model.IngredientInteraction
	err := r.db.WithContext(ctx).Find(&list).Error
	return list, err
}

// Update 更新成分交互规则。
func (r *IngredientInteractionRepo) Update(ctx context.Context, ii *model.IngredientInteraction) error {
	return r.db.WithContext(ctx).Model(ii).Updates(ii).Error
}

// Delete 删除成分交互规则。
func (r *IngredientInteractionRepo) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.IngredientInteraction{}, id).Error
}

// List 分页查询成分交互规则。
func (r *IngredientInteractionRepo) List(ctx context.Context, offset, limit int) ([]model.IngredientInteraction, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.IngredientInteraction{})
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.IngredientInteraction
	if err := q.Order("id ASC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// ---- 分类相互作用 ----

// ClassInteractionRepo 分类级交互规则仓储。
type ClassInteractionRepo struct {
	db *gorm.DB
}

// NewClassInteractionRepo 创建分类交互仓储。
func NewClassInteractionRepo(db *gorm.DB) *ClassInteractionRepo {
	return &ClassInteractionRepo{db: db}
}

// Create 创建分类交互规则。
func (r *ClassInteractionRepo) Create(ctx context.Context, ci *model.ClassInteractionRule) error {
	return r.db.WithContext(ctx).Create(ci).Error
}

// GetAllActive 加载全部启用的分类交互规则。
func (r *ClassInteractionRepo) GetAllActive(ctx context.Context) ([]model.ClassInteractionRule, error) {
	var list []model.ClassInteractionRule
	err := r.db.WithContext(ctx).Where("is_active = true").Find(&list).Error
	return list, err
}

// GetAll 加载全部分类交互规则。
func (r *ClassInteractionRepo) GetAll(ctx context.Context) ([]model.ClassInteractionRule, error) {
	var list []model.ClassInteractionRule
	err := r.db.WithContext(ctx).Find(&list).Error
	return list, err
}

// Update 更新分类交互规则。
func (r *ClassInteractionRepo) Update(ctx context.Context, ci *model.ClassInteractionRule) error {
	return r.db.WithContext(ctx).Model(ci).Updates(ci).Error
}

// Delete 删除分类交互规则。
func (r *ClassInteractionRepo) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.ClassInteractionRule{}, id).Error
}

// List 分页查询分类交互规则。
func (r *ClassInteractionRepo) List(ctx context.Context, offset, limit int) ([]model.ClassInteractionRule, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.ClassInteractionRule{})
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.ClassInteractionRule
	if err := q.Order("id ASC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// ---- 标签相互作用 ----

// TagInteractionRepo 标签级交互规则仓储。
type TagInteractionRepo struct {
	db *gorm.DB
}

// NewTagInteractionRepo 创建标签交互仓储。
func NewTagInteractionRepo(db *gorm.DB) *TagInteractionRepo {
	return &TagInteractionRepo{db: db}
}

// Create 创建标签交互规则。
func (r *TagInteractionRepo) Create(ctx context.Context, ti *model.TagInteraction) error {
	return r.db.WithContext(ctx).Create(ti).Error
}

// GetAllActive 加载全部启用的标签交互规则。
func (r *TagInteractionRepo) GetAllActive(ctx context.Context) ([]model.TagInteraction, error) {
	var list []model.TagInteraction
	err := r.db.WithContext(ctx).Where("is_active = true").Find(&list).Error
	return list, err
}

// GetAll 加载全部标签交互规则。
func (r *TagInteractionRepo) GetAll(ctx context.Context) ([]model.TagInteraction, error) {
	var list []model.TagInteraction
	err := r.db.WithContext(ctx).Find(&list).Error
	return list, err
}

// Update 更新标签交互规则。
func (r *TagInteractionRepo) Update(ctx context.Context, ti *model.TagInteraction) error {
	return r.db.WithContext(ctx).Model(ti).Updates(ti).Error
}

// Delete 删除标签交互规则。
func (r *TagInteractionRepo) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.TagInteraction{}, id).Error
}

// List 分页查询标签交互规则。
func (r *TagInteractionRepo) List(ctx context.Context, offset, limit int) ([]model.TagInteraction, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.TagInteraction{})
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.TagInteraction
	if err := q.Order("id ASC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// ---- 交互结果快照 ----

// InteractionResultRepo 交互检测结果仓储。
type InteractionResultRepo struct {
	db *gorm.DB
}

// NewInteractionResultRepo 创建交互结果仓储。
func NewInteractionResultRepo(db *gorm.DB) *InteractionResultRepo {
	return &InteractionResultRepo{db: db}
}

// BatchCreate 批量保存交互检测结果。
func (r *InteractionResultRepo) BatchCreate(ctx context.Context, results []model.InteractionResult) error {
	if len(results) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&results).Error
}

// ListByPrescription 按处方查询交互结果。
func (r *InteractionResultRepo) ListByPrescription(ctx context.Context, prescriptionID int64) ([]model.InteractionResult, error) {
	var list []model.InteractionResult
	err := r.db.WithContext(ctx).Where("prescription_id = ?", prescriptionID).Find(&list).Error
	return list, err
}

// ---- 药品画像批量加载 ----

// BatchGetDrugProfiles 批量加载药品用于交互检测。
func (r *DrugRepo) BatchGetDrugProfiles(ctx context.Context, drugIDs []int64) ([]model.Drug, error) {
	var list []model.Drug
	err := r.db.WithContext(ctx).Where("id IN ?", drugIDs).Find(&list).Error
	return list, err
}
