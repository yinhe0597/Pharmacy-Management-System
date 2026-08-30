package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"yaofang/internal/domain/enum"
	"yaofang/internal/domain/rule"
	"yaofang/internal/model"
	"yaofang/internal/pkg/errs"
	"yaofang/internal/pkg/seq"
	"yaofang/internal/repository"
)

// ---- 盘点 ----

// CreateStocktake 创建盘点单（草稿），并按当前库存快照生成明细。
func (s *InventoryService) CreateStocktake(ctx context.Context, locationID int64, stocktakeType int, operatorID int64) (*model.Stocktake, error) {
	var st *model.Stocktake
	err := s.db.Transaction(func(tx *gorm.DB) error {
		locRepo := repository.NewLocationRepo(tx)
		if _, err := locRepo.GetByID(ctx, locationID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errs.ErrNotFound
			}
			return err
		}
		// 同一库房不允许同时存在进行中的盘点
		var n int64
		if err := tx.Model(&model.Stocktake{}).
			Where("location_id = ? AND status IN ('draft','counting')", locationID).Count(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			return errs.ErrStocktakeAlreadyOpen
		}
		st = &model.Stocktake{
			StocktakeNo: seq.Next("STK"), LocationID: locationID,
			Type: stocktakeType, Status: "draft", StartedBy: operatorID,
		}
		if err := repository.NewStocktakeRepo(tx).Create(ctx, st); err != nil {
			return err
		}
		// 快照库存明细
		var rows []model.Inventory
		if err := tx.Where("location_id = ? AND quantity > 0", locationID).Find(&rows).Error; err != nil {
			return err
		}
		items := make([]*model.StocktakeItem, 0, len(rows))
		for _, r := range rows {
			items = append(items, &model.StocktakeItem{
				StocktakeID: st.ID, InventoryID: r.ID, DrugID: r.DrugID,
				BatchNo: r.BatchNo, ExpiryDate: r.ExpiryDate, IsSplit: r.IsSplit,
				BookQuantity: r.Quantity,
			})
		}
		if len(items) > 0 {
			if err := repository.NewStocktakeItemRepo(tx).CreateBatch(ctx, items); err != nil {
				return err
			}
		}
		return nil
	})
	return st, err
}

// StartStocktake 开始盘点（draft→counting），期间禁止出入库。
func (s *InventoryService) StartStocktake(ctx context.Context, id int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		st, err := repository.NewStocktakeRepo(tx).LockForUpdate(ctx, id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errs.ErrNotFound
			}
			return err
		}
		if st.Status != "draft" {
			return errs.ErrStateConflict
		}
		return repository.NewStocktakeRepo(tx).UpdateStatus(ctx, id, "counting")
	})
}

// EnterCounted 录入实盘数量（可多次录入，覆盖式）。
func (s *InventoryService) EnterCounted(ctx context.Context, stocktakeID int64, counted []CountedItem) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		st, err := repository.NewStocktakeRepo(tx).LockForUpdate(ctx, stocktakeID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errs.ErrNotFound
			}
			return err
		}
		if st.Status != "counting" {
			return errs.ErrStateConflict
		}
		itemRepo := repository.NewStocktakeItemRepo(tx)
		for _, c := range counted {
			it, err := itemRepo.GetByID(ctx, c.ItemID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return errs.ErrNotFound
				}
				return err
			}
			if it.StocktakeID != stocktakeID {
				return errs.ErrBadRequest
			}
			if err := itemRepo.UpdateCounted(ctx, c.ItemID, c.CountedQuantity); err != nil {
				return err
			}
		}
		return nil
	})
}

// CountedItem 实盘录入项。
type CountedItem struct {
	ItemID          int64 `json:"item_id"`
	CountedQuantity int64 `json:"counted_quantity"`
}

