//go:build integration

// 统一记账口径（000040）回归测试。
//
// 决策：charges（结算单）+ charge_items（费用行）是**唯一记账凭证**；
// charge_records 降级为「应收计费项目源」，只被结算单消费，不被任何报表直接统计。
package service_test

import (
	"context"
	"testing"
	"time"

	"gorm.io/gorm"

	"yaofang/internal/model"
	"yaofang/internal/repository"
	"yaofang/internal/service"
	"yaofang/internal/service/pricing"
)

// newVisitAndPatient 建一个患者 + 就诊（visiting 状态，可生成结算单）。
func newVisitAndPatient(t *testing.T, db *gorm.DB) (*model.Patient, *model.Visit) {
	t.Helper()
	ctx := context.Background()
	pat := mustCreatePatient(t, db)
	visitSvc := service.NewVisitService(db)
	v, err := visitSvc.Register(ctx, service.VisitInput{PatientID: pat.ID}, "挂号员")
	if err != nil {
		t.Fatalf("挂号失败: %v", err)
	}
	if err := visitSvc.Start(ctx, v.ID); err != nil {
		t.Fatalf("接诊失败: %v", err)
	}
	return pat, v
}

// TestBillCollectsOnlyTargetVisit 结算单只归集本就诊的费用项。
//
// 回归：charge_records 没有 visit_id 列，CalculateBill ② 只能用
// (patient_id, created_at ∈ [挂号, 结束]) 的时间窗口猜归属——同一患者在窗口内
// 其它就诊录入的计费项会被并入本次结算，造成账单串号。
func TestBillCollectsOnlyTargetVisit(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	pat := mustCreatePatient(t, db)
	visitSvc := service.NewVisitService(db)
	clinical := service.NewClinicalService(db)

	// 就诊1：开始 → 录费用 → 结束
	v1, err := visitSvc.Register(ctx, service.VisitInput{PatientID: pat.ID}, "挂号员")
	if err != nil {
		t.Fatalf("挂号1失败: %v", err)
	}
	if err := visitSvc.Start(ctx, v1.ID); err != nil {
		t.Fatalf("接诊1失败: %v", err)
	}
	if _, err := clinical.CreateCharge(ctx, service.ChargeInput{
		VisitID: v1.ID, PatientID: pat.ID, PatientName: pat.Name,
		ItemType: "clinical_service", ItemName: "就诊1项目", Quantity: 1, UnitPrice: 1000,
	}, 1, "护士"); err != nil {
		t.Fatalf("就诊1计费失败: %v", err)
	}
	if err := visitSvc.Finish(ctx, v1.ID); err != nil {
		t.Fatalf("结束1失败: %v", err)
	}

	// 就诊2：同一患者，紧随其后（落在就诊1的时间窗口之后，故不会被误并）
	v2, err := visitSvc.Register(ctx, service.VisitInput{PatientID: pat.ID}, "挂号员")
	if err != nil {
		t.Fatalf("挂号2失败: %v", err)
	}
	if err := visitSvc.Start(ctx, v2.ID); err != nil {
		t.Fatalf("接诊2失败: %v", err)
	}
	if _, err := clinical.CreateCharge(ctx, service.ChargeInput{
		VisitID: v2.ID, PatientID: pat.ID, PatientName: pat.Name,
		ItemType: "clinical_service", ItemName: "就诊2项目", Quantity: 1, UnitPrice: 2000,
	}, 1, "护士"); err != nil {
		t.Fatalf("就诊2计费失败: %v", err)
	}

	// 就诊2 的结算单只应含自己的 2000
	chargeSvc := service.NewChargeService(db, pricing.NewSimplePricingService(db))
	c2, err := chargeSvc.Create(ctx, v2.ID, 0, "收费员")
	if err != nil {
		t.Fatalf("生成就诊2结算单失败: %v", err)
	}
	full, err := chargeSvc.Get(ctx, c2.ID)
	if err != nil {
		t.Fatalf("读取结算单失败: %v", err)
	}
	var sum int64
	for _, it := range full.Items {
		sum += it.Amount
	}
	if sum != 2000 {
		t.Fatalf("就诊2结算金额应为 2000（只含本就诊），got %d（明细 %d 条）", sum, len(full.Items))
	}
}

