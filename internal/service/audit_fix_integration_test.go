//go:build integration

package service_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"gorm.io/gorm"

	"yaofang/internal/model"
	"yaofang/internal/pkg/errs"
	"yaofang/internal/repository"
	"yaofang/internal/service"
)

// 本文件覆盖 2026-09 审计修复项的**真实数据库**验证（需 PostgreSQL：go test -tags=integration）：
//  1. 并发入库同一新批次 → 原子 upsert（ON CONFLICT）不因唯一键冲突整单失败；
//  2. 采购收货防超收 → 创建期在途校验 + 完成期条件更新双护栏；
//  3. 盘点差异口径 → difference = 实盘 - 账面。

func mustCreateSupplier(t *testing.T, db *gorm.DB) *model.Supplier {
	t.Helper()
	sp := &model.Supplier{
		Code: "SUP-IT-01", Name: "集成测试供应商", ContactPerson: "张供应",
		Phone: "13800000000", Status: 1,
	}
	if err := service.NewSupplierService(db).Create(context.Background(), sp); err != nil {
		t.Fatalf("创建供应商失败: %v", err)
	}
	return sp
}

// TestConcurrentStockInSameNewBatch 并发入库同一「尚不存在」的批次：
// 全部入库必须成功且数量精确累加（旧实现「Create 失败→FindByKey 回退」在 PostgreSQL 下
// 唯一键冲突后事务已 aborted（25P02），回退查询必然失败，会出现整单失败/数量丢失）。
func TestConcurrentStockInSameNewBatch(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	inv := service.NewInventoryService(db)
	drug := mustCreateDrug(t, db)

	const goroutines = 4
	const perEntry = 5
	const batchNo = "BATCH-CONC-1"

	start := make(chan struct{})
	errsCh := make([]error, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			<-start
			errsCh[idx] = inv.StockIn(ctx, []service.StockEntry{{
				DrugID: drug.ID, LocationID: 2, BatchNo: batchNo,
				ExpiryDate: time.Now().AddDate(1, 0, 0), IsSplit: false,
				Quantity: perEntry, UnitPrice: drug.PurchasePrice,
			}}, 1, "并发入库")
		}(i)
	}
	close(start)
	wg.Wait()

	for i, err := range errsCh {
		if err != nil {
			t.Fatalf("第 %d 个并发入库失败（upsert 应保证全部成功）: %v", i, err)
		}
	}

	var row model.Inventory
	if err := db.WithContext(ctx).
		Where("drug_id = ? AND location_id = ? AND batch_no = ? AND is_split = ?", drug.ID, 2, batchNo, false).
		First(&row).Error; err != nil {
		t.Fatalf("查询库存行失败: %v", err)
	}
	if want := int64(goroutines * perEntry); row.Quantity != want {
		t.Fatalf("并发入库数量错误: got %d want %d（存在丢失更新）", row.Quantity, want)
	}

	// 流水条数与数量一致性：条数 = goroutines，最后一条的 after 应为总量
	var txns []model.InventoryTransaction
	if err := db.WithContext(ctx).
		Where("drug_id = ? AND batch_no = ?", drug.ID, batchNo).
		Order("id ASC").Find(&txns).Error; err != nil {
		t.Fatalf("查询流水失败: %v", err)
	}
	if len(txns) != goroutines {
		t.Fatalf("入库流水条数错误: got %d want %d", len(txns), goroutines)
	}
	var sum int64
	for _, tx := range txns {
		sum += tx.Quantity
	}
	if sum != int64(goroutines*perEntry) {
		t.Fatalf("流水数量合计错误: got %d want %d", sum, goroutines*perEntry)
	}
}

