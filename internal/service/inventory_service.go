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
	"yaofang/internal/pkg/money"
	"yaofang/internal/pkg/seq"
	"yaofang/internal/repository"
	"yaofang/internal/service/port"
)

// StockEntry 入库条目（数量以 isSplit 对应口径）。
type StockEntry struct {
	DrugID     int64     `json:"drug_id"`
	LocationID int64     `json:"location_id"`
	BatchNo    string    `json:"batch_no"`
	ExpiryDate time.Time `json:"expiry_date"`
	IsSplit    bool      `json:"is_split"`
	Quantity   int64     `json:"quantity"`
	UnitPrice  int64     `json:"unit_price"`
}

// AdjustRequest 库存调整（报损/修正）。
type AdjustRequest struct {
	InventoryID int64
	Quantity    int64 // 有符号：正=盘盈/补入，负=报损/扣出
	Reason      string
}

// TransferItem 调拨条目。
type TransferItem struct {
	InventoryID int64 `json:"inventory_id"`
	Quantity    int64 `json:"quantity"` // 行口径数量
}

// InventoryService 库存领域服务，实现 port.IStockService。
type InventoryService struct {
	db *gorm.DB
}

// NewInventoryService 构建库存服务。
func NewInventoryService(db *gorm.DB) *InventoryService { return &InventoryService{db: db} }

// todayNow 当前业务日期（统一取系统当前日）。
func todayNow() time.Time { return time.Now() }

// checkLocationNotCounting 盘点中的库房禁止入库/出库/调拨。
func (s *InventoryService) checkLocationNotCounting(ctx context.Context, tx *gorm.DB, locationID int64) error {
	var n int64
	err := tx.Model(&model.Stocktake{}).
		Where("location_id = ? AND status = 'counting'", locationID).Count(&n).Error
	if err != nil {
		return err
	}
	if n > 0 {
		return errs.ErrStocktakingInProgress
	}
	return nil
}

// loadDrugMap 批量加载药品（供 LDU 折算）。
func loadDrugMap(ctx context.Context, db *gorm.DB, ids []int64) (map[int64]*model.Drug, error) {
	if len(ids) == 0 {
		return map[int64]*model.Drug{}, nil
	}
	var drugs []model.Drug
	if err := db.WithContext(ctx).Where("id IN ?", ids).Find(&drugs).Error; err != nil {
		return nil, err
	}
	m := make(map[int64]*model.Drug, len(drugs))
	for i := range drugs {
		m[drugs[i].ID] = &drugs[i]
	}
	return m, nil
}

// StockIn 通用入库（其他入库），开启独立事务。
func (s *InventoryService) StockIn(ctx context.Context, entries []StockEntry, operatorID int64, operatorName string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		return s.addStockTx(ctx, tx, entries, "", 0, enum.TxnStockIn, operatorID, operatorName)
	})
}

// Requisition 领用出库（医护内部消耗，不计费）。FEFO 扣减。
func (s *InventoryService) Requisition(ctx context.Context, drugID, locationID, quantity int64, reason string, operatorID int64, operatorName string) error {
	if quantity <= 0 {
		return errs.ErrBadRequest
	}
	remarks := "领用出库"
	if reason != "" {
		remarks = "领用: " + reason
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		invRepo := repository.NewInventoryRepo(tx)
		txnRepo := repository.NewInventoryTransactionRepo(tx)
		if err := s.checkLocationNotCounting(ctx, tx, locationID); err != nil {
			return err
		}
		batches, err := invRepo.FindAvailableForDispenseUnit(ctx, drugID, locationID, false, todayNow())
		if err != nil {
			return err
		}
		splitBatches, _ := invRepo.FindAvailableForDispenseUnit(ctx, drugID, locationID, true, todayNow())
		batches = append(batches, splitBatches...)

		remaining := quantity
		for i := range batches {
			if remaining <= 0 {
				break
			}
			avail := batches[i].Available()
			if avail <= 0 {
				continue
			}
			take := remaining
			if take > avail {
				take = avail
			}
			ok, err := invRepo.Deduct(ctx, batches[i].ID, take)
			if err != nil {
				return err
			}
			if !ok {
				return errs.ErrNegativeStock
			}
			before := batches[i].Quantity
			if err := txnRepo.Create(ctx, &model.InventoryTransaction{
				TransactionNo: seq.Next("ITN"),
				DrugID:        drugID, LocationID: locationID,
				BatchNo: batches[i].BatchNo, ExpiryDate: &batches[i].ExpiryDate,
				Quantity: -take, IsSplit: batches[i].IsSplit,
				BeforeQuantity: before, AfterQuantity: before - take,
				TxnType: "requisition", RefType: "requisition", RefID: 0,
				OperatorID: operatorID, OperatorName: operatorName, Remarks: remarks,
			}); err != nil {
				return err
			}
			remaining -= take
		}
		if remaining > 0 {
			return errs.ErrStockNotEnough
		}
		return nil
	})
}