// AdjustStocktake 确认差异并调整库存（一次盘点仅允许一次，事务+流水）。
func (s *InventoryService) AdjustStocktake(ctx context.Context, id int64, operatorID int64, operatorName string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		stRepo := repository.NewStocktakeRepo(tx)
		st, err := stRepo.LockForUpdate(ctx, id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errs.ErrNotFound
			}
			return err
		}
		if st.Status != "counting" && st.Status != "adjusted" {
			return errs.ErrStateConflict
		}
		if st.Status == "adjusted" {
			return errs.ErrStocktakeAdjusted
		}
		itemRepo := repository.NewStocktakeItemRepo(tx)
		items, err := itemRepo.ListPending(ctx, id)
		if err != nil {
			return err
		}
		invRepo := repository.NewInventoryRepo(tx)
		for _, it := range items {
			diff := it.CountedQuantity - it.BookQuantity
			if diff == 0 {
				if err := itemRepo.MarkAdjusted(ctx, it.ID); err != nil {
					return err
				}
				continue
			}
			inv, err := invRepo.LockForUpdate(ctx, it.InventoryID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return errs.ErrNotFound
				}
				return err
			}
			var before int64
			if diff > 0 {
				before = inv.Quantity
				if err := invRepo.Add(ctx, inv.ID, diff); err != nil {
					return err
				}
			} else {
				before = inv.Quantity
				// 调整后数量不得低于已预占数量，防止破坏不变量
				if inv.Quantity+diff < inv.ReservedQuantity {
					return errs.ErrStockNotEnough
				}
				ok, err := invRepo.Deduct(ctx, inv.ID, -diff)
				if err != nil {
					return err
				}
				if !ok {
					return errs.ErrNegativeStock
				}
			}
			expiry := inv.ExpiryDate
			txn := &model.InventoryTransaction{
				TransactionNo: seq.Next("ITN"),
				DrugID:        inv.DrugID, LocationID: inv.LocationID,
				BatchNo: inv.BatchNo, ExpiryDate: &expiry,
				Quantity: diff, IsSplit: inv.IsSplit,
				BeforeQuantity: before, AfterQuantity: before + diff,
				TxnType: enum.TxnStocktakeAdjust, RefType: "stocktake", RefID: st.ID,
				OperatorID: operatorID, OperatorName: operatorName,
				Remarks: "盘点差异调整",
			}
			if err := repository.NewInventoryTransactionRepo(tx).Create(ctx, txn); err != nil {
				return err
			}
			if err := itemRepo.MarkAdjusted(ctx, it.ID); err != nil {
				return err
			}
		}
		return stRepo.UpdateStatus(ctx, id, "adjusted")
	})
}

// CompleteStocktake 完成盘点。
func (s *InventoryService) CompleteStocktake(ctx context.Context, id int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		st, err := repository.NewStocktakeRepo(tx).LockForUpdate(ctx, id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errs.ErrNotFound
			}
			return err
		}
		if st.Status != "adjusted" {
			return errs.ErrStateConflict
		}
		return repository.NewStocktakeRepo(tx).UpdateStatus(ctx, id, "completed")
	})
}

// CancelStocktake 取消盘点。
func (s *InventoryService) CancelStocktake(ctx context.Context, id int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		st, err := repository.NewStocktakeRepo(tx).LockForUpdate(ctx, id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errs.ErrNotFound
			}
			return err
		}
		if st.Status == "completed" || st.Status == "adjusted" {
			return errs.ErrStateConflict
		}
		return repository.NewStocktakeRepo(tx).UpdateStatus(ctx, id, "cancelled")
	})
}

// ---- 预警与采购建议 ----

// LockExpiredBatches 过期批次锁定（每日兜底）。
func (s *InventoryService) LockExpiredBatches(ctx context.Context, today time.Time) (int64, error) {
	return repository.NewInventoryRepo(s.db).LockExpired(ctx, today)
}

// GenerateExpiryWarnings 效期预警：近效期与已过期（去重写入）。
func (s *InventoryService) GenerateExpiryWarnings(ctx context.Context, today time.Time, defaultWarningDays int) error {
	rows, err := repository.NewInventoryRepo(s.db).FindActive(ctx)
	if err != nil {
		return err
	}
	drugMap, err := loadDrugMap(ctx, s.db, uniqueIDs(rows, func(r model.Inventory) int64 { return r.DrugID }))
	if err != nil {
		return err
	}
	alertRepo := repository.NewStockAlertRepo(s.db)
	for _, r := range rows {
		drug := drugMap[r.DrugID]
		warningDays := defaultWarningDays
		if drug != nil && drug.ExpiryWarningDays > 0 {
			warningDays = drug.ExpiryWarningDays
		}
		alertType := ""
		msg := ""
		switch {
		case rule.IsExpired(r.ExpiryDate, today):
			alertType, msg = enum.AlertExpired, "批次已过期，禁止发药"
		case rule.IsNearExpiry(r.ExpiryDate, today, warningDays):
			alertType = enum.AlertExpiry
			msg = "近效期提醒，剩余" + fmt.Sprintf("%d天", rule.DaysToExpiry(r.ExpiryDate, today))
		default:
			continue
		}
		exists, err := alertRepo.HasOpenByKey(ctx, alertType, r.DrugID, r.BatchNo)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		expiry := r.ExpiryDate
		if err := alertRepo.Create(ctx, &model.StockAlert{
			AlertType: alertType, DrugID: r.DrugID, LocationID: r.LocationID,
			BatchNo: r.BatchNo, ExpiryDate: &expiry, Quantity: r.Quantity,
			Message: msg, Status: "open",
		}); err != nil {
			return err
		}
	}
	return nil
}

