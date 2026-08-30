package service

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"yaofang/internal/domain/enum"
	"yaofang/internal/domain/rule"
	"yaofang/internal/model"
	"yaofang/internal/pkg/errs"
	"yaofang/internal/pkg/seq"
	"yaofang/internal/repository"
)

// POItemInput 采购单明细输入。
type POItemInput struct {
	DrugID    int64 `json:"drug_id"`
	Quantity  int64 `json:"quantity"`   // 基本单位
	UnitPrice int64 `json:"unit_price"` // 分/基本单位
}

// ReceiveItemInput 收货明细输入。
type ReceiveItemInput struct {
	OrderItemID      int64     `json:"order_item_id"`
	ReceivedQuantity int64     `json:"received_quantity"` // 基本单位
	BatchNo          string    `json:"batch_no"`
	ExpiryDate       time.Time `json:"expiry_date"`
	QCResult         int       `json:"qc_result"` // 1合格 2不合格
	QCNotes          string    `json:"qc_notes"`
}

// PurchaseService 采购服务（计划、采购单、收货质检入库）。
type PurchaseService struct {
	db        *gorm.DB
	inventory *InventoryService
}

// NewPurchaseService 构建采购服务。
func NewPurchaseService(db *gorm.DB, inventory *InventoryService) *PurchaseService {
	return &PurchaseService{db: db, inventory: inventory}
}

// CreateOrder 创建采购单（草稿）。
func (s *PurchaseService) CreateOrder(ctx context.Context, supplierID int64, items []POItemInput, expectedAt *time.Time, remarks string, operatorID int64) (*model.PurchaseOrder, error) {
	if len(items) == 0 {
		return nil, errs.ErrBadRequest
	}
	if _, err := repository.NewSupplierRepo(s.db).GetByID(ctx, supplierID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, err
	}
	drugRepo := repository.NewDrugRepo(s.db)
	po := &model.PurchaseOrder{
		PurchaseNo: seq.Next("PO"),
		SupplierID: supplierID, Status: "draft",
		ExpectedAt: expectedAt, CreatedBy: operatorID, Remarks: remarks,
	}
	var total int64
	poItems := make([]*model.PurchaseOrderItem, 0, len(items))
	for _, it := range items {
		drug, err := drugRepo.GetByID(ctx, it.DrugID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errs.ErrDrugNotFound
			}
			return nil, err
		}
		if drug.Status != 1 {
			return nil, errs.ErrDrugInactive
		}
		if it.Quantity <= 0 || it.UnitPrice < 0 {
			return nil, errs.ErrBadRequest
		}
		amount := it.Quantity * it.UnitPrice
		total += amount
		poItems = append(poItems, &model.PurchaseOrderItem{
			DrugID: it.DrugID, Quantity: it.Quantity, UnitPrice: it.UnitPrice, Amount: amount,
		})
	}
	po.TotalAmount = total
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := repository.NewPurchaseOrderRepo(tx).Create(ctx, po); err != nil {
			return err
		}
		for _, it := range poItems {
			it.PurchaseOrderID = po.ID
		}
		return repository.NewPOItemRepo(tx).CreateBatch(ctx, poItems)
	})
	if err != nil {
		return nil, err
	}
	return po, nil
}

// SubmitOrder 提交采购单。
func (s *PurchaseService) SubmitOrder(ctx context.Context, id int64) error {
	return s.setState(ctx, id, "submitted")
}

// CancelOrder 作废采购单（草稿或已提交可作废）。
func (s *PurchaseService) CancelOrder(ctx context.Context, id int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		po, err := repository.NewPurchaseOrderRepo(tx).LockForUpdate(ctx, id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errs.ErrNotFound
			}
			return err
		}
		if po.Status != "draft" && po.Status != "submitted" {
			return errs.ErrStateConflict
		}
		return repository.NewPurchaseOrderRepo(tx).UpdateStatus(ctx, id, "cancelled")
	})
}

func (s *PurchaseService) setState(ctx context.Context, id int64, to string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		po, err := repository.NewPurchaseOrderRepo(tx).LockForUpdate(ctx, id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errs.ErrNotFound
			}
			return err
		}
		if to == "submitted" && po.Status != "draft" {
			return errs.ErrStateConflict
		}
		return repository.NewPurchaseOrderRepo(tx).UpdateStatus(ctx, id, to)
	})
}

