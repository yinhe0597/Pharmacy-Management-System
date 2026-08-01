package repository

import (
	"context"

	"gorm.io/gorm"

	"yaofang/internal/model"
)

// SupplierRepo 供应商仓储。
type SupplierRepo struct {
	db *gorm.DB
}

// NewSupplierRepo 创建供应商仓储。
func NewSupplierRepo(db *gorm.DB) *SupplierRepo { return &SupplierRepo{db: db} }

// Create 新建供应商。
func (r *SupplierRepo) Create(ctx context.Context, s *model.Supplier) error {
	return r.db.WithContext(ctx).Create(s).Error
}

// GetByID 查询供应商。
func (r *SupplierRepo) GetByID(ctx context.Context, id int64) (*model.Supplier, error) {
	var s model.Supplier
	if err := r.db.WithContext(ctx).First(&s, id).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

// GetByCode 按编码查询。
func (r *SupplierRepo) GetByCode(ctx context.Context, code string) (*model.Supplier, error) {
	var s model.Supplier
	if err := r.db.WithContext(ctx).Where("code = ?", code).First(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

// Update 更新供应商。
func (r *SupplierRepo) Update(ctx context.Context, s *model.Supplier) error {
	return r.db.WithContext(ctx).Model(s).Omit("code", "created_at").Updates(s).Error
}

// Delete 软删除供应商。
func (r *SupplierRepo) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.Supplier{}, id).Error
}

// List 分页查询供应商。
func (r *SupplierRepo) List(ctx context.Context, keyword string, status int, offset, limit int) ([]model.Supplier, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.Supplier{})
	if keyword != "" {
		q = q.Where("name ILIKE ? OR code ILIKE ? OR contact_person ILIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	if status > 0 {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Supplier
	if err := q.Order("id ASC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// DrugSupplierRepo 药品-供应商关系仓储。
type DrugSupplierRepo struct {
	db *gorm.DB
}

// NewDrugSupplierRepo 创建关系仓储。
func NewDrugSupplierRepo(db *gorm.DB) *DrugSupplierRepo { return &DrugSupplierRepo{db: db} }

// Create 新建关系。
func (r *DrugSupplierRepo) Create(ctx context.Context, ds *model.DrugSupplier) error {
	return r.db.WithContext(ctx).Create(ds).Error
}

// GetByDrugSupplier 查询关系。
func (r *DrugSupplierRepo) GetByDrugSupplier(ctx context.Context, drugID, supplierID int64) (*model.DrugSupplier, error) {
	var ds model.DrugSupplier
	if err := r.db.WithContext(ctx).Where("drug_id = ? AND supplier_id = ?", drugID, supplierID).First(&ds).Error; err != nil {
		return nil, err
	}
	return &ds, nil
}

// Update 更新关系。
func (r *DrugSupplierRepo) Update(ctx context.Context, ds *model.DrugSupplier) error {
	return r.db.WithContext(ctx).Model(ds).Updates(ds).Error
}

// Delete 删除关系。
func (r *DrugSupplierRepo) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.DrugSupplier{}, id).Error
}

// ListByDrug 查询某药品的供应商关系。
func (r *DrugSupplierRepo) ListByDrug(ctx context.Context, drugID int64) ([]model.DrugSupplier, error) {
	var list []model.DrugSupplier
	err := r.db.WithContext(ctx).Where("drug_id = ?", drugID).Order("is_default DESC, id ASC").Find(&list).Error
	return list, err
}

// SetDefault 将某关系置为默认（同时清除同药品其他默认）。
func (r *DrugSupplierRepo) SetDefault(ctx context.Context, drugID, supplierID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.DrugSupplier{}).Where("drug_id = ?", drugID).Update("is_default", false).Error; err != nil {
			return err
		}
		return tx.Model(&model.DrugSupplier{}).Where("drug_id = ? AND supplier_id = ?", drugID, supplierID).Update("is_default", true).Error
	})
}