// addStockTx 入库核心：累加或新建库存行并写流水（须在调用方事务内执行）。
func (s *InventoryService) addStockTx(ctx context.Context, tx *gorm.DB, entries []StockEntry, refType string, refID int64, txnType string, operatorID int64, operatorName string) error {
	invRepo := repository.NewInventoryRepo(tx)
	txnRepo := repository.NewInventoryTransactionRepo(tx)
	for _, e := range entries {
		if e.Quantity <= 0 {
			continue
		}
		if err := s.checkLocationNotCounting(ctx, tx, e.LocationID); err != nil {
			return err
		}
		inv, err := invRepo.FindByKey(ctx, e.DrugID, e.LocationID, e.BatchNo, e.ExpiryDate, e.IsSplit)
		var before int64
		if err == nil {
			before = inv.Quantity
			if err := invRepo.Add(ctx, inv.ID, e.Quantity); err != nil {
				return err
			}
		} else if errors.Is(err, gorm.ErrRecordNotFound) {
			newInv := &model.Inventory{
				DrugID: e.DrugID, LocationID: e.LocationID,
				BatchNo: e.BatchNo, ExpiryDate: e.ExpiryDate,
				Quantity: e.Quantity, IsSplit: e.IsSplit,
				UnitPrice: e.UnitPrice, ReceivedAt: time.Now(), Status: 1,
			}
			// 过期批次入库直接锁定，防止发药（服务层校验在采购处做拦截）。
			if rule.IsExpired(e.ExpiryDate, todayNow()) {
				newInv.Status = 2
			}
			if err := invRepo.Create(ctx, newInv); err != nil {
				// 并发下其他事务已创建同键行 → 回退为累加
				existing, ferr := invRepo.FindByKey(ctx, e.DrugID, e.LocationID, e.BatchNo, e.ExpiryDate, e.IsSplit)
				if ferr != nil {
					return err
				}
				before = existing.Quantity
				if aerr := invRepo.Add(ctx, existing.ID, e.Quantity); aerr != nil {
					return aerr
				}
			}
		} else {
			return err
		}
		expiry := e.ExpiryDate
		txn := &model.InventoryTransaction{
			TransactionNo: seq.Next("ITN"),
			DrugID:        e.DrugID, LocationID: e.LocationID,
			BatchNo: e.BatchNo, ExpiryDate: &expiry,
			Quantity: e.Quantity, IsSplit: e.IsSplit,
			BeforeQuantity: before, AfterQuantity: before + e.Quantity,
			TxnType: txnType, RefType: refType, RefID: refID,
			OperatorID: operatorID, OperatorName: operatorName,
		}
		if err := txnRepo.Create(ctx, txn); err != nil {
			return err
		}
	}
	return nil
}

// Transfer 调拨：from 库房扣减 + to 库房累加，单事务。
func (s *InventoryService) Transfer(ctx context.Context, fromLoc, toLoc int64, items []TransferItem, operatorID int64, operatorName string) error {
	if fromLoc == toLoc {
		return errs.ErrBadRequest
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.checkLocationNotCounting(ctx, tx, fromLoc); err != nil {
			return err
		}
		if err := s.checkLocationNotCounting(ctx, tx, toLoc); err != nil {
			return err
		}
		invRepo := repository.NewInventoryRepo(tx)
		txnRepo := repository.NewInventoryTransactionRepo(tx)
		for _, it := range items {
			inv, err := invRepo.LockForUpdate(ctx, it.InventoryID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return errs.ErrNotFound
				}
				return err
			}
			if inv.LocationID != fromLoc {
				return errs.ErrBadRequest
			}
			if inv.Available() < it.Quantity {
				return errs.ErrStockNotEnough
			}
			ok, err := invRepo.Deduct(ctx, inv.ID, it.Quantity)
			if err != nil {
				return err
			}
			if !ok {
				return errs.ErrNegativeStock
			}
			// 写入转出流水
			expiry := inv.ExpiryDate
			before := inv.Quantity
			out := &model.InventoryTransaction{
				TransactionNo: seq.Next("ITN"),
				DrugID:        inv.DrugID, LocationID: fromLoc,
				BatchNo: inv.BatchNo, ExpiryDate: &expiry,
				Quantity: -it.Quantity, IsSplit: inv.IsSplit,
				BeforeQuantity: before, AfterQuantity: before - it.Quantity,
				TxnType: enum.TxnTransferOut, RefType: "transfer_order", RefID: 0,
				OperatorID: operatorID, OperatorName: operatorName,
			}
			if err := txnRepo.Create(ctx, out); err != nil {
				return err
			}
			// 目标端累加（保持批号效期与口径）
			if err := s.addStockTx(ctx, tx, []StockEntry{{
				DrugID: inv.DrugID, LocationID: toLoc, BatchNo: inv.BatchNo,
				ExpiryDate: inv.ExpiryDate, IsSplit: inv.IsSplit, Quantity: it.Quantity, UnitPrice: inv.UnitPrice,
			}}, "transfer_order", 0, enum.TxnTransferIn, operatorID, operatorName); err != nil {
				return err
			}
		}
		return nil
	})
}

