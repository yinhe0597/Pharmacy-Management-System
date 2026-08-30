package service

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"yaofang/internal/model"
	"yaofang/internal/pkg/errs"
	"yaofang/internal/pkg/seq"
	"yaofang/internal/repository"
)

// RequisitionOrderItemInput 领用/补发明细（quantity 按 LDU 拆零单位）。
type RequisitionOrderItemInput struct {
	DrugID   int64 `json:"drug_id"`
	Quantity int64 `json:"quantity"` // LDU
}

// RequisitionOrderInput 领用/补发登记单输入。
type RequisitionOrderInput struct {
	LocationID int64                       `json:"location_id"`
	Purpose    string                      `json:"purpose"` // supplement(补发)/clinical(临床领用)/other
	Reason     string                      `json:"reason"`
	Items      []RequisitionOrderItemInput `json:"items"`
}

// RequisitionOrderDetail 领用/补发登记单详情。
type RequisitionOrderDetail struct {
	model.RequisitionOrder
	Items []model.RequisitionOrderItem `json:"items"`
}

// CreateRequisitionOrder 创建领用/补发登记单：多明细 LDU 口径 FEFO 扣减 + 落单（docs/18）。
// 分配顺序：优先拆零批次（近效期先出），不足再整盒（整盒按 pack_size 折算，允许向上取整到整盒）。
func (s *InventoryService) CreateRequisitionOrder(ctx context.Context, input RequisitionOrderInput, operatorID int64, operatorName string) (*model.RequisitionOrder, error) {
	if input.LocationID <= 0 || len(input.Items) == 0 {
		return nil, errs.ErrBadRequest
	}
	if input.Purpose == "" {
		input.Purpose = "supplement"
	}
	var order *model.RequisitionOrder
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.checkLocationNotCounting(ctx, tx, input.LocationID); err != nil {
			return err
		}
		order = &model.RequisitionOrder{
			RequisitionNo: seq.Next("REQ"),
			LocationID:    input.LocationID,
			Purpose:       input.Purpose,
			Reason:        input.Reason,
			OperatorID:    operatorID,
			OperatorName:  operatorName,
			Status:        "completed",
		}
		if err := repository.NewRequisitionOrderRepo(tx).Create(ctx, order); err != nil {
			return err
		}
		invRepo := repository.NewInventoryRepo(tx)
		txnRepo := repository.NewInventoryTransactionRepo(tx)
		var items []*model.RequisitionOrderItem
		for _, in := range input.Items {
			if in.DrugID <= 0 || in.Quantity <= 0 {
				return errs.ErrBadRequest
			}
			got, err := s.requisitionDrugTx(ctx, tx, invRepo, txnRepo, order, in, operatorID, operatorName)
			if err != nil {
				return err
			}
			items = append(items, got...)
		}
		return repository.NewRequisitionOrderRepo(tx).CreateItems(ctx, items)
	})
	if err != nil {
		return nil, err
	}
	return order, nil
}

// requisitionDrugTx 单个药品按 LDU FEFO 扣减（拆零优先，整盒按 pack_size 折算）。
func (s *InventoryService) requisitionDrugTx(ctx context.Context, tx *gorm.DB, invRepo *repository.InventoryRepo, txnRepo *repository.InventoryTransactionRepo, order *model.RequisitionOrder, in RequisitionOrderItemInput, operatorID int64, operatorName string) ([]*model.RequisitionOrderItem, error) {
	drug, err := repository.NewDrugRepo(tx).GetByID(ctx, in.DrugID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrDrugNotFound
		}
		return nil, err
	}
	pack := int64(drug.PackSize)
	if pack <= 0 {
		pack = 1
	}
	var items []*model.RequisitionOrderItem
	remaining := in.Quantity
	takeFromBatch := func(b model.Inventory, take int64) error {
		// 可用量扣减（排除预占）：领用不得吞掉处方已预占库存
		ok, err := invRepo.DeductAvailable(ctx, b.ID, take)
		if err != nil {
			return err
		}
		if !ok {
			return errs.ErrNegativeStock
		}
		before := b.Quantity
		if err := txnRepo.Create(ctx, &model.InventoryTransaction{
			TransactionNo: seq.Next("ITN"),
			DrugID:        in.DrugID, LocationID: order.LocationID,
			BatchNo: b.BatchNo, ExpiryDate: &b.ExpiryDate,
			Quantity: -take, IsSplit: b.IsSplit,
			BeforeQuantity: before, AfterQuantity: before - take,
			TxnType: "requisition", RefType: "requisition_order", RefID: order.ID,
			OperatorID: operatorID, OperatorName: operatorName, Remarks: "领用/补发 " + order.Purpose,
		}); err != nil {
			return err
		}
		items = append(items, &model.RequisitionOrderItem{
			RequisitionOrderID: order.ID, DrugID: in.DrugID, BatchNo: b.BatchNo,
			IsSplit: b.IsSplit, Quantity: take, UnitPrice: b.UnitPrice,
		})
		return nil
	}
	// 1) 拆零批次（单位即 LDU）
	splitBatches, err := invRepo.FindAvailableForDispenseUnit(ctx, in.DrugID, order.LocationID, true, todayNow())
	if err != nil {
		return nil, err
	}
	for i := range splitBatches {
		if remaining <= 0 {
			break
		}
		avail := splitBatches[i].Available()
		if avail <= 0 {
			continue
		}
		take := remaining
		if take > avail {
			take = avail
		}
		if err := takeFromBatch(splitBatches[i], take); err != nil {
			return nil, err
		}
		remaining -= take
	}
	// 2) 整盒批次（1 盒 = pack_size LDU，向上取整到整盒）
	if remaining > 0 {
		wholeBatches, err := invRepo.FindAvailableForDispenseUnit(ctx, in.DrugID, order.LocationID, false, todayNow())
		if err != nil {
			return nil, err
		}
		for i := range wholeBatches {
			if remaining <= 0 {
				break
			}
			availBoxes := wholeBatches[i].Available()
			if availBoxes <= 0 {
				continue
			}
			needBoxes := (remaining + pack - 1) / pack
			take := needBoxes
			if take > availBoxes {
				take = availBoxes
			}
			if err := takeFromBatch(wholeBatches[i], take); err != nil {
				return nil, err
			}
			remaining -= take * pack
		}
	}
	if remaining > 0 {
		return nil, errs.ErrStockNotEnough
	}
	return items, nil
}

// ListRequisitionOrders 领用/补发登记单列表。
func (s *InventoryService) ListRequisitionOrders(ctx context.Context, locationID int64, purpose string, page, pageSize int) ([]model.RequisitionOrder, int64, error) {
	return repository.NewRequisitionOrderRepo(s.db).List(ctx, locationID, purpose, (page-1)*pageSize, pageSize)
}

// GetRequisitionOrder 领用/补发登记单详情。
func (s *InventoryService) GetRequisitionOrder(ctx context.Context, id int64) (*RequisitionOrderDetail, error) {
	o, err := repository.NewRequisitionOrderRepo(s.db).GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, err
	}
	items, err := repository.NewRequisitionOrderRepo(s.db).ListItems(ctx, id)
	if err != nil {
		return nil, err
	}
	return &RequisitionOrderDetail{RequisitionOrder: *o, Items: items}, nil
}
