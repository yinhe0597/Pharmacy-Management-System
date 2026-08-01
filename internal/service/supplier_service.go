package service

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"yaofang/internal/model"
	"yaofang/internal/pkg/errs"
	"yaofang/internal/repository"
)

// SupplierService 供应商服务。
type SupplierService struct {
	db *gorm.DB
}

// NewSupplierService 构建供应商服务。
func NewSupplierService(db *gorm.DB) *SupplierService { return &SupplierService{db: db} }

// Create 新建供应商。
func (s *SupplierService) Create(ctx context.Context, sp *model.Supplier) error {
	if sp.Code == "" || sp.Name == "" {
		return errs.ErrBadRequest
	}
	if _, err := repository.NewSupplierRepo(s.db).GetByCode(ctx, sp.Code); err == nil {
		return errs.ErrSupplierCodeExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return repository.NewSupplierRepo(s.db).Create(ctx, sp)
}

// Update 更新供应商。
func (s *SupplierService) Update(ctx context.Context, id int64, sp *model.Supplier) error {
	existing, err := repository.NewSupplierRepo(s.db).GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.ErrNotFound
		}
		return err
	}
	if other, err := repository.NewSupplierRepo(s.db).GetByCode(ctx, sp.Code); err == nil && other.ID != id {
		return errs.ErrSupplierCodeExists
	}
	sp.ID = existing.ID
	sp.CreatedAt = existing.CreatedAt
	return repository.NewSupplierRepo(s.db).Update(ctx, sp)
}

// Get 供应商详情。
func (s *SupplierService) Get(ctx context.Context, id int64) (*model.Supplier, error) {
	sp, err := repository.NewSupplierRepo(s.db).GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, err
	}
	return sp, nil
}

// Delete 删除供应商。
func (s *SupplierService) Delete(ctx context.Context, id int64) error {
	return repository.NewSupplierRepo(s.db).Delete(ctx, id)
}

// List 分页查询供应商。
func (s *SupplierService) List(ctx context.Context, keyword string, status, page, pageSize int) ([]model.Supplier, int64, error) {
	return repository.NewSupplierRepo(s.db).List(ctx, keyword, status, (page-1)*pageSize, pageSize)
}

// BindDrug 绑定药品-供应商关系。
func (s *SupplierService) BindDrug(ctx context.Context, drugID, supplierID int64, isDefault bool, purchasePrice int64) error {
	if _, err := repository.NewDrugRepo(s.db).GetByID(ctx, drugID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.ErrNotFound
		}
		return err
	}
	if _, err := repository.NewSupplierRepo(s.db).GetByID(ctx, supplierID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.ErrNotFound
		}
		return err
	}
	dsRepo := repository.NewDrugSupplierRepo(s.db)
	if _, err := dsRepo.GetByDrugSupplier(ctx, drugID, supplierID); err == nil {
		return errs.New(1007, "该供货关系已存在", 409)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	ds := &model.DrugSupplier{DrugID: drugID, SupplierID: supplierID, IsDefault: isDefault, PurchasePrice: purchasePrice, Status: 1}
	return dsRepo.Create(ctx, ds)
}

// ListDrugSuppliers 查询某药品的供货关系。
func (s *SupplierService) ListDrugSuppliers(ctx context.Context, drugID int64) ([]model.DrugSupplier, error) {
	return repository.NewDrugSupplierRepo(s.db).ListByDrug(ctx, drugID)
}

// DeleteDrugSupplier 删除供货关系。
func (s *SupplierService) DeleteDrugSupplier(ctx context.Context, id int64) error {
	return repository.NewDrugSupplierRepo(s.db).Delete(ctx, id)
}