// SplitRequest 按盒拆零（整盒行扣减 Packs 盒，拆零行累加 Packs×pack_size）。
type SplitRequest struct {
	InventoryID  int64
	Packs        int64
	ReviewerID   int64 // 复核人（麻精强制双人）
	ReviewerName string
}

// SplitUnitsRequest 按片拆零：开盒零头入账 + 损耗登记。
// 约束：Units + Damaged == Boxes × pack_size（账目平齐）；Damaged 走报损记录。
type SplitUnitsRequest struct {
	InventoryID  int64
	Boxes        int64 // 本次拆几盒
	Units        int64 // 入拆零行片数（零头）
	Damaged      int64 // 破损报损片数
	ReviewerID   int64 // 复核人（麻精强制双人）
	ReviewerName string
}

// Split 按盒拆零（等价 SplitUnits(Packs, Packs×pack_size, 0)）。
func (s *InventoryService) Split(ctx context.Context, req SplitRequest, operatorID int64, operatorName string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		return s.doSplit(ctx, tx, req.InventoryID, req.Packs, -1, 0, operatorID, operatorName, req.ReviewerID, req.ReviewerName)
	})
}

// SplitUnits 按片拆零：支持开盒零头入账与破损报损。
func (s *InventoryService) SplitUnits(ctx context.Context, req SplitUnitsRequest, operatorID int64, operatorName string) error {
	if req.Boxes <= 0 || req.Units < 0 || req.Damaged < 0 {
		return errs.ErrBadRequest
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		return s.doSplit(ctx, tx, req.InventoryID, req.Boxes, req.Units, req.Damaged, operatorID, operatorName, req.ReviewerID, req.ReviewerName)
	})
}

// doSplit 拆零核心：整盒行扣 Boxes 盒 → 拆零行入 Units 片 → 破损 Damaged 片登记报损。
// 拆零行进价 = round(整盒行实际进价 / pack_size)，保证批次成本一致。
// 同时落拆零操作单（split_orders）；麻精药品强制双人复核。
func (s *InventoryService) doSplit(ctx context.Context, tx *gorm.DB, invID, boxes, units, damaged int64, operatorID int64, operatorName string, reviewerID int64, reviewerName string) error {
	invRepo := repository.NewInventoryRepo(tx)
	inv, err := invRepo.LockForUpdate(ctx, invID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.ErrNotFound
		}
		return err
	}
	if err := s.checkLocationNotCounting(ctx, tx, inv.LocationID); err != nil {
		return err
	}
	if inv.IsSplit {
		return errs.ErrBadRequest
	}
	// 只能拆零「未被预占」的整盒，否则破坏 quantity>=reserved 不变量
	if boxes <= 0 || inv.Available() < boxes {
		return errs.ErrStockNotEnough
	}
	drug, err := repository.NewDrugRepo(tx).GetByID(ctx, inv.DrugID)
	if err != nil {
		return err
	}
	if !drug.IsSplitAllowed || drug.PackSize <= 1 {
		return errs.ErrSplitNotAllowed
	}
	// 麻精药品拆零强制双人复核（五专：专人负责 + 双人核对）
	if enum.IsSpecialControlled(drug.SpecialControlType) && (reviewerID == 0 || reviewerID == operatorID) {
		return errs.ErrSplitDualCheck
	}
	if units < 0 {
		// 按盒拆：整盒整入，无损耗
		units = boxes * int64(drug.PackSize)
		damaged = 0
	} else {
		// 账目平齐：开出的盒数所包含片数 = 入拆零行 + 破损
		if units+damaged != boxes*int64(drug.PackSize) {
			return errs.ErrSplitUnbalanced
		}
	}
	// 扣整盒
	ok, err := invRepo.Deduct(ctx, inv.ID, boxes)
	if err != nil {
		return err
	}
	if !ok {
		return errs.ErrNegativeStock
	}
	expiry := inv.ExpiryDate
	before := inv.Quantity
	remarks := ""
	if damaged > 0 {
		remarks = fmt.Sprintf("拆零损耗%d片", damaged)
	}
	out := &model.InventoryTransaction{
		TransactionNo: seq.Next("ITN"),
		DrugID:        inv.DrugID, LocationID: inv.LocationID,
		BatchNo: inv.BatchNo, ExpiryDate: &expiry,
		Quantity: -boxes, IsSplit: false,
		BeforeQuantity: before, AfterQuantity: before - boxes,
		TxnType: enum.TxnSplitOut, RefType: "split", RefID: 0,
		OperatorID: operatorID, OperatorName: operatorName, Remarks: remarks,
	}
	if err := repository.NewInventoryTransactionRepo(tx).Create(ctx, out); err != nil {
		return err
	}
	// 拆零行进价 = round(批次实际进价 / pack_size)；落拆零操作单
	unitPrice := money.Cents(inv.UnitPrice).SplitPrice(drug.PackSize).Int64()
	expiryDate := inv.ExpiryDate
	if err := repository.NewSplitOrderRepo(tx).Create(ctx, &model.SplitOrder{
		SplitNo: seq.Next("SPL"), InventoryID: inv.ID, DrugID: inv.DrugID,
		LocationID: inv.LocationID, BatchNo: inv.BatchNo, ExpiryDate: &expiryDate,
		Boxes: boxes, Units: units, Damaged: damaged, SplitUnitCost: unitPrice,
		OperatorID: operatorID, OperatorName: operatorName,
		ReviewerID: reviewerID, ReviewerName: reviewerName, Remarks: remarks,
	}); err != nil {
		return err
	}
	if units <= 0 {
		return nil
	}
	// 入拆零行：进价取「该批次实际进价」折算，而非主数据价（保证批次成本一致）
	return s.addStockTx(ctx, tx, []StockEntry{{
		DrugID: inv.DrugID, LocationID: inv.LocationID, BatchNo: inv.BatchNo,
		ExpiryDate: inv.ExpiryDate, IsSplit: true, Quantity: units, UnitPrice: unitPrice,
	}}, "split", 0, enum.TxnSplitIn, operatorID, operatorName)
}

