//go:build integration

// 二期契约测试：以「模拟诊疗模块」身份通过 port 接口调用，验证出入参契约稳定（docs/05）。
package service_test

import (
	"context"
	"testing"
	"time"

	"gorm.io/gorm"

	"yaofang/internal/model"
	"yaofang/internal/repository"
	"yaofang/internal/service"
	"yaofang/internal/service/patient"
	"yaofang/internal/service/port"
	"yaofang/internal/service/pricing"
)

// 编译期断言：InventoryService 必须完整实现 port.IStockService（契约守门）。
var (
	_ port.IStockService   = (*service.InventoryService)(nil)
	_ port.IPatientService = (*patient.SimplePatientService)(nil)
	_ port.IPricingService = (*pricing.SimplePricingService)(nil)
)

func mustStockWhole(t *testing.T, db *gorm.DB, drugID int64, batch string, qty int64) {
	t.Helper()
	inv := service.NewInventoryService(db)
	if err := inv.StockIn(context.Background(), []service.StockEntry{{
		DrugID: drugID, LocationID: 2, BatchNo: batch,
		ExpiryDate: time.Now().AddDate(1, 0, 0), IsSplit: false, Quantity: qty, UnitPrice: 1500,
	}}, 1, "契约测试"); err != nil {
		t.Fatalf("入库失败: %v", err)
	}
}

// TestStockServiceContract 库存服务接口契约：预占→查询→实扣→释放 全链路。
func TestStockServiceContract(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	inv := service.NewInventoryService(db)
	drug := mustCreateDrug(t, db)
	mustStockWhole(t, db, drug.ID, "CT-BATCH", 10)

	const refType, refID = "contract_test", 9001

	// 1. 预占 6 盒（整盒口径）
	results, err := inv.ReserveStock(ctx, refType, refID, []port.ReserveItem{
		{DrugID: drug.ID, LocationID: 2, IsSplit: false, Quantity: 6},
	})
	if err != nil {
		t.Fatalf("ReserveStock 失败: %v", err)
	}
	if len(results) != 1 || results[0].Reserved != 6 || results[0].Shortage != 0 {
		t.Fatalf("预占结果不符合契约: %+v", results)
	}

	// 2. 可用量查询：药房可用应为 4 盒（10-6）
	avail, err := inv.GetDrugAvailability(ctx, drug.ID)
	if err != nil {
		t.Fatalf("GetDrugAvailability 失败: %v", err)
	}
	var gotAvail int64
	for _, a := range avail {
		if a.LocationID == 2 {
			gotAvail = a.Available
		}
	}
	if gotAvail != int64(4)*int64(drug.PackSize) {
		t.Fatalf("可用量契约不符: got %d, want %d", gotAvail, 4*drug.PackSize)
	}

	// 3. 实扣 6 盒（按预占核销）
	if err := inv.DispenseAndReduceStock(ctx, refType, refID, []port.DispenseItem{
		{DrugID: drug.ID, LocationID: 2, IsSplit: false, Quantity: 6},
	}); err != nil {
		t.Fatalf("DispenseAndReduceStock 失败: %v", err)
	}

	// 4. 释放预占（对第二个预占单）
	if _, err := inv.ReserveStock(ctx, refType, 9002, []port.ReserveItem{
		{DrugID: drug.ID, LocationID: 2, IsSplit: false, Quantity: 3},
	}); err != nil {
		t.Fatalf("二次预占失败: %v", err)
	}
	if err := inv.CancelReservation(ctx, refType, 9002, []port.ReserveItem{
		{DrugID: drug.ID, LocationID: 2, IsSplit: false, Quantity: 3},
	}); err != nil {
		t.Fatalf("CancelReservation 失败: %v", err)
	}
	// 释放后该库房可用应为 1 盒（10-6 已实扣 6 → 剩 4，再预占 3 又释放 → 剩 4）
	avail, _ = inv.GetDrugAvailability(ctx, drug.ID)
	for _, a := range avail {
		if a.LocationID == 2 {
			gotAvail = a.Available
		}
	}
	if gotAvail != int64(4)*int64(drug.PackSize) {
		t.Fatalf("释放后可用量契约不符: got %d, want %d", gotAvail, 4*drug.PackSize)
	}
}

