//go:build integration

// 计费与领用回归测试：红冲幂等、一键计费幂等、实收校验、结算单幂等、领用不吞预占。
package service_test

import (
	"context"
	"testing"
	"time"

	"gorm.io/gorm"

	"yaofang/internal/model"
	"yaofang/internal/pkg/errs"
	"yaofang/internal/repository"
	"yaofang/internal/service"
	"yaofang/internal/service/pricing"
)

func truncateBillingTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	tables := []string{
		"charge_records", "charge_items", "charges",
		"medical_record_diagnoses", "medical_records", "visits", "patients",
	}
	for _, tb := range tables {
		if err := db.Exec("TRUNCATE TABLE " + tb + " RESTART IDENTITY CASCADE").Error; err != nil {
			t.Fatalf("清空表 %s 失败: %v", tb, err)
		}
	}
}

// TestVoidChargeIdempotent 并发/重复红冲只允许产生一张负冲正单。
func TestVoidChargeIdempotent(t *testing.T) {
	db := setupTestDB(t)
	truncateBillingTables(t, db)
	ctx := context.Background()
	svc := service.NewClinicalService(db)

	cr, err := svc.CreateCharge(ctx, service.ChargeInput{
		PatientName: "红冲患者", ItemType: "clinical_service", ItemName: "手法复位",
		Quantity: 1, UnitPrice: 5000,
	}, 1, "收费员")
	if err != nil {
		t.Fatalf("创建计费失败: %v", err)
	}
	if err := svc.VoidCharge(ctx, cr.ID, 2, "红冲员"); err != nil {
		t.Fatalf("首次红冲失败: %v", err)
	}
	if err := svc.VoidCharge(ctx, cr.ID, 2, "红冲员"); !errs.Is(err, errs.ErrChargeVoided) {
		t.Fatalf("重复红冲应返回 ErrChargeVoided, got %v", err)
	}
	var n int64
	db.Model(&model.ChargeRecord{}).Where("ref_type = 'charge_void' AND ref_id = ?", cr.ID).Count(&n)
	if n != 1 {
		t.Fatalf("负冲正单应恰为 1 张, got %d", n)
	}
}

// TestChargePrescriptionIdempotent 同处方重复一键计费不重复产生记录。
func TestChargePrescriptionIdempotent(t *testing.T) {
	db := setupTestDB(t)
	truncateBillingTables(t, db)
	ctx := context.Background()
	inv := service.NewInventoryService(db)
	presc := service.NewPrescriptionService(db, inv, service.NewSpecialDrugService(db), nil, nil, service.NewClinicalService(db))
	clinical := service.NewClinicalService(db)
	drug := mustCreateDrug(t, db)

	if err := inv.StockIn(ctx, []service.StockEntry{{
		DrugID: drug.ID, LocationID: 2, BatchNo: "BILL-A",
		ExpiryDate: time.Now().AddDate(1, 0, 0), IsSplit: false, Quantity: 5, UnitPrice: drug.PurchasePrice,
	}}, 1, "集成测试"); err != nil {
		t.Fatalf("入库失败: %v", err)
	}
	p, err := presc.Create(ctx, service.PrescriptionInput{
		PatientName: "计费患者",
		Items:       []service.PrescriptionItemInput{{DrugID: drug.ID, Quantity: 24}},
	})
	if err != nil {
		t.Fatalf("创建处方失败: %v", err)
	}
	for _, step := range []func() error{
		func() error { return presc.Submit(ctx, p.ID, 1, "药师A") },
		func() error {
			_, err := presc.Review(ctx, p.ID, service.AuditInput{Action: "pass"}, 2, "药师B")
			return err
		},
		func() error { return presc.Dispense(ctx, p.ID, 3, "调配员", "pharmacist") },
		func() error { return presc.ConfirmDispense(ctx, p.ID, 4, "核对员", "pharmacist") },
	} {
		if err := step(); err != nil {
			t.Fatalf("处方流转失败: %v", err)
		}
	}

	first, err := clinical.ChargePrescription(ctx, p.ID, 5, "收费员")
	if err != nil {
		t.Fatalf("一键计费失败: %v", err)
	}
	if len(first) == 0 {
		t.Fatal("一键计费应产生计费记录")
	}
	second, err := clinical.ChargePrescription(ctx, p.ID, 5, "收费员")
	if err != nil {
		t.Fatalf("重复一键计费失败: %v", err)
	}
	if len(second) != 0 {
		t.Fatalf("重复一键计费应返回空, got %d 条", len(second))
	}
	var n int64
	db.Model(&model.ChargeRecord{}).Where("ref_type = 'prescription' AND ref_id = ? AND amount > 0", p.ID).Count(&n)
	if n != int64(len(first)) {
		t.Fatalf("正计费记录应恰为 %d 条, got %d", len(first), n)
	}
}