// Adjust 库存调整（报损/修正）：正=补入，负=报损。
func (s *InventoryService) Adjust(ctx context.Context, req AdjustRequest, operatorID int64, operatorName string) error {
	if req.Quantity == 0 {
		return errs.ErrBadRequest
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		invRepo := repository.NewInventoryRepo(tx)
		txnRepo := repository.NewInventoryTransactionRepo(tx)
		inv, err := invRepo.LockForUpdate(ctx, req.InventoryID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errs.ErrNotFound
			}
			return err
		}
		if err := s.checkLocationNotCounting(ctx, tx, inv.LocationID); err != nil {
			return err
		}
		// 报损只能针对「未被预占」的部分，防止把已预占库存写没
		if req.Quantity < 0 && inv.Available() < -req.Quantity {
			return errs.ErrStockNotEnough
		}
		var before, after int64
		if req.Quantity > 0 {
			if err := invRepo.Add(ctx, inv.ID, req.Quantity); err != nil {
				return err
			}
			before, after = inv.Quantity, inv.Quantity+req.Quantity
		} else {
			ok, err := invRepo.Deduct(ctx, inv.ID, -req.Quantity)
			if err != nil {
				return err
			}
			if !ok {
				return errs.ErrNegativeStock
			}
			before, after = inv.Quantity, inv.Quantity+req.Quantity
		}
		expiry := inv.ExpiryDate
		txn := &model.InventoryTransaction{
			TransactionNo: seq.Next("ITN"),
			DrugID:        inv.DrugID, LocationID: inv.LocationID,
			BatchNo: inv.BatchNo, ExpiryDate: &expiry,
			Quantity: req.Quantity, IsSplit: inv.IsSplit,
			BeforeQuantity: before, AfterQuantity: after,
			TxnType: enum.TxnWaste, RefType: "adjust", RefID: 0,
			OperatorID: operatorID, OperatorName: operatorName,
			Remarks: req.Reason,
		}
		return txnRepo.Create(ctx, txn)
	})
}

// ---- port.IStockService 实现 ----

// ReserveStock 预占库存（开方即锁定）。
func (s *InventoryService) ReserveStock(ctx context.Context, refType string, refID int64, items []port.ReserveItem) ([]port.ReservationResult, error) {
	results := make([]port.ReservationResult, 0, len(items))
	err := s.db.Transaction(func(tx *gorm.DB) error {
		for _, it := range items {
			res, err := s.reserveItemTx(ctx, tx, refType, refID, it)
			if err != nil {
				return err
			}
			results = append(results, res)
		}
		return nil
	})
	return results, err
}

// reserveItemsTx 批量预占（在调用方事务内执行）。
func (s *InventoryService) reserveItemsTx(ctx context.Context, tx *gorm.DB, refType string, refID int64, items []port.ReserveItem) ([]port.ReservationResult, error) {
	results := make([]port.ReservationResult, 0, len(items))
	for _, it := range items {
		res, err := s.reserveItemTx(ctx, tx, refType, refID, it)
		if err != nil {
			return nil, err
		}
		results = append(results, res)
	}
	return results, nil
}

// reserveItemTx 单个药品的 FEFO 预占，支持三种分配模式（见 port.ReserveItem 注释）。
func (s *InventoryService) reserveItemTx(ctx context.Context, tx *gorm.DB, refType string, refID int64, item port.ReserveItem) (port.ReservationResult, error) {
	res := port.ReservationResult{DrugID: item.DrugID, Quantity: item.Quantity}
	if item.Quantity <= 0 {
		return res, errs.ErrBadRequest
	}
	if err := s.checkLocationNotCounting(ctx, tx, item.LocationID); err != nil {
		return res, err
	}
	invRepo := repository.NewInventoryRepo(tx)
	resvRepo := repository.NewStockReservationRepo(tx)

	switch {
	case item.Mixed:
		// 混合发药：数量按 LDU。优先整盒（整盒部分），零头部分走拆零库存。
		return s.reserveMixedTx(ctx, tx, invRepo, resvRepo, refType, refID, item)
	case item.IsSplit:
		// 强制拆零：仅拆零库存，数量按片。
		return s.reserveSameUnitTx(ctx, tx, invRepo, resvRepo, refType, refID, item, true, item.Quantity)
	default:
		// 整盒口径：仅整盒库存，数量按盒。
		return s.reserveSameUnitTx(ctx, tx, invRepo, resvRepo, refType, refID, item, false, item.Quantity)
	}
}

