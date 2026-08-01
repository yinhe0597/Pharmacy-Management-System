//go:build integration

// 拆零专项回归测试：批次成本、按片拆零（零头/损耗）、可配拆零价、平齐校验。
package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"

	"yaofang/internal/model"
	"yaofang/internal/pkg/errs"
	"yaofang/internal/repository"
	"yaofang/internal/service"
)

func splitRowQty(t *testing.T, db *gorm.DB, drugID int64) (qty int64, unitPrice int64) {
	t.Helper()
	rows, _, err := repository.NewInventoryRepo(db).List(context.Background(),
		repository.InventoryListFilter{DrugID: drugID, LocationID: 2}, 0, 20)
	if err != nil {
		t.Fatalf("查询库存失败: %v", err)
	}
	for _, r := range rows {
		if r.IsSplit {
			return r.Quantity, r.UnitPrice
		}
	}
	return 0, 0
}

func wholeRow(t *testing.T, db *gorm.DB, drugID int64) model.Inventory {
	t.Helper()
	rows, _, err := repository.NewInventoryRepo(db).List(context.Background(),
		repository.InventoryListFilter{DrugID: drugID, LocationID: 2}, 0, 20)
	if err != nil {
		t.Fatalf("查询库存失败: %v", err)
	}
	for _, r := range rows {
		if !r.IsSplit {
			return r
		}
	}
	t.Fatal("无整盒行")
	return model.Inventory{}
}

// TestSplitCostFromBatch 拆零行进价应取「批次实际进价」折算，而非主数据拆零进价。
func TestSplitCostFromBatch(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	inv := service.NewInventoryService(db)
	drug := mustCreateDrug(t, db) // 主数据 purchase_price=1800
	// 批次实际进价 2000 分/盒（与主数据不一致）
	if err := inv.StockIn(ctx, []service.StockEntry{{
		DrugID: drug.ID, LocationID: 2, BatchNo: "COST-B",
		ExpiryDate: time.Now().AddDate(1, 0, 0), IsSplit: false, Quantity: 5, UnitPrice: 2000,
	}}, 1, "拆零测试"); err != nil {
		t.Fatalf("入库失败: %v", err)
	}
	whole := wholeRow(t, db, drug.ID)
	if err := inv.Split(ctx, service.SplitRequest{InventoryID: whole.ID, Packs: 1}, 1, "拆零测试"); err != nil {
		t.Fatalf("拆零失败: %v", err)
	}
	_, price := splitRowQty(t, db, drug.ID)
	if price != 83 { // round(2000/24)=83，而非 round(1800/24)=75
		t.Fatalf("拆零行进价应取批次实际进价折算 round(2000/24)=83, got %d", price)
	}
}

// TestSplitUnitsWithDamaged 按片拆零：开盒零头入账 + 破损报损，账目平齐。
func TestSplitUnitsWithDamaged(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	inv := service.NewInventoryService(db)
	drug := mustCreateDrug(t, db)
	if err := inv.StockIn(ctx, []service.StockEntry{{
		DrugID: drug.ID, LocationID: 2, BatchNo: "UNIT-B",
		ExpiryDate: time.Now().AddDate(1, 0, 0), IsSplit: false, Quantity: 5, UnitPrice: 1800,
	}}, 1, "拆零测试"); err != nil {
		t.Fatalf("入库失败: %v", err)
	}
	whole := wholeRow(t, db, drug.ID)
	// 拆 1 盒：20 片入拆零行、4 片破损 → 平齐
	if err := inv.SplitUnits(ctx, service.SplitUnitsRequest{InventoryID: whole.ID, Boxes: 1, Units: 20, Damaged: 4}, 1, "拆零测试"); err != nil {
		t.Fatalf("按片拆零失败: %v", err)
	}
	splitQty, _ := splitRowQty(t, db, drug.ID)
	if splitQty != 20 {
		t.Fatalf("拆零行应为20片, got %d", splitQty)
	}
	if w := wholeRow(t, db, drug.ID); w.Quantity != 4 {
		t.Fatalf("整盒行应剩4盒, got %d", w.Quantity)
	}
	// 不平齐应被拦截
	if err := inv.SplitUnits(ctx, service.SplitUnitsRequest{InventoryID: whole.ID, Boxes: 1, Units: 20, Damaged: 5}, 1, "拆零测试"); err == nil {
		t.Fatal("不平齐的拆零应被拦截")
	} else {
		var e *errs.Error
		if !errors.As(err, &e) || e.Code != 2012 {
			t.Fatalf("期望不平齐错误码2012, got %v", err)
		}
	}
}