// TestPurchaseReceiveOverReceiveGuards 采购收货双重防超收：
// 阶段一在创建收货单时扣减在途（pending_quality）数量；阶段二在完成入库时以条件更新
// （received_quantity + ? <= quantity）原子校验，任何路径都不得使累计收货超过订购量。
func TestPurchaseReceiveOverReceiveGuards(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	inv := service.NewInventoryService(db)
	purchase := service.NewPurchaseService(db, inv)
	drug := mustCreateDrug(t, db)
	sup := mustCreateSupplier(t, db)

	const ordered = 10
	po, err := purchase.CreateOrder(ctx, sup.ID,
		[]service.POItemInput{{DrugID: drug.ID, Quantity: ordered, UnitPrice: drug.PurchasePrice}},
		nil, "集成测试：防超收", 1)
	if err != nil {
		t.Fatalf("创建采购单失败: %v", err)
	}
	if err := purchase.SubmitOrder(ctx, po.ID); err != nil {
		t.Fatalf("提交采购单失败: %v", err)
	}
	detail, err := purchase.GetOrder(ctx, po.ID)
	if err != nil || len(detail.Items) == 0 {
		t.Fatalf("查询采购单明细失败: %v", err)
	}
	itemID := detail.Items[0].ID
	expiry := time.Now().AddDate(1, 0, 0)

	newReceiptInput := func(qty int64, batch string) []service.ReceiveItemInput {
		return []service.ReceiveItemInput{{
			OrderItemID: itemID, ReceivedQuantity: qty, BatchNo: batch,
			ExpiryDate: expiry, QCResult: 1,
		}}
	}

	// 第一张：6 件（在途）
	r1, err := purchase.Receive(ctx, po.ID, newReceiptInput(6, "RCV-GUARD-1"), 1)
	if err != nil {
		t.Fatalf("第一张收货单创建失败: %v", err)
	}
	// 第二张：5 件 → 6+5=11 > 10，应在创建期被在途校验拒绝
	if _, err := purchase.Receive(ctx, po.ID, newReceiptInput(5, "RCV-GUARD-2"), 1); !errors.Is(err, errs.ErrReceiveExceeded) {
		t.Fatalf("在途超收应被拒绝(ErrReceiveExceeded), got %v", err)
	}
	// 第二张：4 件 → 恰好收满，允许创建
	r2, err := purchase.Receive(ctx, po.ID, newReceiptInput(4, "RCV-GUARD-3"), 1)
	if err != nil {
		t.Fatalf("合法收货单创建失败: %v", err)
	}

	// 完成两张收货单（6 + 4 = 10）
	if err := purchase.CompleteReceipt(ctx, r1.ID, 1, "集成测试"); err != nil {
		t.Fatalf("完成收货单 1 失败: %v", err)
	}
	if err := purchase.CompleteReceipt(ctx, r2.ID, 1, "集成测试"); err != nil {
		t.Fatalf("完成收货单 2 失败: %v", err)
	}

	var poItem model.PurchaseOrderItem
	if err := db.WithContext(ctx).First(&poItem, itemID).Error; err != nil {
		t.Fatalf("查询采购明细失败: %v", err)
	}
	if poItem.ReceivedQuantity != ordered {
		t.Fatalf("已收数量错误: got %d want %d", poItem.ReceivedQuantity, ordered)
	}

	// 绕过创建期校验，直接落一张「已申报 1 件」的待质检收货单，模拟并发/历史脏数据，
	// 验证完成期条件更新护栏：必须拒绝且不得入库。
	rogueReceipt := &model.PurchaseReceipt{
		ReceiptNo: "RCV-ROGUE-1", PurchaseOrderID: po.ID, SupplierID: sup.ID,
		Status: "pending_quality", TotalAmount: drug.PurchasePrice, ReceivedBy: 1,
	}
	if err := db.WithContext(ctx).Create(rogueReceipt).Error; err != nil {
		t.Fatalf("构造越界收货单失败: %v", err)
	}
	rogueItem := &model.PurchaseReceiptItem{
		ReceiptID: rogueReceipt.ID, OrderItemID: itemID, DrugID: drug.ID,
		OrderedQuantity: ordered, ReceivedQuantity: 1, BatchNo: "RCV-ROGUE-1",
		ExpiryDate: expiry, UnitPrice: drug.PurchasePrice, QCResult: 1,
	}
	if err := db.WithContext(ctx).Create(rogueItem).Error; err != nil {
		t.Fatalf("构造越界收货明细失败: %v", err)
	}

	before := inventoryQuantity(t, db, drug.ID, "RCV-ROGUE-1")
	if err := purchase.CompleteReceipt(ctx, rogueReceipt.ID, 1, "集成测试"); !errors.Is(err, errs.ErrReceiveExceeded) {
		t.Fatalf("完成期超收应被拒绝(ErrReceiveExceeded), got %v", err)
	}
	after := inventoryQuantity(t, db, drug.ID, "RCV-ROGUE-1")
	if after != before {
		t.Fatalf("超收被拒后库存不得变化: before=%d after=%d", before, after)
	}
	if err := db.WithContext(ctx).First(&poItem, itemID).Error; err != nil {
		t.Fatalf("复查采购明细失败: %v", err)
	}
	if poItem.ReceivedQuantity != ordered {
		t.Fatalf("超收被拒后已收数量不得变化: got %d want %d", poItem.ReceivedQuantity, ordered)
	}
	// 收货单应保持待质检（未入库）
	var reloaded model.PurchaseReceipt
	if err := db.WithContext(ctx).First(&reloaded, rogueReceipt.ID).Error; err != nil {
		t.Fatalf("复查收货单失败: %v", err)
	}
	if reloaded.Status != "pending_quality" {
		t.Fatalf("超收被拒后收货单状态应保持 pending_quality, got %s", reloaded.Status)
	}
}