// reserveSameUnitTx 在同口径批次（整盒或拆零）中 FEFO 预占，数量以该口径计。
func (s *InventoryService) reserveSameUnitTx(ctx context.Context, tx *gorm.DB, invRepo *repository.InventoryRepo, resvRepo *repository.StockReservationRepo, refType string, refID int64, item port.ReserveItem, isSplit bool, limit int64) (port.ReservationResult, error) {
	res := port.ReservationResult{DrugID: item.DrugID, Quantity: item.Quantity}
	batches, err := invRepo.FindAvailableForDispenseUnit(ctx, item.DrugID, item.LocationID, isSplit, todayNow())
	if err != nil {
		return res, err
	}
	reserved, err := s.reserveAcrossBatches(ctx, tx, invRepo, resvRepo, refType, refID, item, batches, limit)
	if err != nil {
		return res, err
	}
	res.Reserved = reserved
	res.Shortage = item.Quantity - res.Reserved
	if res.Shortage < 0 {
		res.Shortage = 0
	}
	return res, nil
}

// reserveMixedTx 混合发药分配（含自动拆零规划）。
// 分配原则（docs/13 §4.4）：
//   - 拆零库存足以覆盖整单 LDU → 整单拆零（优先消耗拆零）
//   - 否则：整盒部分走整盒库存；零头部分先用拆零库存，
//     拆零不足时自动规划「预留 1 盒待拆」（need_split），发药时拆盒入账。
//
// 返回的 Reserved/Shortage 均按 LDU 计（整盒折算 pack 后计入）。
func (s *InventoryService) reserveMixedTx(ctx context.Context, tx *gorm.DB, invRepo *repository.InventoryRepo, resvRepo *repository.StockReservationRepo, refType string, refID int64, item port.ReserveItem) (port.ReservationResult, error) {
	res := port.ReservationResult{DrugID: item.DrugID, Quantity: item.Quantity}
	drug, err := repository.NewDrugRepo(tx).GetByID(ctx, item.DrugID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return res, errs.ErrNotFound
		}
		return res, err
	}
	if !drug.IsSplitAllowed || drug.PackSize <= 1 {
		return res, errs.ErrSplitNotAllowed
	}
	pack := int64(drug.PackSize)

	// 拆零库存总量（用于整单优先拆零判定）
	splitBatches, err := invRepo.FindAvailableForDispenseUnit(ctx, item.DrugID, item.LocationID, true, todayNow())
	if err != nil {
		return res, err
	}
	var splitAvail int64
	for _, b := range splitBatches {
		splitAvail += b.Available()
	}

	if splitAvail >= item.Quantity {
		// 整单由拆零库存覆盖 → 优先拆零（拆零单位即 LDU）
		reserved, err := s.reserveAcrossBatches(ctx, tx, invRepo, resvRepo, refType, refID, item, splitBatches, item.Quantity)
		if err != nil {
			return res, err
		}
		res.Reserved = reserved
	} else {
		boxes := item.Quantity / pack
		remainder := item.Quantity % pack

		// 整盒部分（数量按盒，LDU 折算 ×pack）
		if boxes > 0 {
			wholeBatches, err := invRepo.FindAvailableForDispenseUnit(ctx, item.DrugID, item.LocationID, false, todayNow())
			if err != nil {
				return res, err
			}
			reserved, err := s.reserveAcrossBatches(ctx, tx, invRepo, resvRepo, refType, refID, item, wholeBatches, boxes)
			if err != nil {
				return res, err
			}
			res.Reserved += reserved * pack
		}
		// 零头部分：先用拆零库存
		if remainder > 0 {
			reserved, err := s.reserveAcrossBatches(ctx, tx, invRepo, resvRepo, refType, refID, item, splitBatches, remainder)
			if err != nil {
				return res, err
			}
			res.Reserved += reserved
			// 拆零不足：自动规划预留 1 盒待拆（need_split），拆开补足余片
			if short := remainder - reserved; short > 0 {
				reserved, err := s.reserveOpenBox(ctx, tx, invRepo, resvRepo, refType, refID, item, short)
				if err != nil {
					return res, err
				}
				res.Reserved += reserved
			}
		}
	}
	res.Shortage = item.Quantity - res.Reserved
	if res.Shortage < 0 {
		res.Shortage = 0
	}
	return res, nil
}