// TestSplitFullBoxAsUnits 按片拆零整盒（units=pack_size, damaged=0）等价按盒拆。
func TestSplitFullBoxAsUnits(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	inv := service.NewInventoryService(db)
	drug := mustCreateDrug(t, db)
	if err := inv.StockIn(ctx, []service.StockEntry{{
		DrugID: drug.ID, LocationID: 2, BatchNo: "FULL-B",
		ExpiryDate: time.Now().AddDate(1, 0, 0), IsSplit: false, Quantity: 3, UnitPrice: 1800,
	}}, 1, "拆零测试"); err != nil {
		t.Fatalf("入库失败: %v", err)
	}
	whole := wholeRow(t, db, drug.ID)
	if err := inv.SplitUnits(ctx, service.SplitUnitsRequest{InventoryID: whole.ID, Boxes: 1, Units: 24, Damaged: 0}, 1, "拆零测试"); err != nil {
		t.Fatalf("按片整盒拆零失败: %v", err)
	}
	splitQty, _ := splitRowQty(t, db, drug.ID)
	if splitQty != 24 {
		t.Fatalf("拆零行应为24片, got %d", splitQty)
	}
}

// TestMixedDispense 混合发药：36 片 = 1 整盒 + 12 片拆零，计价精确、跨口径分配、发药记录按形态计价。
func TestMixedDispense(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	inv := service.NewInventoryService(db)
	presc := service.NewPrescriptionService(db, inv, service.NewSpecialDrugService(db))
	drug := mustCreateDrug(t, db) // 24片/盒，零售 2400 分，拆零 100 分/片

	// 3 整盒 + 拆零 12 片
	if err := inv.StockIn(ctx, []service.StockEntry{{
		DrugID: drug.ID, LocationID: 2, BatchNo: "MIX-B",
		ExpiryDate: time.Now().AddDate(1, 0, 0), IsSplit: false, Quantity: 3, UnitPrice: 1800,
	}}, 1, "混合测试"); err != nil {
		t.Fatalf("入库失败: %v", err)
	}
	whole := wholeRow(t, db, drug.ID)
	if err := inv.SplitUnits(ctx, service.SplitUnitsRequest{InventoryID: whole.ID, Boxes: 1, Units: 12, Damaged: 12}, 1, "混合测试"); err != nil {
		t.Fatalf("拆零失败: %v", err)
	}
	// 整盒 2 盒 + 拆零 12 片

	// 处方 36 片（1 盒 + 12 片），混合发药（IsSplit=false）
	p, err := presc.Create(ctx, service.PrescriptionInput{
		PatientName: "混合患者",
		Items:       []service.PrescriptionItemInput{{DrugID: drug.ID, Quantity: 36}},
	})
	if err != nil {
		t.Fatalf("创建处方失败: %v", err)
	}
	// 计价：1×2400 + 12×100 = 3600
	items, _ := repository.NewPrescriptionItemRepo(db).ListByPrescription(ctx, p.ID)
	if len(items) != 1 || items[0].Amount != 3600 {
		t.Fatalf("混合计价不符: amount=%d want 3600", items[0].Amount)
	}
	if items[0].Quantity != 36 || items[0].UnitPrice != 100 || items[0].RetailPrice != 2400 {
		t.Fatalf("明细快照不符: qty=%d unit=%d retail=%d", items[0].Quantity, items[0].UnitPrice, items[0].RetailPrice)
	}

	// 预占：1 盒（整盒行）+ 12 片（拆零行）
	if err := presc.Submit(ctx, p.ID, 1, "药师A"); err != nil {
		t.Fatalf("提交失败: %v", err)
	}
	resvs, _ := repository.NewStockReservationRepo(db).ListActiveByRef(ctx, "prescription", p.ID)
	var wholeResv, splitResv int64
	for _, r := range resvs {
		if r.IsSplit {
			splitResv += r.Quantity
		} else {
			wholeResv += r.Quantity
		}
	}
	if wholeResv != 1 || splitResv != 12 {
		t.Fatalf("混合分配不符: 整盒=%d want 1, 拆零=%d want 12", wholeResv, splitResv)
	}

	if err := presc.Review(ctx, p.ID, service.AuditInput{Action: "pass"}, 2, "药师B"); err != nil {
		t.Fatalf("审核失败: %v", err)
	}
	if err := presc.Dispense(ctx, p.ID, 3, "调配员"); err != nil {
		t.Fatalf("调配失败: %v", err)
	}
	if err := presc.ConfirmDispense(ctx, p.ID, 4, "核对员"); err != nil {
		t.Fatalf("发药确认失败: %v", err)
	}

	// 发药记录：整盒 1 盒按盒价 2400，拆零 12 片按拆零价 100
	records, _ := repository.NewDispenseRecordRepo(db).ListByPrescription(ctx, p.ID)
	var totalAmount int64
	for _, rec := range records {
		if !rec.IsSplit && (rec.Quantity != 1 || rec.UnitPrice != 2400) {
			t.Fatalf("整盒发药记录不符: qty=%d price=%d", rec.Quantity, rec.UnitPrice)
		}
		if rec.IsSplit && (rec.Quantity != 12 || rec.UnitPrice != 100) {
			t.Fatalf("拆零发药记录不符: qty=%d price=%d", rec.Quantity, rec.UnitPrice)
		}
		totalAmount += rec.Amount
	}
	if totalAmount != 3600 {
		t.Fatalf("发药记录金额合计不符: %d want 3600", totalAmount)
	}
	// 库存核销：整盒剩 1 盒，拆零归 0
	whole = wholeRow(t, db, drug.ID)
	if whole.Quantity != 1 {
		t.Fatalf("整盒应剩1盒, got %d", whole.Quantity)
	}
	if splitQty, _ := splitRowQty(t, db, drug.ID); splitQty != 0 {
		t.Fatalf("拆零应归0, got %d", splitQty)
	}
}

// TestConfigurableSplitPrice 拆零零售价可显式配置覆盖，不强制公式推算。
func TestConfigurableSplitPrice(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	drugSvc := service.NewDrugService(db)
	d := &model.Drug{
		Code: "PRICE1", GenericName: "可配价药", DosageForm: "片剂",
		Specification: "0.5g", Manufacturer: "拆零测试药厂",
		BaseUnit: "盒", SplitUnit: "片", PackSize: 24, IsSplitAllowed: true,
		RetailPrice: 2400, PurchasePrice: 1800, SplitRetailPrice: 120, // 显式配置 120 分/片（分摊损耗）
		Status: 1,
	}
	if err := drugSvc.Create(ctx, d); err != nil {
		t.Fatalf("创建药品失败: %v", err)
	}
	if d.SplitRetailPrice != 120 {
		t.Fatalf("显式配置的拆零零售价应保留 120, got %d", d.SplitRetailPrice)
	}
	if d.SplitPurchasePrice != 75 { // 进价仍公式推导 round(1800/24)=75
		t.Fatalf("拆零进价仍应公式推算 75, got %d", d.SplitPurchasePrice)
	}
}