// TestChargeVisitIdempotentAndPay 同一就诊仅一张结算单；实收必须等于应收。
func TestChargeVisitIdempotentAndPay(t *testing.T) {
	db := setupTestDB(t)
	truncateBillingTables(t, db)
	ctx := context.Background()

	pat := mustCreatePatient(t, db)
	visitSvc := service.NewVisitService(db)
	visit, err := visitSvc.Register(ctx, service.VisitInput{PatientID: pat.ID}, "挂号员")
	if err != nil {
		t.Fatalf("挂号失败: %v", err)
	}
	// 结算要求就诊处于 visiting/finished 状态
	if err := visitSvc.Start(ctx, visit.ID); err != nil {
		t.Fatalf("接诊失败: %v", err)
	}

	clinical := service.NewClinicalService(db)
	if _, err := clinical.CreateCharge(ctx, service.ChargeInput{
		PatientID: pat.ID, PatientName: pat.Name, ItemType: "clinical_service", ItemName: "注射",
		Quantity: 1, UnitPrice: 3000,
	}, 1, "护士"); err != nil {
		t.Fatalf("创建计费失败: %v", err)
	}

	chargeSvc := service.NewChargeService(db, pricing.NewSimplePricingService(db))
	c1, err := chargeSvc.Create(ctx, visit.ID, 0, "收费员")
	if err != nil {
		t.Fatalf("生成结算单失败: %v", err)
	}
	_, err = chargeSvc.Create(ctx, visit.ID, 0, "收费员")
	if !errs.Is(err, errs.ErrChargeExists) {
		t.Fatalf("重复生成结算单应返回 ErrChargeExists, got %v", err)
	}

	// 实收 ≠ 应收 → 拒绝
	if err := chargeSvc.Pay(ctx, c1.ID, c1.PayableAmount-1, "收费员"); !errs.Is(err, errs.ErrPaidAmountMismatch) {
		t.Fatalf("实收与应收不一致应返回 ErrPaidAmountMismatch, got %v", err)
	}
	// 实收 = 应收 → 成功
	if err := chargeSvc.Pay(ctx, c1.ID, c1.PayableAmount, "收费员"); err != nil {
		t.Fatalf("收费失败: %v", err)
	}
	var paid model.Charge
	if err := db.First(&paid, c1.ID).Error; err != nil {
		t.Fatal(err)
	}
	if paid.Status != "paid" || paid.PaidAmount != c1.PayableAmount {
		t.Fatalf("收费后状态/金额不符: status=%s paid=%d payable=%d", paid.Status, paid.PaidAmount, c1.PayableAmount)
	}
}

