//go:build integration

// 审计修复回归测试：覆盖第五轮审查确认的 Critical/High 缺陷修复。
package service_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"gorm.io/gorm"

	"yaofang/internal/model"
	"yaofang/internal/pkg/auth"
	"yaofang/internal/pkg/errs"
	"yaofang/internal/repository"
	"yaofang/internal/service"
)

// TestTransferRejectsNonPositiveQuantity 调拨数量必须为正。
// 回归：负数时 Available()<qty 恒真、Deduct 生成 "quantity + N"，
// 源库房库存凭空增加且 transfer_out 流水被记为正数（伪入库）。
func TestTransferRejectsNonPositiveQuantity(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	inv := service.NewInventoryService(db)
	drug := mustCreateDrug(t, db)

	exp := time.Now().AddDate(1, 0, 0)
	if err := inv.StockIn(ctx, []service.StockEntry{{
		DrugID: drug.ID, LocationID: 2, BatchNo: "TR-NEG", ExpiryDate: exp,
		IsSplit: false, Quantity: 10, UnitPrice: drug.PurchasePrice,
	}}, 1, "调拨测试"); err != nil {
		t.Fatalf("入库失败: %v", err)
	}
	row, err := repository.NewInventoryRepo(db).FindByKey(ctx, drug.ID, 2, "TR-NEG", exp, false)
	if err != nil {
		t.Fatalf("查询库存行失败: %v", err)
	}

	for _, qty := range []int64{-5, 0} {
		err := inv.Transfer(ctx, 2, 3, []service.TransferItem{
			{InventoryID: row.ID, Quantity: qty},
		}, 1, "调拨测试")
		if !errs.Is(err, errs.ErrBadRequest) {
			t.Fatalf("quantity=%d 应返回 ErrBadRequest，got %v", qty, err)
		}
	}
	// 库存必须原样未变
	after, err := repository.NewInventoryRepo(db).FindByKey(ctx, drug.ID, 2, "TR-NEG", exp, false)
	if err != nil {
		t.Fatalf("复查库存行失败: %v", err)
	}
	if after.Quantity != 10 {
		t.Fatalf("拒绝后库存应仍为 10，got %d", after.Quantity)
	}
}

// TestVoidChargeNetZero 红冲后患者净额必须归零，而非变成 -X。
// 回归：原单置 voided=true 被下游 "voided=FALSE" 过滤，而冲正单保持 voided=false，
// 结算与按患者计费报表只捞到 -X，凭空给患者减钱（甚至 payable<0 阻断结算）。
func TestVoidChargeNetZero(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	svc := service.NewClinicalService(db)

	cr, err := svc.CreateCharge(ctx, service.ChargeInput{
		PatientName: "红冲净额患者", ItemType: "clinical_service",
		ItemName: "雾化吸入", Quantity: 1, UnitPrice: 5000,
	}, 1, "收费员")
	if err != nil {
		t.Fatalf("创建计费失败: %v", err)
	}
	if err := svc.VoidCharge(ctx, cr.ID, 2, "红冲员"); err != nil {
		t.Fatalf("红冲失败: %v", err)
	}
	// 与下游报表/计价同口径：voided = false 的行求和应为 0
	var net int64
	if err := db.Model(&model.ChargeRecord{}).
		Where("voided = false AND patient_name = ?", "红冲净额患者").
		Select("COALESCE(SUM(amount),0)").Scan(&net).Error; err != nil {
		t.Fatalf("汇总净额失败: %v", err)
	}
	if net != 0 {
		t.Fatalf("红冲后净额应为 0，got %d（未红冲的负冲正单仍在下游口径内生效）", net)
	}
	// 冲正单本身应仍留痕（红冲审计不因净额归零而丢失）
	var n int64
	db.Model(&model.ChargeRecord{}).
		Where("ref_type = 'charge_void' AND ref_id = ?", cr.ID).Count(&n)
	if n != 1 {
		t.Fatalf("红冲冲正单应留痕 1 条，got %d", n)
	}
}