// reserveOpenBox 自动拆零规划：预留 1 盒整盒（need_split），拆开后 splitUnits 片用于处方。
// 返回该盒对处方的 LDU 覆盖量（splitUnits）；无可用整盒返回 0（缺货）。
func (s *InventoryService) reserveOpenBox(ctx context.Context, tx *gorm.DB, invRepo *repository.InventoryRepo, resvRepo *repository.StockReservationRepo, refType string, refID int64, item port.ReserveItem, splitUnits int64) (int64, error) {
	wholeBatches, err := invRepo.FindAvailableForDispenseUnit(ctx, item.DrugID, item.LocationID, false, todayNow())
	if err != nil {
		return 0, err
	}
	for _, b := range wholeBatches {
		if b.Available() <= 0 {
			continue
		}
		ok, err := invRepo.Reserve(ctx, b.ID, 1)
		if err != nil {
			return 0, err
		}
		if !ok {
			continue
		}
		resv := &model.StockReservation{
			ReservationNo: seq.Next("RSV"),
			RefType:       refType, RefID: refID, ItemID: item.ItemID,
			InventoryID: b.ID, DrugID: b.DrugID, LocationID: b.LocationID,
			BatchNo: b.BatchNo, ExpiryDate: b.ExpiryDate, IsSplit: false,
			Quantity: 1, NeedSplit: true, SplitUnits: splitUnits, Status: "active",
		}
		if err := resvRepo.Create(ctx, resv); err != nil {
			return 0, err
		}
		return splitUnits, nil
	}
	return 0, nil
}

// reserveAcrossBatches 在同口径批次列表中按 FEFO 顺序预占 limit 数量，返回实际预占数。
// 并发冲突时重读并重试部分预占，避免失败方拿到 0。
func (s *InventoryService) reserveAcrossBatches(ctx context.Context, tx *gorm.DB, invRepo *repository.InventoryRepo, resvRepo *repository.StockReservationRepo, refType string, refID int64, item port.ReserveItem, batches []model.Inventory, limit int64) (int64, error) {
	var reserved int64
	for _, b := range batches {
		if reserved >= limit {
			break
		}
		for {
			need := limit - reserved
			if need <= 0 {
				break
			}
			avail := b.Available()
			if avail <= 0 {
				break
			}
			take := need
			if take > avail {
				take = avail
			}
			ok, err := invRepo.Reserve(ctx, b.ID, take)
			if err != nil {
				return reserved, err
			}
			if ok {
				resv := &model.StockReservation{
					ReservationNo: seq.Next("RSV"),
					RefType:       refType, RefID: refID, ItemID: item.ItemID,
					InventoryID: b.ID, DrugID: b.DrugID, LocationID: b.LocationID,
					BatchNo: b.BatchNo, ExpiryDate: b.ExpiryDate, IsSplit: b.IsSplit,
					Quantity: take, Status: "active",
				}
				if err := resvRepo.Create(ctx, resv); err != nil {
					return reserved, err
				}
				reserved += take
				continue
			}
			// 条件更新失败（并发冲突）：重读最新状态后重试
			nb, err := invRepo.GetByID(ctx, b.ID)
			if err != nil {
				return reserved, err
			}
			b = *nb
			if b.Available() <= 0 {
				break
			}
		}
	}
	return reserved, nil
}

// DispenseAndReduceStock 发药实扣（按该单据的 active 预占核销）。
func (s *InventoryService) DispenseAndReduceStock(ctx context.Context, refType string, refID int64, items []port.DispenseItem) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		return s.consumeTx(ctx, tx, refType, refID, items)
	})
}

// consumeTx 核销预占并实扣库存、写发药流水。
// 以该单据的 active 预占为准核销（分配即定义），items 参数保留供接口契约校验参考。
func (s *InventoryService) consumeTx(ctx context.Context, tx *gorm.DB, refType string, refID int64, _ []port.DispenseItem) error {
	resvRepo := repository.NewStockReservationRepo(tx)
	invRepo := repository.NewInventoryRepo(tx)
	txnRepo := repository.NewInventoryTransactionRepo(tx)
	resvs, err := resvRepo.ListActiveByRef(ctx, refType, refID)
	if err != nil {
		return err
	}
	for _, r := range resvs {
		if err := s.checkLocationNotCounting(ctx, tx, r.LocationID); err != nil {
			return err
		}
		if r.NeedSplit {
			// 自动拆零：整盒拆开，splitUnits 片用于处方，余片入拆零柜
			if err := s.consumeOpenBoxTx(ctx, tx, invRepo, txnRepo, refType, refID, &r); err != nil {
				return err
			}
			continue
		}
		inv, err := invRepo.LockForUpdate(ctx, r.InventoryID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errs.ErrNotFound
			}
			return err
		}
		before := inv.Quantity
		consumed, err := invRepo.Consume(ctx, r.InventoryID, r.Quantity)
		if err != nil {
			return err
		}
		if !consumed {
			return errs.ErrNegativeStock
		}
		if err := resvRepo.UpdateStatus(ctx, r.ID, "consumed"); err != nil {
			return err
		}
		expiry := r.ExpiryDate
		txn := &model.InventoryTransaction{
			TransactionNo: seq.Next("ITN"),
			DrugID:        r.DrugID, LocationID: r.LocationID,
			BatchNo: r.BatchNo, ExpiryDate: &expiry,
			Quantity: -r.Quantity, IsSplit: r.IsSplit,
			BeforeQuantity: before, AfterQuantity: before - r.Quantity,
			TxnType: enum.TxnDispense, RefType: refType, RefID: refID,
		}
		if err := txnRepo.Create(ctx, txn); err != nil {
			return err
		}
	}
	return nil
}