// TestRequisitionNotConsumeReserved 领用出库不得扣掉处方已预占库存。
func TestRequisitionNotConsumeReserved(t *testing.T) {
	db := setupTestDB(t)
	truncateBillingTables(t, db)
	ctx := context.Background()
	inv := service.NewInventoryService(db)
	presc := service.NewPrescriptionService(db, inv, service.NewSpecialDrugService(db), nil, nil, service.NewClinicalService(db))
	drug := mustCreateDrug(t, db)

	if err := inv.StockIn(ctx, []service.StockEntry{{
		DrugID: drug.ID, LocationID: 2, BatchNo: "REQ-A",
		ExpiryDate: time.Now().AddDate(1, 0, 0), IsSplit: false, Quantity: 10, UnitPrice: drug.PurchasePrice,
	}}, 1, "集成测试"); err != nil {
		t.Fatalf("入库失败: %v", err)
	}
	// 预占 2 盒（48 LDU）→ 整盒行 reserved=2（盒口径），可用 8 盒
	p, err := presc.Create(ctx, service.PrescriptionInput{
		PatientName: "预占患者",
		Items:       []service.PrescriptionItemInput{{DrugID: drug.ID, Quantity: 48}},
	})
	if err != nil {
		t.Fatalf("创建处方失败: %v", err)
	}
	if err := presc.Submit(ctx, p.ID, 1, "药师A"); err != nil {
		t.Fatalf("提交失败: %v", err)
	}

	// 领用 12 盒（Requisition quantity 为行口径：整盒行=盒数）> 可用 8 盒（扣除预占）：应被拦截且库存不变
	err = inv.Requisition(ctx, drug.ID, 2, 12, "科室领用", 1, "护士")
	if err == nil {
		t.Fatal("领用超过可用量（扣除预占）应被拦截")
	}
	var row model.Inventory
	if err := db.Where("drug_id = ? AND location_id = 2 AND batch_no = 'REQ-A'", drug.ID).First(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.Quantity != 10 || row.ReservedQuantity != 2 {
		t.Fatalf("领用被拦截后库存不得变动: quantity=%d reserved=%d", row.Quantity, row.ReservedQuantity)
	}
	// 领用 8 盒（可用量内）应成功，剩余 2 盒
	if err := inv.Requisition(ctx, drug.ID, 2, 8, "科室领用", 1, "护士"); err != nil {
		t.Fatalf("领用可用量应成功: %v", err)
	}
	if err := db.Where("drug_id = ? AND location_id = 2 AND batch_no = 'REQ-A'", drug.ID).First(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.Quantity != 2 || row.ReservedQuantity != 2 {
		t.Fatalf("领用后库存应为 2、预占保持 2: quantity=%d reserved=%d", row.Quantity, row.ReservedQuantity)
	}
}

// TestCalculateBillPartialReturn 部分退药后合并结算按净额计费。
func TestCalculateBillPartialReturn(t *testing.T) {
	db := setupTestDB(t)
	truncateBillingTables(t, db)
	ctx := context.Background()
	inv := service.NewInventoryService(db)
	presc := service.NewPrescriptionService(db, inv, service.NewSpecialDrugService(db), nil, nil, service.NewClinicalService(db))
	drug := mustCreateDrug(t, db)
	pat := mustCreatePatient(t, db)

	if err := inv.StockIn(ctx, []service.StockEntry{{
		DrugID: drug.ID, LocationID: 2, BatchNo: "RET-A",
		ExpiryDate: time.Now().AddDate(1, 0, 0), IsSplit: false, Quantity: 10, UnitPrice: drug.PurchasePrice,
	}}, 1, "集成测试"); err != nil {
		t.Fatalf("入库失败: %v", err)
	}
	// 处方 3 盒（72 LDU），单价 2400 分/盒
	p, err := presc.Create(ctx, service.PrescriptionInput{
		PatientID: pat.ID, PatientName: pat.Name,
		Items: []service.PrescriptionItemInput{{DrugID: drug.ID, Quantity: 72}},
	})
	if err != nil {
		t.Fatalf("创建处方失败: %v", err)
	}
	for _, step := range []func() error{
		func() error { return presc.Submit(ctx, p.ID, 1, "药师A") },
		func() error {
			_, err := presc.Review(ctx, p.ID, service.AuditInput{Action: "pass"}, 2, "药师B")
			return err
		},
		func() error { return presc.Dispense(ctx, p.ID, 3, "调配员", "pharmacist") },
		func() error { return presc.ConfirmDispense(ctx, p.ID, 4, "核对员", "pharmacist") },
	} {
		if err := step(); err != nil {
			t.Fatalf("处方流转失败: %v", err)
		}
	}
	items, err := repository.NewPrescriptionItemRepo(db).ListByPrescription(ctx, p.ID)
	if err != nil || len(items) == 0 {
		t.Fatalf("查询处方明细失败: %v", err)
	}
	// 部分退药 1 盒（24 LDU）
	if err := presc.Return(ctx, p.ID, []service.ReturnItemInput{{ItemID: items[0].ID, ReturnQuantity: 24}}, 5, "药师C"); err != nil {
		t.Fatalf("退药失败: %v", err)
	}
	// 挂号并关联处方 → 就诊
	visitSvc := service.NewVisitService(db)
	visit, err := visitSvc.Register(ctx, service.VisitInput{PatientID: pat.ID}, "挂号员")
	if err != nil {
		t.Fatalf("挂号失败: %v", err)
	}
	if err := db.Model(&model.Prescription{}).Where("id = ?", p.ID).Update("visit_id", visit.ID).Error; err != nil {
		t.Fatal(err)
	}

	pricer := pricing.NewSimplePricingService(db)
	lines, err := pricer.CalculateBill(ctx, visit.ID)
	if err != nil {
		t.Fatalf("CalculateBill 失败: %v", err)
	}
	var netAmount int64
	for _, l := range lines {
		if l.ItemType == "drug" {
			netAmount += l.Amount
		}
	}
	// 净额 = 3 盒 - 退 1 盒 = 2 盒 × 2400 = 4800 分
	if netAmount != 4800 {
		t.Fatalf("部分退药后药费净额应为 4800 分, got %d", netAmount)
	}
}