// Receive 收货：创建收货单（质检），批次录入。
func (s *PurchaseService) Receive(ctx context.Context, orderID int64, items []ReceiveItemInput, operatorID int64) (*model.PurchaseReceipt, error) {
	if len(items) == 0 {
		return nil, errs.ErrBadRequest
	}
	var receipt *model.PurchaseReceipt
	err := s.db.Transaction(func(tx *gorm.DB) error {
		poRepo := repository.NewPurchaseOrderRepo(tx)
		po, err := poRepo.LockForUpdate(ctx, orderID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errs.ErrNotFound
			}
			return err
		}
		if po.Status != "submitted" {
			return errs.ErrStateConflict
		}
		poItemRepo := repository.NewPOItemRepo(tx)
		poItems, err := poItemRepo.ListByOrder(ctx, orderID)
		if err != nil {
			return err
		}
		byID := make(map[int64]model.PurchaseOrderItem, len(poItems))
		for _, it := range poItems {
			byID[it.ID] = it
		}
		// 先计算金额与明细（receipt.ID 在 Create 后回填）
		var total int64
		receiptItems := make([]*model.PurchaseReceiptItem, 0, len(items))
		for _, it := range items {
			poIt, ok := byID[it.OrderItemID]
			if !ok {
				return errs.ErrBadRequest
			}
			if it.ReceivedQuantity <= 0 || it.ReceivedQuantity > poIt.Quantity-poIt.ReceivedQuantity {
				return errs.ErrReceiveExceeded
			}
			if it.BatchNo == "" || it.ExpiryDate.IsZero() {
				return errs.ErrBadRequest
			}
			total += it.ReceivedQuantity * poIt.UnitPrice
			receiptItems = append(receiptItems, &model.PurchaseReceiptItem{
				OrderItemID: poIt.ID, DrugID: poIt.DrugID,
				OrderedQuantity: poIt.Quantity, ReceivedQuantity: it.ReceivedQuantity,
				BatchNo: it.BatchNo, ExpiryDate: it.ExpiryDate,
				UnitPrice: poIt.UnitPrice, QCResult: it.QCResult, QCNotes: it.QCNotes,
			})
		}
		receipt = &model.PurchaseReceipt{
			ReceiptNo: seq.Next("RCV"), PurchaseOrderID: orderID,
			SupplierID: po.SupplierID, Status: "pending_quality",
			TotalAmount: total, ReceivedBy: operatorID,
		}
		if err := repository.NewPurchaseReceiptRepo(tx).Create(ctx, receipt); err != nil {
			return err
		}
		for _, ri := range receiptItems {
			ri.ReceiptID = receipt.ID
		}
		return repository.NewReceiptItemRepo(tx).CreateBatch(ctx, receiptItems)
	})
	return receipt, err
}