// TestMultiItemPartialReturnReachesReturned 多明细分次退药后，整方应能到达 returned 终态。
// 回归：终态判定此前基于「本次请求的明细集合」，分次退药时先退的明细在后续请求中查不到，
// allReturned 恒 false，处方永远卡在 dispensed，returned 终态不可达。
func TestMultiItemPartialReturnReachesReturned(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	inv := service.NewInventoryService(db)
	presc := service.NewPrescriptionService(db, inv, service.NewSpecialDrugService(db), nil, nil, service.NewClinicalService(db))

	drugA := mustCreateDrug(t, db)
	// 第二个药品：编码/通用名/规格/厂家全部不同，避开 uq_drug 一品一规一商唯一键
	drugB := &model.Drug{
		Code: "RTB001", GenericName: "退货测试药B", BrandName: "RTB", DosageForm: "胶囊",
		Specification: "0.25g", Manufacturer: "退货测试药厂B", ApprovalNumber: "国药准字H88888888",
		BaseUnit: "盒", SplitUnit: "粒", PackSize: 20, IsSplitAllowed: true,
		RetailPrice: 2000, PurchasePrice: 1500, Status: 1,
	}
	if err := service.NewDrugService(db).Create(ctx, drugB); err != nil {
		t.Fatalf("创建第二个药品失败: %v", err)
	}

	exp := time.Now().AddDate(1, 0, 0)
	if err := inv.StockIn(ctx, []service.StockEntry{
		{DrugID: drugA.ID, LocationID: 2, BatchNo: "RT-A", ExpiryDate: exp, IsSplit: false, Quantity: 20, UnitPrice: drugA.PurchasePrice},
		{DrugID: drugB.ID, LocationID: 2, BatchNo: "RT-B", ExpiryDate: exp, IsSplit: false, Quantity: 20, UnitPrice: drugB.PurchasePrice},
	}, 1, "退货测试"); err != nil {
		t.Fatalf("入库失败: %v", err)
	}

	p, err := presc.Create(ctx, service.PrescriptionInput{
		PatientName: "分次退货患者",
		Items: []service.PrescriptionItemInput{
			{DrugID: drugA.ID, Quantity: 2},
			{DrugID: drugB.ID, Quantity: 3},
		},
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

	items, err := repository.NewPrescriptionItemRepo(db).ListByPrescription(ctx, p.ID)
	if err != nil {
		t.Fatalf("查询处方明细失败: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("应有两个明细，got %d", len(items))
	}
	// 第一次只退明细 A
	if err := presc.Return(ctx, p.ID, []service.ReturnItemInput{
		{ItemID: items[0].ID, ReturnQuantity: items[0].DispensedQuantity},
	}, 5, "药师C"); err != nil {
		t.Fatalf("第一次退药失败: %v", err)
	}
	// 第二次只退明细 B —— 此处此前会因 A 缺席而无法置 returned
	if err := presc.Return(ctx, p.ID, []service.ReturnItemInput{
		{ItemID: items[1].ID, ReturnQuantity: items[1].DispensedQuantity},
	}, 5, "药师C"); err != nil {
		t.Fatalf("第二次退药失败: %v", err)
	}

	after, err := repository.NewPrescriptionRepo(db).GetByID(ctx, p.ID)
	if err != nil {
		t.Fatalf("复查处方失败: %v", err)
	}
	if after.Status != "returned" {
		t.Fatalf("全退后处方应进入 returned 终态，got %q", after.Status)
	}
}

// TestAutoSplitReturnRestoresToSplitRow 自动拆零处方的退药应回补拆零行，而非整盒行。
// 回归：need_split 发药记录按拆零形态（片）计量，但 rec.InventoryID 仍指向被拆开的
// 整盒行；直接回补会把「片」写进以「盒」计量的行，库存虚增一个整盒。
func TestAutoSplitReturnRestoresToSplitRow(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	inv := service.NewInventoryService(db)
	presc := service.NewPrescriptionService(db, inv, service.NewSpecialDrugService(db), nil, nil, service.NewClinicalService(db))
	drugSvc := service.NewDrugService(db)

	drug := &model.Drug{
		Code: "SPLITRT01", GenericName: "自动拆零退货药", DosageForm: "片剂",
		Specification: "0.1g", Manufacturer: "拆零退货药厂",
		BaseUnit: "盒", SplitUnit: "片", PackSize: 15, IsSplitAllowed: true,
		RetailPrice: 1500, PurchasePrice: 1200, Status: 1,
	}
	if err := drugSvc.Create(ctx, drug); err != nil {
		t.Fatalf("创建药品失败: %v", err)
	}

	exp := time.Now().AddDate(1, 0, 0)
	// 整盒 1 盒 + 拆零 2 片（拆零不足 → 触发自动拆零规划）
	if err := inv.StockIn(ctx, []service.StockEntry{
		{DrugID: drug.ID, LocationID: 2, BatchNo: "SR-W", ExpiryDate: exp, IsSplit: false, Quantity: 1, UnitPrice: 1200},
		{DrugID: drug.ID, LocationID: 2, BatchNo: "SR-S", ExpiryDate: exp, IsSplit: true, Quantity: 2, UnitPrice: 80},
	}, 1, "自动拆零退货测试"); err != nil {
		t.Fatalf("入库失败: %v", err)
	}

	p, err := presc.Create(ctx, service.PrescriptionInput{
		PatientName: "自动拆零退货患者",
		Items:       []service.PrescriptionItemInput{{DrugID: drug.ID, Quantity: 3}},
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

	// 发药后：整盒 0，拆零 = 0(原) + 14(开盒余片) = 14
	if q := wholeRow(t, db, drug.ID); q.Quantity != 0 {
		t.Fatalf("发药后整盒应剩 0，got %d", q.Quantity)
	}
	splitQty, _ := splitRowQty(t, db, drug.ID)
	if splitQty != 14 {
		t.Fatalf("发药后拆零应为 14 片，got %d", splitQty)
	}

	items, err := repository.NewPrescriptionItemRepo(db).ListByPrescription(ctx, p.ID)
	if err != nil {
		t.Fatalf("查询处方明细失败: %v", err)
	}
	if err := presc.Return(ctx, p.ID, []service.ReturnItemInput{
		{ItemID: items[0].ID, ReturnQuantity: 3},
	}, 5, "药师C"); err != nil {
		t.Fatalf("退药失败: %v", err)
	}

	// 退药 3 片 → 拆零行 14+3=17；整盒行必须仍为 0（回补到整盒行即为虚增 1 盒 = 15 片）
	splitQty, _ = splitRowQty(t, db, drug.ID)
	if splitQty != 17 {
		t.Fatalf("退药后拆零应为 17 片，got %d", splitQty)
	}
	if q := wholeRow(t, db, drug.ID); q.Quantity != 0 {
		t.Fatalf("退药不应凭空增加整盒库存（+1 即 15 片幽灵库存），got %d", q.Quantity)
	}
}

// TestDrugSetStatusCanDisable 药品停用必须真正落库。
// 回归：GORM 结构体 Updates 跳过零值，status=0 被静默忽略，接口 200 但药品未停用。
func TestDrugSetStatusCanDisable(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	drugSvc := service.NewDrugService(db)
	drug := mustCreateDrug(t, db)

	if err := drugSvc.SetStatus(ctx, drug.ID, 0); err != nil {
		t.Fatalf("停用药品失败: %v", err)
	}
	got, err := repository.NewDrugRepo(db).GetByID(ctx, drug.ID)
	if err != nil {
		t.Fatalf("复查药品失败: %v", err)
	}
	if got.Status != 0 {
		t.Fatalf("药品应已停用（status=0），got %d", got.Status)
	}
}

// TestUpdateUserKeepsRoleWhenOmitted 局部更新（仅改停用状态）不得清空角色。
// 回归：role 无条件赋值 → 省略该字段即写入空串，RequireRoles 全线 403，账号被静默锁死。
func TestUpdateUserKeepsRoleWhenOmitted(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	svc := service.NewAuthService(db, auth.NewManager("audit-round5-test-secret-0123456789", time.Hour))

	// 用专用测试账号：users 属种子表不在 TRUNCATE 列表内，改种子 admin 会污染其它用例；
	// 用户名带纳秒后缀保证重复运行不撞唯一约束。
	created, err := svc.CreateUser(ctx, &model.User{
		Username: fmt.Sprintf("audit-role-keep-%d", time.Now().UnixNano()),
		Name:     "角色保留测试", Role: "pharmacist", Phone: "13800000000",
	}, "Init-Passw0rd!")
	if err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}

	// 仅提交 {status:0}，name/role/phone 一律省略
	disabled := 0
	if err := svc.UpdateUser(ctx, created.ID, "", "", "", &disabled, ""); err != nil {
		t.Fatalf("局部更新用户失败: %v", err)
	}
	after, err := repository.NewUserRepo(db).GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("复查用户失败: %v", err)
	}
	if after.Role != created.Role {
		t.Fatalf("省略 role 时角色应保持 %q，实际变成 %q（账号会被锁死）", created.Role, after.Role)
	}
	if after.Name != created.Name {
		t.Fatalf("省略 name 时姓名应保持 %q，实际变成 %q", created.Name, after.Name)
	}
	if after.Phone != created.Phone {
		t.Fatalf("省略 phone 时电话应保持 %q，实际变成 %q", created.Phone, after.Phone)
	}
	if after.Status != 0 {
		t.Fatalf("显式传入的 status=0 应生效，got %d", after.Status)
	}
}

// TestPrescriptionUpdateStatusKeepsAuditFields 后续状态流转不得抹除已记录的审核痕迹。
// 回归：UpdateStatus 曾无条件写入 reviewed_at/auditor_id，
// 调配阶段若未回填就把已记录的审核人与审核时间清空。
func TestPrescriptionUpdateStatusKeepsAuditFields(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	inv := service.NewInventoryService(db)
	presc := service.NewPrescriptionService(db, inv, service.NewSpecialDrugService(db), nil, nil, service.NewClinicalService(db))
	drug := mustCreateDrug(t, db)

	exp := time.Now().AddDate(1, 0, 0)
	if err := inv.StockIn(ctx, []service.StockEntry{{
		DrugID: drug.ID, LocationID: 2, BatchNo: "AUDIT-1", ExpiryDate: exp,
		IsSplit: false, Quantity: 10, UnitPrice: drug.PurchasePrice,
	}}, 1, "审核痕迹测试"); err != nil {
		t.Fatalf("入库失败: %v", err)
	}
	p, err := presc.Create(ctx, service.PrescriptionInput{
		PatientName: "审核痕迹患者",
		Items:       []service.PrescriptionItemInput{{DrugID: drug.ID, Quantity: 1}},
	})
	if err != nil {
		t.Fatalf("创建处方失败: %v", err)
	}
	if err := presc.Submit(ctx, p.ID, 1, "药师A"); err != nil {
		t.Fatalf("提交失败: %v", err)
	}
	if _, err := presc.Review(ctx, p.ID, service.AuditInput{Action: "pass"}, 2, "药师B"); err != nil {
		t.Fatalf("审核失败: %v", err)
	}
	reviewed, err := repository.NewPrescriptionRepo(db).GetByID(ctx, p.ID)
	if err != nil {
		t.Fatalf("读取处方失败: %v", err)
	}
	if reviewed.ReviewedAt == nil || reviewed.AuditorID != 2 {
		t.Fatalf("审核后应记录审核人与审核时间，got auditor=%d reviewedAt=%v", reviewed.AuditorID, reviewed.ReviewedAt)
	}

	// 继续走调配：只更新 checker/dispensing 字段
	if err := presc.Dispense(ctx, p.ID, 3, "调配员", "pharmacist"); err != nil {
		t.Fatalf("调配失败: %v", err)
	}
	after, err := repository.NewPrescriptionRepo(db).GetByID(ctx, p.ID)
	if err != nil {
		t.Fatalf("复查处方失败: %v", err)
	}
	if after.ReviewedAt == nil || after.AuditorID != 2 || after.AuditorName != reviewed.AuditorName {
		t.Fatalf("调配不应抹除审核痕迹：auditor=%d reviewedAt=%v（审核时 %d/%v）",
			after.AuditorID, after.ReviewedAt, reviewed.AuditorID, reviewed.ReviewedAt)
	}
}

// TestResolveAlertRejectsAlreadyHandled 对已处置的预警重复提交应报状态冲突，
// 而不是返回成功让调用方误以为已落库（此前忽略 RowsAffected，静默丢单）。
func TestResolveAlertRejectsAlreadyHandled(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	inv := service.NewInventoryService(db)
	drug := mustCreateDrug(t, db)
	exp := time.Now().AddDate(1, 0, 0)
	if err := inv.StockIn(ctx, []service.StockEntry{{
		DrugID: drug.ID, LocationID: 2, BatchNo: "ALERT-1", ExpiryDate: exp,
		IsSplit: false, Quantity: 1, UnitPrice: drug.PurchasePrice,
	}}, 1, "预警处置测试"); err != nil {
		t.Fatalf("入库失败: %v", err)
	}
	alert := &model.StockAlert{
		DrugID: drug.ID, LocationID: 2, BatchNo: "ALERT-1",
		AlertType: "test", Message: "测试预警", Status: "open",
	}
	if err := repository.NewStockAlertRepo(db).Create(ctx, alert); err != nil {
		t.Fatalf("创建预警失败: %v", err)
	}
	if alert.ID == 0 {
		t.Fatal("创建预警后 ID 未回填")
	}
	if err := inv.ResolveAlert(ctx, alert.ID, "resolved", 1, "处置员"); err != nil {
		t.Fatalf("首次处置失败: %v", err)
	}
	if err := inv.ResolveAlert(ctx, alert.ID, "resolved", 1, "处置员"); !errs.Is(err, errs.ErrStateConflict) {
		t.Fatalf("重复处置应返回 ErrStateConflict，got %v", err)
	}
}

// TestReceiveQCResultUninspectedBlocks 收货时未登记质检结论（0）必须被入库门禁拦截。
// 回归：QCResult 带 default:1 时 0 会被覆盖为 1，HasUninspected 恒 false，
// 收货单可在零质检下入库（GSP 合规破口）。
func TestReceiveQCResultUninspectedBlocks(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	inv := service.NewInventoryService(db)
	poSvc := service.NewPurchaseService(db, inv)
	drug := mustCreateDrug(t, db)

	sup := &model.Supplier{Code: "QC-SUP", Name: "质检测试供应商", Status: 1}
	if err := repository.NewSupplierRepo(db).Create(ctx, sup); err != nil {
		t.Fatalf("创建供应商失败: %v", err)
	}
	po, err := poSvc.CreateOrder(ctx, sup.ID,
		[]service.POItemInput{{DrugID: drug.ID, Quantity: 10, UnitPrice: drug.PurchasePrice}},
		nil, "", 1)
	if err != nil {
		t.Fatalf("创建采购单失败: %v", err)
	}
	if err := poSvc.SubmitOrder(ctx, po.ID); err != nil {
		t.Fatalf("提交采购单失败: %v", err)
	}

	// qc_result 留空（0=未质检）
	receipt, err := poSvc.Receive(ctx, po.ID, []service.ReceiveItemInput{{
		OrderItemID:      mustFirstOrderItemID(t, db, po.ID),
		ReceivedQuantity: 10,
		BatchNo:          "QC-B1",
		ExpiryDate:       time.Now().AddDate(1, 0, 0),
		QCResult:         0,
	}}, 1)
	if err != nil {
		t.Fatalf("创建收货单失败: %v", err)
	}
	// 落库值必须保持 0，而不是被 default 覆盖为 1
	items, err := repository.NewReceiptItemRepo(db).ListByReceipt(ctx, receipt.ID)
	if err != nil {
		t.Fatalf("查询收货明细失败: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("应有一条收货明细，got %d", len(items))
	}
	if items[0].QCResult != 0 {
		t.Fatalf("未质检(0) 不应被改写为 1，got %d（质检门禁已失效）", items[0].QCResult)
	}
	// 入库必须被拦截
	if err := poSvc.CompleteReceipt(ctx, receipt.ID, 1, "收货员"); !errs.Is(err, errs.ErrQCNotComplete) {
		t.Fatalf("存在未质检项时入库应被拦截（ErrQCNotComplete），got %v", err)
	}
}

// TestReceiveQCResultOutOfRangeRejected 越界的 qc_result 必须直接拒绝。
func TestReceiveQCResultOutOfRangeRejected(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	inv := service.NewInventoryService(db)
	poSvc := service.NewPurchaseService(db, inv)
	drug := mustCreateDrug(t, db)

	sup2 := &model.Supplier{Code: "QC-SUP2", Name: "质检越界供应商", Status: 1}
	if err := repository.NewSupplierRepo(db).Create(ctx, sup2); err != nil {
		t.Fatalf("创建供应商失败: %v", err)
	}
	po, err := poSvc.CreateOrder(ctx, sup2.ID,
		[]service.POItemInput{{DrugID: drug.ID, Quantity: 5, UnitPrice: drug.PurchasePrice}},
		nil, "", 1)
	if err != nil {
		t.Fatalf("创建采购单失败: %v", err)
	}
	if err := poSvc.SubmitOrder(ctx, po.ID); err != nil {
		t.Fatalf("提交采购单失败: %v", err)
	}
	orderItemID := mustFirstOrderItemID(t, db, po.ID)
	_, err = poSvc.Receive(ctx, po.ID, []service.ReceiveItemInput{{
		OrderItemID: orderItemID, ReceivedQuantity: 5,
		BatchNo: "QC-B2", ExpiryDate: time.Now().AddDate(1, 0, 0),
		QCResult: 7,
	}}, 1)
	if !errs.Is(err, errs.ErrBadRequest) {
		t.Fatalf("qc_result=7 应返回 ErrBadRequest，got %v", err)
	}
}

func mustFirstOrderItemID(t *testing.T, db *gorm.DB, orderID int64) int64 {
	t.Helper()
	items, err := repository.NewPOItemRepo(db).ListByOrder(context.Background(), orderID)
	if err != nil {
		t.Fatalf("查询采购明细失败: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("采购单无明细")
	}
	return items[0].ID
}