// consumeOpenBoxTx 自动拆零入账：整盒行 -1 → 拆零行 +(pack-splitUnits) → 预占置 consumed。
// 患者取走的 splitUnits 片由整盒扣减涵盖（不重复扣拆零行），发药记录在 buildDispenseRecords 生成。
func (s *InventoryService) consumeOpenBoxTx(ctx context.Context, tx *gorm.DB, invRepo *repository.InventoryRepo, txnRepo *repository.InventoryTransactionRepo, refType string, refID int64, r *model.StockReservation) error {
	inv, err := invRepo.LockForUpdate(ctx, r.InventoryID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.ErrNotFound
		}
		return err
	}
	// 打开该盒：整盒扣减（含预占核销）
	before := inv.Quantity
	consumed, err := invRepo.Consume(ctx, inv.ID, 1)
	if err != nil {
		return err
	}
	if !consumed {
		return errs.ErrNegativeStock
	}
	// 整盒拆分流水（split_out）
	expiry := inv.ExpiryDate
	out := &model.InventoryTransaction{
		TransactionNo: seq.Next("ITN"),
		DrugID:        inv.DrugID, LocationID: inv.LocationID,
		BatchNo: inv.BatchNo, ExpiryDate: &expiry,
		Quantity: -1, IsSplit: false,
		BeforeQuantity: before, AfterQuantity: before - 1,
		TxnType: enum.TxnSplitOut, RefType: refType, RefID: refID,
		Remarks: "自动拆零发放",
	}
	if err := txnRepo.Create(ctx, out); err != nil {
		return err
	}
	// 余片入拆零柜（split_in）
	drug, err := repository.NewDrugRepo(tx).GetByID(ctx, inv.DrugID)
	if err != nil {
		return err
	}
	if remain := int64(drug.PackSize) - r.SplitUnits; remain > 0 {
		unitPrice := money.Cents(inv.UnitPrice).SplitPrice(drug.PackSize).Int64()
		if err := s.addStockTx(ctx, tx, []StockEntry{{
			DrugID: inv.DrugID, LocationID: inv.LocationID, BatchNo: inv.BatchNo,
			ExpiryDate: inv.ExpiryDate, IsSplit: true, Quantity: remain, UnitPrice: unitPrice,
		}}, refType, refID, enum.TxnSplitIn, 0, ""); err != nil {
			return err
		}
	}
	return repository.NewStockReservationRepo(tx).UpdateStatus(ctx, r.ID, "consumed")
}

// CancelReservation 退方/驳回/作废释放预占。
func (s *InventoryService) CancelReservation(ctx context.Context, refType string, refID int64, items []port.ReserveItem) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		return s.cancelTx(ctx, tx, refType, refID, items)
	})
}

// cancelTx 释放预占（该单据全部 active 预占）。
func (s *InventoryService) cancelTx(ctx context.Context, tx *gorm.DB, refType string, refID int64, _ []port.ReserveItem) error {
	resvRepo := repository.NewStockReservationRepo(tx)
	invRepo := repository.NewInventoryRepo(tx)
	resvs, err := resvRepo.ListActiveByRef(ctx, refType, refID)
	if err != nil {
		return err
	}
	for _, r := range resvs {
		released, err := invRepo.ReleaseReserve(ctx, r.InventoryID, r.Quantity)
		if err != nil {
			return err
		}
		if !released {
			return errs.ErrReservationConflict
		}
		if err := resvRepo.UpdateStatus(ctx, r.ID, "released"); err != nil {
			return err
		}
	}
	return nil
}

// CreateLocation 新建库房。
func (s *InventoryService) CreateLocation(ctx context.Context, l *model.InventoryLocation) error {
	return repository.NewLocationRepo(s.db).Create(ctx, l)
}

// UpdateLocation 更新库房。
func (s *InventoryService) UpdateLocation(ctx context.Context, id int64, l *model.InventoryLocation) error {
	existing, err := repository.NewLocationRepo(s.db).GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.ErrNotFound
		}
		return err
	}
	l.ID = existing.ID
	l.CreatedAt = existing.CreatedAt
	return repository.NewLocationRepo(s.db).Update(ctx, l)
}

// ListLocations 库房列表。
func (s *InventoryService) ListLocations(ctx context.Context) ([]model.InventoryLocation, error) {
	return repository.NewLocationRepo(s.db).List(ctx)
}