// CompleteReceipt 收货确认入库：质检合格才允许，批次写入库存。
// 质检不合格时在独立事务中落库 qc_failed（避免被回滚）。
//
//nolint:gocyclo // 两阶段质检入库须整体原子完成，拆分会破坏「不合格落库+主事务回滚」语义
func (s *PurchaseService) CompleteReceipt(ctx context.Context, receiptID int64, operatorID int64, operatorName string) error {
	// 阶段一：锁定收货单并校验状态与质检；质检不合格则落库 qc_failed 并返回。
	var qcFailed bool
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		receiptRepo := repository.NewPurchaseReceiptRepo(tx)
		receipt, err := receiptRepo.LockForUpdate(ctx, receiptID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errs.ErrNotFound
			}
			return err
		}
		if receipt.Status != "pending_quality" {
			return errs.ErrStateConflict
		}
		failed, err := repository.NewReceiptItemRepo(tx).HasQCFailed(ctx, receiptID)
		if err != nil {
			return err
		}
		if failed {
			if err := receiptRepo.UpdateStatus(ctx, receiptID, "qc_failed"); err != nil {
				return err
			}
			qcFailed = true
			return nil // 提交状态变更
		}
		// 存在未质检项（qc_result 未登记）时禁止入库：缺省 0 不得视同合格
		uninspected, err := repository.NewReceiptItemRepo(tx).HasUninspected(ctx, receiptID)
		if err != nil {
			return err
		}
		if uninspected {
			return errs.ErrQCNotComplete
		}
		return nil
	}); err != nil {
		return err
	}
	if qcFailed {
		return errs.ErrQCFailed
	}

	// 阶段二：正式入库（再次锁定收货单，防并发）。
	return s.db.Transaction(func(tx *gorm.DB) error {
		receiptRepo := repository.NewPurchaseReceiptRepo(tx)
		receipt, err := receiptRepo.LockForUpdate(ctx, receiptID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errs.ErrNotFound
			}
			return err
		}
		if receipt.Status != "pending_quality" {
			return errs.ErrStateConflict
		}
		itemRepo := repository.NewReceiptItemRepo(tx)
		receiptItems, err := itemRepo.ListByReceipt(ctx, receiptID)
		if err != nil {
			return err
		}
		poItemRepo := repository.NewPOItemRepo(tx)
		entries := make([]StockEntry, 0, len(receiptItems))
		for _, it := range receiptItems {
			if rule.IsExpired(it.ExpiryDate, todayNow()) {
				return errs.ErrExpiredLot
			}
			// 按采购单明细精确归集，并校验未收余额（防止超收/重复入账）
			if it.OrderItemID > 0 {
				poIt, err := poItemRepo.GetByID(ctx, it.OrderItemID)
				if err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						return errs.ErrBadRequest
					}
					return err
				}
				if poIt.ReceivedQuantity+it.ReceivedQuantity > poIt.Quantity {
					return errs.ErrReceiveExceeded
				}
				if err := poItemRepo.UpdateReceived(ctx, poIt.ID, it.ReceivedQuantity); err != nil {
					return err
				}
			}
			entries = append(entries, StockEntry{
				DrugID: it.DrugID, LocationID: defaultReceiveLocation(receipt.SupplierID),
				BatchNo: it.BatchNo, ExpiryDate: it.ExpiryDate,
				IsSplit: false, Quantity: it.ReceivedQuantity, UnitPrice: it.UnitPrice,
			})
		}
		if err := s.inventory.addStockTx(ctx, tx, entries, "purchase_receipt", receiptID, enum.TxnPurchaseIn, operatorID, operatorName); err != nil {
			return err
		}
		// 更新收货单与采购单状态
		if err := receiptRepo.UpdateStatus(ctx, receiptID, "received"); err != nil {
			return err
		}
		if receipt.PurchaseOrderID > 0 {
			poItems, err := poItemRepo.ListByOrder(ctx, receipt.PurchaseOrderID)
			if err != nil {
				return err
			}
			allReceived := true
			for _, it := range poItems {
				if it.ReceivedQuantity < it.Quantity {
					allReceived = false
					break
				}
			}
			poStatus := "partial"
			if allReceived {
				poStatus = "received"
			}
			if err := repository.NewPurchaseOrderRepo(tx).UpdateStatus(ctx, receipt.PurchaseOrderID, poStatus); err != nil {
				return err
			}
		}
		return nil
	})
}

// PurchaseOrderDetail 采购单详情聚合。
type PurchaseOrderDetail struct {
	model.PurchaseOrder
	Items []model.PurchaseOrderItem `json:"items"`
}

// PurchaseReceiptDetail 收货单详情聚合。
type PurchaseReceiptDetail struct {
	model.PurchaseReceipt
	Items []model.PurchaseReceiptItem `json:"items"`
}

// GetOrder 采购单详情。
func (s *PurchaseService) GetOrder(ctx context.Context, id int64) (*PurchaseOrderDetail, error) {
	po, err := repository.NewPurchaseOrderRepo(s.db).GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, err
	}
	items, err := repository.NewPOItemRepo(s.db).ListByOrder(ctx, id)
	if err != nil {
		return nil, err
	}
	return &PurchaseOrderDetail{PurchaseOrder: *po, Items: items}, nil
}

// ListOrders 采购单列表。
func (s *PurchaseService) ListOrders(ctx context.Context, supplierID int64, status string, page, pageSize int) ([]model.PurchaseOrder, int64, error) {
	return repository.NewPurchaseOrderRepo(s.db).List(ctx, supplierID, status, (page-1)*pageSize, pageSize)
}

// GetReceipt 收货单详情。
func (s *PurchaseService) GetReceipt(ctx context.Context, id int64) (*PurchaseReceiptDetail, error) {
	pr, err := repository.NewPurchaseReceiptRepo(s.db).GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, err
	}
	items, err := repository.NewReceiptItemRepo(s.db).ListByReceipt(ctx, id)
	if err != nil {
		return nil, err
	}
	return &PurchaseReceiptDetail{PurchaseReceipt: *pr, Items: items}, nil
}

// ListReceipts 收货单列表。
func (s *PurchaseService) ListReceipts(ctx context.Context, supplierID int64, status string, page, pageSize int) ([]model.PurchaseReceipt, int64, error) {
	return repository.NewPurchaseReceiptRepo(s.db).List(ctx, supplierID, status, (page-1)*pageSize, pageSize)
}

// defaultReceiveLocation 一期收货默认入「中心药库」（ID 由种子数据固定）。
// TODO: 二期扩展为可配置收货库房。
func defaultReceiveLocation(_ int64) int64 { return 1 }
