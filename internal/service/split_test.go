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

// splitRowQty 汇总某药品全部拆零行的数量与（首行）单价。可存在多条拆零批次。
func splitRowQty(t *testing.T, db *gorm.DB, drugID int64) (qty int64, unitPrice int64) {
	t.Helper()
	rows, _, err := repository.NewInventoryRepo(db).List(context.Background(),
		repository.InventoryListFilter{DrugID: drugID, LocationID: 2}, 0, 20)
	if err != nil {
		t.Fatalf("查询库存失败: %v", err)
	}
	for _, r := range rows {
		if r.IsSplit {
			qty += r.Quantity
			if unitPrice == 0 {
				unitPrice = r.UnitPrice
			}
		}
	}
	return qty, unitPrice
}

func wholeRow(t *testing.T, db *gorm.DB, drugID int64) repository.InventoryRow {
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
	return repository.InventoryRow{}
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
	presc := service.NewPrescriptionService(db, inv, service.NewSpecialDrugService(db), nil, nil, service.NewClinicalService(db))
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

	if _, err := presc.Review(ctx, p.ID, service.AuditInput{Action: "pass"}, 2, "药师B"); err != nil {
		t.Fatalf("审核失败: %v", err)
	}
	if err := presc.Dispense(ctx, p.ID, 3, "调配员", "pharmacist"); err != nil {
		t.Fatalf("调配失败: %v", err)
	}
	if err := presc.ConfirmDispense(ctx, p.ID, 4, "核对员", "pharmacist"); err != nil {
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

// TestAutoSplit 自动拆零：无拆零库存时，36 片混合处方自动规划预留 1 盒待拆，发药自动拆盒入账。
func TestAutoSplit(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	inv := service.NewInventoryService(db)
	presc := service.NewPrescriptionService(db, inv, service.NewSpecialDrugService(db), nil, nil, service.NewClinicalService(db))
	drug := mustCreateDrug(t, db)

	// 仅 2 整盒，无拆零库存
	if err := inv.StockIn(ctx, []service.StockEntry{{
		DrugID: drug.ID, LocationID: 2, BatchNo: "AUTO-B",
		ExpiryDate: time.Now().AddDate(1, 0, 0), IsSplit: false, Quantity: 2, UnitPrice: 1800,
	}}, 1, "自动拆零测试"); err != nil {
		t.Fatalf("入库失败: %v", err)
	}

	p, err := presc.Create(ctx, service.PrescriptionInput{
		PatientName: "自动拆零患者",
		Items:       []service.PrescriptionItemInput{{DrugID: drug.ID, Quantity: 36}}, // 1盒+12片
	})
	if err != nil {
		t.Fatalf("创建处方失败: %v", err)
	}

	// 提交：应规划 1 盒正常 + 1 盒待拆（split_units=12），无缺货
	if err := presc.Submit(ctx, p.ID, 1, "药师A"); err != nil {
		t.Fatalf("提交失败: %v", err)
	}
	resvs, _ := repository.NewStockReservationRepo(db).ListActiveByRef(ctx, "prescription", p.ID)
	var normalBoxes, needSplit int64
	var splitUnits int64
	for _, r := range resvs {
		if r.NeedSplit {
			needSplit++
			splitUnits = r.SplitUnits
		} else {
			normalBoxes += r.Quantity
		}
	}
	if normalBoxes != 1 || needSplit != 1 || splitUnits != 12 {
		t.Fatalf("自动拆零规划不符: normalBoxes=%d needSplit=%d splitUnits=%d", normalBoxes, needSplit, splitUnits)
	}

	if _, err := presc.Review(ctx, p.ID, service.AuditInput{Action: "pass"}, 2, "药师B"); err != nil {
		t.Fatalf("审核失败: %v", err)
	}
	if err := presc.Dispense(ctx, p.ID, 3, "调配员", "pharmacist"); err != nil {
		t.Fatalf("调配失败: %v", err)
	}
	if err := presc.ConfirmDispense(ctx, p.ID, 4, "核对员", "pharmacist"); err != nil {
		t.Fatalf("发药确认失败: %v", err)
	}

	// 发药后：整盒 -2（1 正常 + 1 待拆），拆零柜 +12（拆开的余片）
	whole := wholeRow(t, db, drug.ID)
	if whole.Quantity != 0 {
		t.Fatalf("整盒应剩0, got %d", whole.Quantity)
	}
	if splitQty, _ := splitRowQty(t, db, drug.ID); splitQty != 12 {
		t.Fatalf("拆零柜应为12片, got %d", splitQty)
	}
	// 发药记录：1 盒@2400 + 12 片@100 = 3600
	records, _ := repository.NewDispenseRecordRepo(db).ListByPrescription(ctx, p.ID)
	var total int64
	for _, rec := range records {
		if rec.IsSplit && (rec.Quantity != 12 || rec.UnitPrice != 100) {
			t.Fatalf("拆零发药记录不符: qty=%d price=%d", rec.Quantity, rec.UnitPrice)
		}
		if !rec.IsSplit && (rec.Quantity != 1 || rec.UnitPrice != 2400) {
			t.Fatalf("整盒发药记录不符: qty=%d price=%d", rec.Quantity, rec.UnitPrice)
		}
		total += rec.Amount
	}
	if total != 3600 {
		t.Fatalf("发药金额合计不符: %d want 3600", total)
	}
}

// TestAutoSplitPartialSplit 部分拆零覆盖：拆零柜有 6 片，36 片处方 → 用 6 片拆零 + 待拆 1 盒(split_units=6)。
func TestAutoSplitPartialSplit(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	inv := service.NewInventoryService(db)
	presc := service.NewPrescriptionService(db, inv, service.NewSpecialDrugService(db), nil, nil, service.NewClinicalService(db))
	drug := mustCreateDrug(t, db)

	// 备 3 盒：拆 1 盒留拆零、调成 6 片，剩 2 整盒供「1 盒整发 + 1 盒待拆」
	if err := inv.StockIn(ctx, []service.StockEntry{{
		DrugID: drug.ID, LocationID: 2, BatchNo: "PART-B",
		ExpiryDate: time.Now().AddDate(1, 0, 0), IsSplit: false, Quantity: 3, UnitPrice: 1800,
	}}, 1, "自动拆零测试"); err != nil {
		t.Fatalf("入库失败: %v", err)
	}
	whole := wholeRow(t, db, drug.ID)
	// 拆零 6 片（开 1 盒：6 片入拆零 + 18 片报损…不平齐）。改用按片拆：units=6, damaged=0 需 units+damaged==pack → 不平齐。
	// 因此用 Split 拆零后调整：直接拆 0 盒不可行。改为：拆 1 盒全入拆零(24)，再 Adjust -18 报损。
	if err := inv.Split(ctx, service.SplitRequest{InventoryID: whole.ID, Packs: 1}, 1, "自动拆零测试"); err != nil {
		t.Fatalf("拆零失败: %v", err)
	}
	splitRows, _, _ := repository.NewInventoryRepo(db).List(ctx, repository.InventoryListFilter{DrugID: drug.ID, LocationID: 2}, 0, 20)
	for _, r := range splitRows {
		if r.IsSplit {
			// 报损 18 片 → 剩 6 片
			if err := inv.Adjust(ctx, service.AdjustRequest{InventoryID: r.ID, Quantity: -18, Reason: "测试调整"}, 1, "自动拆零测试"); err != nil {
				t.Fatalf("调整失败: %v", err)
			}
		}
	}

	p, err := presc.Create(ctx, service.PrescriptionInput{
		PatientName: "部分拆零患者",
		Items:       []service.PrescriptionItemInput{{DrugID: drug.ID, Quantity: 36}}, // 1盒 + 12片
	})
	if err != nil {
		t.Fatalf("创建处方失败: %v", err)
	}
	if err := presc.Submit(ctx, p.ID, 1, "药师A"); err != nil {
		t.Fatalf("提交失败: %v", err)
	}
	resvs, _ := repository.NewStockReservationRepo(db).ListActiveByRef(ctx, "prescription", p.ID)
	var splitResv, needSplitUnits int64
	for _, r := range resvs {
		if r.NeedSplit {
			needSplitUnits = r.SplitUnits
		} else if r.IsSplit {
			splitResv += r.Quantity
		}
	}
	if splitResv != 6 || needSplitUnits != 6 {
		t.Fatalf("部分拆零规划不符: splitResv=%d needSplitUnits=%d", splitResv, needSplitUnits)
	}

	if _, err := presc.Review(ctx, p.ID, service.AuditInput{Action: "pass"}, 2, "药师B"); err != nil {
		t.Fatalf("审核失败: %v", err)
	}
	if err := presc.Dispense(ctx, p.ID, 3, "调配员", "pharmacist"); err != nil {
		t.Fatalf("调配失败: %v", err)
	}
	if err := presc.ConfirmDispense(ctx, p.ID, 4, "核对员", "pharmacist"); err != nil {
		t.Fatalf("发药确认失败: %v", err)
	}
	// 发药后：拆零柜 6-6(用掉)+18(待拆盒余片)=18，整盒剩 0
	if splitQty, _ := splitRowQty(t, db, drug.ID); splitQty != 18 {
		t.Fatalf("拆零柜应为18片, got %d", splitQty)
	}
	if w := wholeRow(t, db, drug.ID); w.Quantity != 0 {
		t.Fatalf("整盒应剩0, got %d", w.Quantity)
	}
}

// TestReturnSplitScenario 用户场景还原：一盒 15 片、拆零剩 2 片，处方 3 片
// → 系统开盒（整盒-1、拆零+15 净效果）→ 处方扣 3 片；
// 随后其他医生开 5 片零散处方正常扣减；退药 3 片按原批次正常回补拆零柜。
func TestReturnSplitScenario(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	inv := service.NewInventoryService(db)
	presc := service.NewPrescriptionService(db, inv, service.NewSpecialDrugService(db), nil, nil, service.NewClinicalService(db))
	drugSvc := service.NewDrugService(db)

	drug := &model.Drug{
		Code: "SC001", GenericName: "场景药", DosageForm: "片剂",
		Specification: "0.1g", Manufacturer: "场景药厂",
		BaseUnit: "盒", SplitUnit: "片", PackSize: 15, IsSplitAllowed: true,
		RetailPrice: 1500, PurchasePrice: 1200, Status: 1,
	}
	if err := drugSvc.Create(ctx, drug); err != nil {
		t.Fatalf("创建药品失败: %v", err)
	}
	if drug.SplitRetailPrice != 100 { // round(1500/15)
		t.Fatalf("拆零价应为100, got %d", drug.SplitRetailPrice)
	}

	exp := time.Now().AddDate(1, 0, 0)
	// 库存：整盒 1 盒(batch W) + 拆零 2 片(batch S)
	if err := inv.StockIn(ctx, []service.StockEntry{
		{DrugID: drug.ID, LocationID: 2, BatchNo: "W", ExpiryDate: exp, IsSplit: false, Quantity: 1, UnitPrice: 1200},
		{DrugID: drug.ID, LocationID: 2, BatchNo: "S", ExpiryDate: exp, IsSplit: true, Quantity: 2, UnitPrice: 80},
	}, 1, "场景测试"); err != nil {
		t.Fatalf("入库失败: %v", err)
	}

	// Rx1：3 片（拆零不足 → 自动规划待拆盒）
	p1, err := presc.Create(ctx, service.PrescriptionInput{PatientName: "医生A患者", Items: []service.PrescriptionItemInput{{DrugID: drug.ID, Quantity: 3}}})
	if err != nil {
		t.Fatalf("创建Rx1失败: %v", err)
	}
	if err := presc.Submit(ctx, p1.ID, 1, "药师A"); err != nil {
		t.Fatalf("Rx1提交失败: %v", err)
	}
	resvs, _ := repository.NewStockReservationRepo(db).ListActiveByRef(ctx, "prescription", p1.ID)
	var needSplit bool
	for _, r := range resvs {
		if r.NeedSplit {
			needSplit = true
			if r.SplitUnits != 1 { // 拆零缺1片
				t.Fatalf("待拆盒 split_units 应为1, got %d", r.SplitUnits)
			}
		}
	}
	if !needSplit {
		t.Fatal("拆零不足时应规划待拆盒")
	}
	for _, fn := range []func() error{
		func() error {
			_, err := presc.Review(ctx, p1.ID, service.AuditInput{Action: "pass"}, 2, "药师B")
			return err
		},
		func() error { return presc.Dispense(ctx, p1.ID, 3, "调配员", "pharmacist") },
		func() error { return presc.ConfirmDispense(ctx, p1.ID, 4, "核对员", "pharmacist") },
	} {
		if err := fn(); err != nil {
			t.Fatalf("Rx1流程失败: %v", err)
		}
	}
	// 发药后：整盒 W=0，拆零 = S 0 + W-split 14（开盒15片，患者取1片）
	if q := wholeRow(t, db, drug.ID); q.Quantity != 0 {
		t.Fatalf("整盒应剩0, got %d", q.Quantity)
	}
	if splitQty, _ := splitRowQty(t, db, drug.ID); splitQty != 14 {
		t.Fatalf("拆零应剩14片, got %d", splitQty)
	}

	// Rx2：另一医生开 5 片零散处方 → 从拆零柜正常扣减
	p2, err := presc.Create(ctx, service.PrescriptionInput{PatientName: "医生B患者", Items: []service.PrescriptionItemInput{{DrugID: drug.ID, Quantity: 5}}})
	if err != nil {
		t.Fatalf("创建Rx2失败: %v", err)
	}
	if err := presc.Submit(ctx, p2.ID, 1, "药师A"); err != nil {
		t.Fatalf("Rx2提交失败: %v", err)
	}
	if _, err := presc.Review(ctx, p2.ID, service.AuditInput{Action: "pass"}, 2, "药师B"); err != nil {
		t.Fatalf("Rx2审核失败: %v", err)
	}
	if err := presc.Dispense(ctx, p2.ID, 3, "调配员", "pharmacist"); err != nil {
		t.Fatalf("Rx2调配失败: %v", err)
	}
	if err := presc.ConfirmDispense(ctx, p2.ID, 4, "核对员", "pharmacist"); err != nil {
		t.Fatalf("Rx2发药失败: %v", err)
	}
	if splitQty, _ := splitRowQty(t, db, drug.ID); splitQty != 9 {
		t.Fatalf("Rx2扣减5片后拆零应剩9, got %d", splitQty)
	}

	// Rx1 退药 3 片 → 按原批次回补拆零柜（不重新封盒）
	items1, _ := repository.NewPrescriptionItemRepo(db).ListByPrescription(ctx, p1.ID)
	if err := presc.Return(ctx, p1.ID, []service.ReturnItemInput{{ItemID: items1[0].ID, ReturnQuantity: 3}}, 5, "药师C"); err != nil {
		t.Fatalf("Rx1退药失败: %v", err)
	}
	// 最终拆零 = 9 + 3 = 12（Rx2 的 5 片不受影响）
	if splitQty, _ := splitRowQty(t, db, drug.ID); splitQty != 12 {
		t.Fatalf("退药后拆零应为12, got %d", splitQty)
	}
	if w := wholeRow(t, db, drug.ID); w.Quantity != 0 {
		t.Fatalf("整盒仍应0, got %d", w.Quantity)
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