// TestBillIncludesManualDrugCharge 手工录入的药品费必须能进结算单。
// 回归：② 此前用 item_type IN (clinical_service, consumable)，把 drug 整体排除，
// 于是无处方的手工药品费（如急救给药）永远不计入账单，收入静默漏记。
func TestBillIncludesManualDrugCharge(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	pat, visit := newVisitAndPatient(t, db)
	clinical := service.NewClinicalService(db)
	drug := mustCreateDrug(t, db)

	if _, err := clinical.CreateCharge(ctx, service.ChargeInput{
		VisitID: visit.ID, PatientID: pat.ID, PatientName: pat.Name,
		ItemType: "drug", ItemID: &drug.ID, ItemName: "急救给药",
		Quantity: 2, UnitPrice: 1500,
	}, 1, "护士"); err != nil {
		t.Fatalf("手工药品计费失败: %v", err)
	}
	chargeSvc := service.NewChargeService(db, pricing.NewSimplePricingService(db))
	c, err := chargeSvc.Create(ctx, visit.ID, 0, "收费员")
	if err != nil {
		t.Fatalf("生成结算单失败: %v", err)
	}
	full, err := chargeSvc.Get(ctx, c.ID)
	if err != nil {
		t.Fatalf("读取结算单失败: %v", err)
	}
	var drugLines int
	var sum int64
	for _, it := range full.Items {
		sum += it.Amount
		if it.ItemType == "drug" {
			drugLines++
		}
	}
	if drugLines != 1 {
		t.Fatalf("手工药品费应进入结算单（1 条 drug 行），got %d（合计 %d）", drugLines, sum)
	}
	if sum != 3000 {
		t.Fatalf("结算合计应为 3000，got %d", sum)
	}
}

// TestBillDoesNotDoubleCountPrescription 处方药费不得被重复计费。
// ① 按处方快照直读，② 排除 ref_type='prescription'——两者叠加即重复。
func TestBillDoesNotDoubleCountPrescription(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	pat, visit := newVisitAndPatient(t, db)
	inv := service.NewInventoryService(db)
	presc := service.NewPrescriptionService(db, inv, service.NewSpecialDrugService(db), nil, nil, service.NewClinicalService(db))
	clinical := service.NewClinicalService(db)
	drug := mustCreateDrug(t, db)

	exp := time.Now().AddDate(1, 0, 0)
	if err := inv.StockIn(ctx, []service.StockEntry{{
		DrugID: drug.ID, LocationID: 2, BatchNo: "BILL-DUP", ExpiryDate: exp,
		IsSplit: false, Quantity: 10, UnitPrice: drug.PurchasePrice,
	}}, 1, "集成测试"); err != nil {
		t.Fatalf("入库失败: %v", err)
	}
	// 处方关联到本次就诊
	p, err := presc.Create(ctx, service.PrescriptionInput{
		VisitID: visit.ID, PatientID: pat.ID, PatientName: pat.Name,
		Items: []service.PrescriptionItemInput{{DrugID: drug.ID, Quantity: 2, Days: 1, SingleDose: 1, TotalDailyDose: 2}},
	})
	if err != nil {
		t.Fatalf("创建处方失败: %v", err)
	}
	for _, fn := range []func() error{
		func() error { return presc.Submit(ctx, p.ID, 1, "药师A") },
		func() error {
			_, err := presc.Review(ctx, p.ID, service.AuditInput{Action: "pass"}, 2, "药师B")
			return err
		},
		func() error { return presc.Dispense(ctx, p.ID, 3, "调配员", "pharmacist") },
		func() error { return presc.ConfirmDispense(ctx, p.ID, 4, "核对员", "pharmacist") },
	} {
		if err := fn(); err != nil {
			t.Fatalf("处方流转失败: %v", err)
		}
	}
	// 一键计费：为同一处方生成 charge_records（ref_type='prescription'）
	if _, err := clinical.ChargePrescription(ctx, p.ID, 5, "收费员"); err != nil {
		t.Fatalf("一键计费失败: %v", err)
	}
	var crCount int64
	db.Model(&model.ChargeRecord{}).Where("ref_type = 'prescription' AND ref_id = ?", p.ID).Count(&crCount)
	if crCount == 0 {
		t.Fatal("一键计费应产生 charge_records")
	}

	// 结算单中该药费只应出现一次
	chargeSvc := service.NewChargeService(db, pricing.NewSimplePricingService(db))
	c, err := chargeSvc.Create(ctx, visit.ID, 0, "收费员")
	if err != nil {
		t.Fatalf("生成结算单失败: %v", err)
	}
	full, err := chargeSvc.Get(ctx, c.ID)
	if err != nil {
		t.Fatalf("读取结算单失败: %v", err)
	}
	var drugAmount int64
	var drugLines int
	for _, it := range full.Items {
		if it.ItemType == "drug" {
			drugAmount += it.Amount
			drugLines++
		}
	}
	// 处方快照价：拆零价 round(2400/24)=100，2 片 = 200 分
	if drugAmount != 200 {
		t.Fatalf("处方药费应计 200 分（2 片×100），got %d（%d 条 drug 行，重复计费）", drugAmount, drugLines)
	}
	if drugLines != 1 {
		t.Fatalf("处方药费应恰为 1 条 drug 行，got %d（①快照与②charge_records 重复计入）", drugLines)
	}
}