// GenerateStockWarnings 库存下限预警。
func (s *InventoryService) GenerateStockWarnings(ctx context.Context, today time.Time) error {
	settings, err := repository.NewStockSettingRepo(s.db).ListEnabled(ctx)
	if err != nil {
		return err
	}
	alertRepo := repository.NewStockAlertRepo(s.db)
	for _, st := range settings {
		var rows []model.Inventory
		if err := s.db.WithContext(ctx).
			Where("drug_id = ? AND location_id = ? AND status = 1", st.DrugID, st.LocationID).
			Find(&rows).Error; err != nil {
			return err
		}
		drug, err := repository.NewDrugRepo(s.db).GetByID(ctx, st.DrugID)
		if err != nil {
			continue
		}
		var availLDU int64
		for _, r := range rows {
			availLDU += rule.ToLDU(r.IsSplit, r.Available(), drug.PackSize)
		}
		if st.MinQuantity > 0 && availLDU <= st.MinQuantity {
			exists, err := alertRepo.HasOpenByKey(ctx, enum.AlertBelowMin, st.DrugID, "ALL")
			if err != nil {
				return err
			}
			if exists {
				continue
			}
			msg := "低于库存下限，当前可用(拆零单位):" + fmt.Sprintf("%d", availLDU)
			if err := alertRepo.Create(ctx, &model.StockAlert{
				AlertType: enum.AlertBelowMin, DrugID: st.DrugID, LocationID: st.LocationID,
				Quantity: availLDU, Message: msg, Status: "open",
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

// PurchaseSuggestion 采购计划建议项。
type PurchaseSuggestion struct {
	DrugID      int64  `json:"drug_id"`
	DrugName    string `json:"drug_name"`
	LocationID  int64  `json:"location_id"`
	Available   int64  `json:"available"` // 当前可用 LDU
	MinQuantity int64  `json:"min_quantity"`
	MaxQuantity int64  `json:"max_quantity"`
	SuggestQty  int64  `json:"suggest_qty"` // 建议补货量（基本单位，向上取整盒）
}

// PurchaseSuggestions 基于上下限与现有库存生成采购建议。
func (s *InventoryService) PurchaseSuggestions(ctx context.Context) ([]PurchaseSuggestion, error) {
	settings, err := repository.NewStockSettingRepo(s.db).ListEnabled(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]PurchaseSuggestion, 0, len(settings))
	drugRepo := repository.NewDrugRepo(s.db)
	for _, st := range settings {
		drug, err := drugRepo.GetByID(ctx, st.DrugID)
		if err != nil {
			continue
		}
		var rows []model.Inventory
		if err := s.db.WithContext(ctx).
			Where("drug_id = ? AND location_id = ? AND status = 1", st.DrugID, st.LocationID).
			Find(&rows).Error; err != nil {
			return nil, err
		}
		var availLDU int64
		for _, r := range rows {
			availLDU += rule.ToLDU(r.IsSplit, r.Available(), drug.PackSize)
		}
		if st.MaxQuantity > 0 && availLDU >= st.MaxQuantity {
			continue
		}
		need := st.MaxQuantity - availLDU
		if need <= 0 {
			continue
		}
		// 建议补货量取 max(reorder_qty, need)，并按整盒向上取整
		if st.ReorderQty > 0 && st.ReorderQty > need {
			need = st.ReorderQty
		}
		packs := (need + int64(drug.PackSize) - 1) / int64(drug.PackSize)
		out = append(out, PurchaseSuggestion{
			DrugID: drug.ID, DrugName: drug.GenericName, LocationID: st.LocationID,
			Available: availLDU, MinQuantity: st.MinQuantity, MaxQuantity: st.MaxQuantity,
			SuggestQty: packs,
		})
	}
	return out, nil
}

func uniqueIDs(rows []model.Inventory, f func(model.Inventory) int64) []int64 {
	seen := make(map[int64]struct{})
	out := make([]int64, 0, len(rows))
	for _, r := range rows {
		id := f(r)
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