// TestStocktakeDifferencePersisted 盘点差异落库口径 = 实盘 - 账面。
func TestStocktakeDifferencePersisted(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	inv := service.NewInventoryService(db)
	drug := mustCreateDrug(t, db)

	const batchNo = "BATCH-STK-1"
	const bookQty = 10
	const countedQty = 7
	if err := inv.StockIn(ctx, []service.StockEntry{{
		DrugID: drug.ID, LocationID: 2, BatchNo: batchNo,
		ExpiryDate: time.Now().AddDate(1, 0, 0), IsSplit: false,
		Quantity: bookQty, UnitPrice: drug.PurchasePrice,
	}}, 1, "集成测试"); err != nil {
		t.Fatalf("入库失败: %v", err)
	}

	st, err := inv.CreateStocktake(ctx, 2, 1, 1)
	if err != nil {
		t.Fatalf("创建盘点单失败: %v", err)
	}
	if err := inv.StartStocktake(ctx, st.ID); err != nil {
		t.Fatalf("开始盘点失败: %v", err)
	}
	items, err := repository.NewStocktakeItemRepo(db).ListByStocktake(ctx, st.ID)
	if err != nil {
		t.Fatalf("查询盘点明细失败: %v", err)
	}
	var target *model.StocktakeItem
	for i := range items {
		if items[i].BatchNo == batchNo {
			target = &items[i]
			break
		}
	}
	if target == nil {
		t.Fatalf("盘点明细中未找到批次 %s", batchNo)
	}
	if target.BookQuantity != bookQty {
		t.Fatalf("账面数量快照错误: got %d want %d", target.BookQuantity, bookQty)
	}

	if err := inv.EnterCounted(ctx, st.ID, []service.CountedItem{{ItemID: target.ID, CountedQuantity: countedQty}}); err != nil {
		t.Fatalf("录入实盘失败: %v", err)
	}

	var reloaded model.StocktakeItem
	if err := db.WithContext(ctx).First(&reloaded, target.ID).Error; err != nil {
		t.Fatalf("复查盘点明细失败: %v", err)
	}
	if reloaded.CountedQuantity != countedQty {
		t.Fatalf("实盘数量错误: got %d want %d", reloaded.CountedQuantity, countedQty)
	}
	if want := int64(countedQty - bookQty); reloaded.Difference != want {
		t.Fatalf("盘点差异口径错误: got %d want %d（应为 实盘 - 账面）", reloaded.Difference, want)
	}

	// 结束盘点：处于 counting 的库房会禁止出入库，不清理会影响后续用例（测试隔离）
	if err := inv.CancelStocktake(ctx, st.ID); err != nil {
		t.Fatalf("取消盘点失败（应可在 counting 状态下取消）: %v", err)
	}
}

// inventoryQuantity 返回指定药品+批次的库存数量（不存在返回 0）。
func inventoryQuantity(t *testing.T, db *gorm.DB, drugID int64, batchNo string) int64 {
	t.Helper()
	var row model.Inventory
	err := db.WithContext(context.Background()).
		Where("drug_id = ? AND batch_no = ?", drugID, batchNo).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0
	}
	if err != nil {
		t.Fatalf("查询库存失败: %v", err)
	}
	return row.Quantity
}
