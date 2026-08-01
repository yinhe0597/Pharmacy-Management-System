package service

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"yaofang/internal/model"
	"yaofang/internal/pkg/errs"
	"yaofang/internal/pkg/money"
	"yaofang/internal/repository"
)

// DrugService 药品主数据服务。
type DrugService struct {
	db *gorm.DB
}

// NewDrugService 构建药品服务。
func NewDrugService(db *gorm.DB) *DrugService { return &DrugService{db: db} }

// computeSplitPrices 计算并写入拆零价格。
func computeSplitPrices(d *model.Drug) {
	// 未启用拆零时拆零价与整盒价一致口径不可用，置 0。
	if !d.IsSplitAllowed || d.PackSize <= 1 {
		d.SplitRetailPrice = 0
		d.SplitPurchasePrice = 0
		return
	}
	d.SplitRetailPrice = money.Cents(d.RetailPrice).SplitPrice(d.PackSize).Int64()
	d.SplitPurchasePrice = money.Cents(d.PurchasePrice).SplitPrice(d.PackSize).Int64()
}

// Create 新建药品。
func (s *DrugService) Create(ctx context.Context, d *model.Drug) error {
	if d.Code == "" || d.GenericName == "" || d.DosageForm == "" || d.Specification == "" || d.Manufacturer == "" {
		return errs.ErrBadRequest
	}
	drugRepo := repository.NewDrugRepo(s.db)
	if _, err := drugRepo.GetByCode(ctx, d.Code); err == nil {
		return errs.ErrDrugCodeExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	// 一品一规一商唯一性
	if _, err := drugRepo.FindByUnique(ctx, d.GenericName, d.Specification, d.Manufacturer, d.DosageForm); err == nil {
		return errs.New(1009, "同规格同厂家的药品已存在", 409)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	computeSplitPrices(d)
	return drugRepo.Create(ctx, d)
}

// Update 更新药品。
func (s *DrugService) Update(ctx context.Context, id int64, d *model.Drug) error {
	existing, err := repository.NewDrugRepo(s.db).GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.ErrNotFound
		}
		return err
	}
	if other, err := repository.NewDrugRepo(s.db).GetByCode(ctx, d.Code); err == nil && other.ID != id {
		return errs.ErrDrugCodeExists
	}
	d.ID = existing.ID
	d.CreatedAt = existing.CreatedAt
	computeSplitPrices(d)
	return repository.NewDrugRepo(s.db).Update(ctx, d)
}

// Get 查询药品详情。
func (s *DrugService) Get(ctx context.Context, id int64) (*model.Drug, error) {
	d, err := repository.NewDrugRepo(s.db).GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, err
	}
	return d, nil
}

// Delete 删除药品：存在下游单据则冻结并返回提示，否则软删。
func (s *DrugService) Delete(ctx context.Context, id int64) error {
	d, err := repository.NewDrugRepo(s.db).GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.ErrNotFound
		}
		return err
	}
	has, err := repository.NewDrugRepo(s.db).HasDownstream(ctx, id)
	if err != nil {
		return err
	}
	if has {
		d.IsFrozen = true
		if err := repository.NewDrugRepo(s.db).Update(ctx, d); err != nil {
			return err
		}
		return errs.ErrDrugFrozen
	}
	return repository.NewDrugRepo(s.db).Delete(ctx, id)
}

// SetStatus 启停用药品。
func (s *DrugService) SetStatus(ctx context.Context, id int64, status int) error {
	d, err := repository.NewDrugRepo(s.db).GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.ErrNotFound
		}
		return err
	}
	d.Status = status
	return repository.NewDrugRepo(s.db).Update(ctx, d)
}

// List 分页查询药品。
func (s *DrugService) List(ctx context.Context, f repository.DrugListFilter, page, pageSize int) ([]model.Drug, int64, error) {
	return repository.NewDrugRepo(s.db).List(ctx, f, (page-1)*pageSize, pageSize)
}

// ---- 分类 ----

// CreateCategory 新建分类。
func (s *DrugService) CreateCategory(ctx context.Context, c *model.DrugCategory) error {
	return repository.NewCategoryRepo(s.db).Create(ctx, c)
}

// ListCategories 分类列表。
func (s *DrugService) ListCategories(ctx context.Context) ([]model.DrugCategory, error) {
	return repository.NewCategoryRepo(s.db).List(ctx)
}

// UpdateCategory 更新分类。
func (s *DrugService) UpdateCategory(ctx context.Context, id int64, c *model.DrugCategory) error {
	existing, err := repository.NewCategoryRepo(s.db).GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.ErrNotFound
		}
		return err
	}
	c.ID = existing.ID
	c.CreatedAt = existing.CreatedAt
	return repository.NewCategoryRepo(s.db).Update(ctx, c)
}

// DeleteCategory 删除分类（存在药品引用则拒绝）。
func (s *DrugService) DeleteCategory(ctx context.Context, id int64) error {
	n, err := repository.NewCategoryRepo(s.db).CountDrugsByCategory(ctx, id)
	if err != nil {
		return err
	}
	if n > 0 {
		return errs.New(1006, "分类下存在药品，无法删除", 409)
	}
	return repository.NewCategoryRepo(s.db).Delete(ctx, id)
}

// ---- 配伍禁忌 ----

// CreateInteraction 新建配伍禁忌。
func (s *DrugService) CreateInteraction(ctx context.Context, i *model.DrugInteraction) error {
	if i.DrugAID == i.DrugBID {
		return errs.ErrBadRequest
	}
	return repository.NewInteractionRepo(s.db).Create(ctx, i)
}

// ListInteractions 分页查询配伍禁忌。
func (s *DrugService) ListInteractions(ctx context.Context, keyword string, page, pageSize int) ([]model.DrugInteraction, int64, error) {
	return repository.NewInteractionRepo(s.db).List(ctx, keyword, (page-1)*pageSize, pageSize)
}

// UpdateInteraction 更新配伍禁忌。
func (s *DrugService) UpdateInteraction(ctx context.Context, id int64, i *model.DrugInteraction) error {
	existing, err := repository.NewInteractionRepo(s.db).GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.ErrNotFound
		}
		return err
	}
	i.ID = existing.ID
	i.CreatedAt = existing.CreatedAt
	return repository.NewInteractionRepo(s.db).Update(ctx, i)
}

// DeleteInteraction 删除配伍禁忌。
func (s *DrugService) DeleteInteraction(ctx context.Context, id int64) error {
	return repository.NewInteractionRepo(s.db).Delete(ctx, id)
}