// GetInventoryDetail 批次库存详情。
func (s *InventoryService) GetInventoryDetail(ctx context.Context, id int64) (*model.Inventory, error) {
	inv, err := repository.NewInventoryRepo(s.db).GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, err
	}
	return inv, nil
}

// ListInventory 库存列表。
func (s *InventoryService) ListInventory(ctx context.Context, f repository.InventoryListFilter, page, pageSize int) ([]model.Inventory, int64, error) {
	return repository.NewInventoryRepo(s.db).List(ctx, f, (page-1)*pageSize, pageSize)
}

// ListTransactions 库存流水。
func (s *InventoryService) ListTransactions(ctx context.Context, f repository.TxnListFilter, page, pageSize int) ([]model.InventoryTransaction, int64, error) {
	return repository.NewInventoryTransactionRepo(s.db).List(ctx, f, (page-1)*pageSize, pageSize)
}

// ListAlerts 预警列表。
func (s *InventoryService) ListAlerts(ctx context.Context, alertType, status string, page, pageSize int) ([]model.StockAlert, int64, error) {
	return repository.NewStockAlertRepo(s.db).List(ctx, alertType, status, (page-1)*pageSize, pageSize)
}

// ResolveAlert 处理预警：action=resolved(已处理)/ignored(忽略)，记录处理人与时间。
func (s *InventoryService) ResolveAlert(ctx context.Context, id int64, action string, operatorID int64, operatorName string) error {
	if action != "resolved" && action != "ignored" {
		return errs.ErrBadRequest
	}
	return repository.NewStockAlertRepo(s.db).UpdateStatus(ctx, id, action, operatorID, operatorName)
}

// ListStocktakes 盘点单列表。
func (s *InventoryService) ListStocktakes(ctx context.Context, status string, page, pageSize int) ([]model.Stocktake, int64, error) {
	return repository.NewStocktakeRepo(s.db).List(ctx, status, (page-1)*pageSize, pageSize)
}

// GetStocktake 盘点单详情（含明细）。
func (s *InventoryService) GetStocktake(ctx context.Context, id int64) (*StocktakeDetail, error) {
	st, err := repository.NewStocktakeRepo(s.db).GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, err
	}
	items, err := repository.NewStocktakeItemRepo(s.db).ListByStocktake(ctx, id)
	if err != nil {
		return nil, err
	}
	return &StocktakeDetail{Stocktake: *st, Items: items}, nil
}

// StocktakeDetail 盘点单详情聚合。
type StocktakeDetail struct {
	model.Stocktake
	Items []model.StocktakeItem `json:"items"`
}

// ListSplitOrders 拆零操作单列表。
func (s *InventoryService) ListSplitOrders(ctx context.Context, drugID, locationID int64, page, pageSize int) ([]model.SplitOrder, int64, error) {
	return repository.NewSplitOrderRepo(s.db).List(ctx, drugID, locationID, (page-1)*pageSize, pageSize)
}

// GetSplitOrder 拆零操作单详情。
func (s *InventoryService) GetSplitOrder(ctx context.Context, id int64) (*model.SplitOrder, error) {
	o, err := repository.NewSplitOrderRepo(s.db).GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, err
	}
	return o, nil
}

// AvailablePacks 某药品某库房的可用整盒数（未预占），用于拆零缺货提示。
func (s *InventoryService) AvailablePacks(ctx context.Context, drugID, locationID int64) (int64, error) {
	rows, err := repository.NewInventoryRepo(s.db).
		FindAvailableForDispenseUnit(ctx, drugID, locationID, false, todayNow())
	if err != nil {
		return 0, err
	}
	var packs int64
	for _, r := range rows {
		packs += r.Available()
	}
	return packs, nil
}

// GetDrugAvailability 查询某药品各库房可用库存（LDU）。
func (s *InventoryService) GetDrugAvailability(ctx context.Context, drugID int64) ([]port.Availability, error) {
	drug, err := repository.NewDrugRepo(s.db).GetByID(ctx, drugID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, err
	}
	var rows []model.Inventory
	if err := s.db.WithContext(ctx).Where("drug_id = ? AND status = 1", drugID).Find(&rows).Error; err != nil {
		return nil, err
	}
	locRepo := repository.NewLocationRepo(s.db)
	byLoc := make(map[int64]*port.Availability)
	for _, r := range rows {
		a, ok := byLoc[r.LocationID]
		if !ok {
			loc, err := locRepo.GetByID(ctx, r.LocationID)
			if err != nil {
				continue
			}
			a = &port.Availability{LocationID: r.LocationID, LocationName: loc.Name}
			byLoc[r.LocationID] = a
		}
		a.Available += rule.ToLDU(r.IsSplit, r.Available(), drug.PackSize)
		if rule.IsNearExpiry(r.ExpiryDate, todayNow(), drug.ExpiryWarningDays) {
			a.ExpirySoon = true
		}
	}
	list := make([]port.Availability, 0, len(byLoc))
	for _, a := range byLoc {
		list = append(list, *a)
	}
	return list, nil
}