// TestPatientChargesReadsVoucher 按患者计费报表必须读唯一凭证（charges/charge_items），
// 与收入构成报表同源：已结算才计入，未结算不计。
func TestPatientChargesReadsVoucher(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	pat, visit := newVisitAndPatient(t, db)
	clinical := service.NewClinicalService(db)
	reportSvc := service.NewReportService(db)

	// 先录一笔应收（未结算）
	if _, err := clinical.CreateCharge(ctx, service.ChargeInput{
		VisitID: visit.ID, PatientID: pat.ID, PatientName: pat.Name,
		ItemType: "clinical_service", ItemName: "未结算项目", Quantity: 1, UnitPrice: 1000,
	}, 1, "护士"); err != nil {
		t.Fatalf("计费失败: %v", err)
	}
	rows, err := reportSvc.PatientCharges(ctx, pat.ID, nil, nil)
	if err != nil {
		t.Fatalf("查询按患者计费失败: %v", err)
	}
	if totalOf(rows) != 0 {
		t.Fatalf("未结算的应收不应计入「按患者计费」报表，got %d", totalOf(rows))
	}

	// 结算并收费后应计入
	chargeSvc := service.NewChargeService(db, pricing.NewSimplePricingService(db))
	c, err := chargeSvc.Create(ctx, visit.ID, 0, "收费员")
	if err != nil {
		t.Fatalf("生成结算单失败: %v", err)
	}
	if err := chargeSvc.Pay(ctx, c.ID, c.PayableAmount, "收费员"); err != nil {
		t.Fatalf("收费失败: %v", err)
	}
	rows, err = reportSvc.PatientCharges(ctx, pat.ID, nil, nil)
	if err != nil {
		t.Fatalf("查询按患者计费失败: %v", err)
	}
	if got := totalOf(rows); got != c.PayableAmount {
		t.Fatalf("已结算金额应计入报表（%d），got %d", c.PayableAmount, got)
	}

	// 退费后应被排除（与收入构成报表同口径）
	if err := chargeSvc.Refund(ctx, c.ID, "收费员"); err != nil {
		t.Fatalf("退费失败: %v", err)
	}
	rows, err = reportSvc.PatientCharges(ctx, pat.ID, nil, nil)
	if err != nil {
		t.Fatalf("查询按患者计费失败: %v", err)
	}
	if got := totalOf(rows); got != 0 {
		t.Fatalf("已退费结算单不应计入报表，got %d", got)
	}
}

// TestChargeItemLinksSourceRecord 结算单费用行须回溯到其计费项目源（source_record_id），
// 消除 item_id 多态引用的歧义。
func TestChargeItemLinksSourceRecord(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	pat, visit := newVisitAndPatient(t, db)
	clinical := service.NewClinicalService(db)

	cr, err := clinical.CreateCharge(ctx, service.ChargeInput{
		VisitID: visit.ID, PatientID: pat.ID, PatientName: pat.Name,
		ItemType: "clinical_service", ItemName: "回源项目", Quantity: 1, UnitPrice: 1000,
	}, 1, "护士")
	if err != nil {
		t.Fatalf("计费失败: %v", err)
	}
	chargeSvc := service.NewChargeService(db, pricing.NewSimplePricingService(db))
	c, err := chargeSvc.Create(ctx, visit.ID, 0, "收费员")
	if err != nil {
		t.Fatalf("生成结算单失败: %v", err)
	}
	full, err := chargeSvc.Get(ctx, c.ID)
	if err != nil {
		t.Fatalf("读取结算单失败: %v", err)
	}
	var found bool
	for _, it := range full.Items {
		if it.SourceRecordID == nil {
			continue
		}
		found = true
		src, err := repository.NewChargeRecordRepo(db).GetByID(ctx, *it.SourceRecordID)
		if err != nil {
			t.Fatalf("source_record_id 指向的计费项目源不存在: %v", err)
		}
		if src.ID != cr.ID {
			t.Fatalf("source_record_id 应指向 %d，实际 %d", cr.ID, src.ID)
		}
	}
	if !found {
		t.Fatal("结算单费用行未记录 source_record_id，无法回溯到计费项目源")
	}
}

func totalOf(rows []repository.PatientChargeRow) int64 {
	var n int64
	for _, r := range rows {
		n += r.Amount
	}
	return n
}