// TestPricingServiceContract 计价服务契约：药费明细金额与快照一致。
func TestPricingServiceContract(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	inv := service.NewInventoryService(db)
	prescSvc := service.NewPrescriptionService(db, inv, service.NewSpecialDrugService(db), nil, nil, service.NewClinicalService(db))
	drug := mustCreateDrug(t, db)

	p, err := prescSvc.Create(ctx, service.PrescriptionInput{
		PatientName: "契约计价患者",
		Items:       []service.PrescriptionItemInput{{DrugID: drug.ID, Quantity: 12, IsSplit: true}},
	})
	if err != nil {
		t.Fatalf("创建处方失败: %v", err)
	}
	items, _ := repository.NewPrescriptionItemRepo(db).ListByPrescription(ctx, p.ID)
	if len(items) == 0 {
		t.Fatal("处方应含明细")
	}
	ids := []int64{items[0].ID}

	lines, err := pricing.NewSimplePricingService(db).CalculatePrescriptionAmount(ctx, ids)
	if err != nil {
		t.Fatalf("CalculatePrescriptionAmount 失败: %v", err)
	}
	if len(lines) != 1 {
		t.Fatalf("计价行数契约不符: %d", len(lines))
	}
	// 拆零价 100 分/粒 × 12 = 1200 分
	if lines[0].ItemType != "drug" || lines[0].Amount != 1200 {
		t.Fatalf("计价行契约不符: %+v", lines[0])
	}
}

// TestPatientServiceContract 患者服务契约：一期简易实现不报错、返回空集。
func TestPatientServiceContract(t *testing.T) {
	ctx := context.Background()
	ps := patient.NewSimplePatientService()
	if id, err := ps.Register(ctx, &port.Patient{Name: "契约患者", CardNo: "110101200001010000"}); err != nil || id != 0 {
		t.Fatalf("Register 契约不符: id=%d err=%v", id, err)
	}
	if p, err := ps.GetPatient(ctx, 1); err != nil || p != nil {
		t.Fatalf("GetPatient 契约不符: p=%v err=%v", p, err)
	}
	if a, err := ps.GetAllergies(ctx, 1); err != nil || len(a) != 0 {
		t.Fatalf("GetAllergies 契约不符: %v %v", a, err)
	}
	if m, err := ps.GetMedicationHistory(ctx, 1); err != nil || len(m) != 0 {
		t.Fatalf("GetMedicationHistory 契约不符: %v %v", m, err)
	}
}

// TestCalculateBillContract 合并结算计价契约（docs/20 S4）：
// 管理员配置默认诊查费后，CalculateBill 对无处方就诊自动带出诊查费行。
func TestCalculateBillContract(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	svc := service.NewVisitService(db)
	pricer := pricing.NewSimplePricingService(db)
	settingSvc := service.NewSettingService(db)
	// 自包含：先清零默认诊费，结束时恢复（system_settings 不被 setupTestDB 清空）
	for _, k := range []string{model.SettingDefaultRegistrationFee, model.SettingDefaultConsultationFee} {
		if err := settingSvc.Update(ctx, k, "0", "契约测试"); err != nil {
			t.Fatalf("重置设置 %s 失败: %v", k, err)
		}
	}
	t.Cleanup(func() {
		_ = settingSvc.Update(context.Background(), model.SettingDefaultRegistrationFee, "0", "契约测试")
		_ = settingSvc.Update(context.Background(), model.SettingDefaultConsultationFee, "0", "契约测试")
	})

	// 患者 + 就诊（仅挂号，无处方/计费记录）
	pat := mustCreatePatient(t, db)
	visit, err := svc.Register(ctx, service.VisitInput{PatientID: pat.ID, Department: "内科"}, "契约测试")
	if err != nil {
		t.Fatalf("挂号失败: %v", err)
	}
	// 初始无默认诊费 → 空明细
	lines, err := pricer.CalculateBill(ctx, visit.ID)
	if err != nil {
		t.Fatalf("CalculateBill 失败: %v", err)
	}
	if len(lines) != 0 {
		t.Fatalf("默认诊费为 0 时应无明细, got %+v", lines)
	}
	// 配置默认挂号费 500 / 诊查费 2000
	if err := settingSvc.Update(ctx, model.SettingDefaultRegistrationFee, "500", "契约测试"); err != nil {
		t.Fatalf("设置挂号费失败: %v", err)
	}
	if err := settingSvc.Update(ctx, model.SettingDefaultConsultationFee, "2000", "契约测试"); err != nil {
		t.Fatalf("设置诊查费失败: %v", err)
	}
	lines, err = pricer.CalculateBill(ctx, visit.ID)
	if err != nil {
		t.Fatalf("CalculateBill 失败: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("应带出挂号费+诊查费 2 行, got %d", len(lines))
	}
	got := map[string]int64{}
	for _, l := range lines {
		got[l.ItemType] = l.Amount
	}
	if got["registration"] != 500 || got["consultation"] != 2000 {
		t.Fatalf("默认诊费契约不符: %+v", got)
	}
}
