//go:build integration

// 第二轮审查回归测试：效期口径、软删唯一性、JWT 吊销、批量写入与并发 upsert。
package service_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"yaofang/internal/domain/rule"
	"yaofang/internal/model"
	"yaofang/internal/pkg/auth"
	"yaofang/internal/repository"
	"yaofang/internal/service"
)

// ───────────────────────── 效期口径（date-only） ─────────────────────────

// TestFEFOIncludesTodayExpiringBatch 「当天到期」的批次必须可被 FEFO 选到。
//
// 回归（已在真实 PostgreSQL 上实证）：expiry_date 是 DATE 列，PG 把它按会话时区
// 提升为当天 00:00 再与参数比较。原实现传 time.Now()（带时分秒），于是
//
//	00:00 >= 14:23  → false  → 当天到期批次被 FEFO 整日排除，无法发药；
//
// 而 LockExpired 用同一参数做 < 比较又把它判为已过期并锁定——两条路径自相矛盾。
// 域规则 rule.IsExpired 明确定义「效期早于今天才算过期」。
func TestFEFOIncludesTodayExpiringBatch(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	inv := service.NewInventoryService(db)
	drug := mustCreateDrug(t, db)

	today := time.Now()
	todayDate := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	entries := []struct {
		batch string
		exp   time.Time
	}{
		{"EXP-TODAY", todayDate},
		{"EXP-YESTERDAY", todayDate.AddDate(0, 0, -1)},
		{"EXP-NEXTYEAR", todayDate.AddDate(1, 0, 0)},
	}
	for _, e := range entries {
		if err := inv.StockIn(ctx, []service.StockEntry{{
			DrugID: drug.ID, LocationID: 2, BatchNo: e.batch, ExpiryDate: e.exp,
			IsSplit: false, Quantity: 10, UnitPrice: drug.PurchasePrice,
		}}, 1, "效期口径测试"); err != nil {
			t.Fatalf("入库 %s 失败: %v", e.batch, err)
		}
	}

	batches, err := repository.NewInventoryRepo(db).FindAvailableForDispenseUnit(ctx, drug.ID, 2, false)
	if err != nil {
		t.Fatalf("FEFO 查询失败: %v", err)
	}
	got := make(map[string]bool, len(batches))
	for _, b := range batches {
		got[b.BatchNo] = true
	}
	if !got["EXP-TODAY"] {
		t.Error("当天到期的批次应可被 FEFO 选到（当天到期不算过期）")
	}
	if !got["EXP-NEXTYEAR"] {
		t.Error("未过期批次应可被 FEFO 选到")
	}
	if got["EXP-YESTERDAY"] {
		t.Error("昨天到期的批次不应被 FEFO 选到")
	}
}

// TestLockExpiredSkipsTodayExpiringBatch 过期锁定必须与 FEFO 口径一致：
// 当天到期的批次不得被置为「过期锁定」，否则当天即失去发药资格。
func TestLockExpiredSkipsTodayExpiringBatch(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	inv := service.NewInventoryService(db)
	drug := mustCreateDrug(t, db)

	today := time.Now()
	todayDate := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	for _, e := range []struct {
		batch string
		exp   time.Time
	}{
		{"LOCK-TODAY", todayDate},
		{"LOCK-YESTERDAY", todayDate.AddDate(0, 0, -1)},
	} {
		if err := inv.StockIn(ctx, []service.StockEntry{{
			DrugID: drug.ID, LocationID: 2, BatchNo: e.batch, ExpiryDate: e.exp,
			IsSplit: false, Quantity: 10, UnitPrice: drug.PurchasePrice,
		}}, 1, "过期锁定测试"); err != nil {
			t.Fatalf("入库 %s 失败: %v", e.batch, err)
		}
	}

	if _, err := inv.LockExpiredBatches(ctx); err != nil {
		t.Fatalf("过期锁定失败: %v", err)
	}
	rows, _, err := repository.NewInventoryRepo(db).List(ctx,
		repository.InventoryListFilter{DrugID: drug.ID, LocationID: 2}, 0, 20)
	if err != nil {
		t.Fatalf("查询库存失败: %v", err)
	}
	seen := 0
	for _, r := range rows {
		switch r.BatchNo {
		case "LOCK-TODAY":
			seen++
			if r.Status == 2 {
				t.Error("当天到期的批次不应被置为过期锁定(status=2)")
			}
		case "LOCK-YESTERDAY":
			seen++
			if r.Status != 2 {
				t.Errorf("昨天到期的批次应被锁定(status=2)，got %d", r.Status)
			}
		}
	}
	if seen != 2 {
		t.Fatalf("应检查到 2 个批次，got %d", seen)
	}
}

// TestTodayExpiringBatchEndToEndDispense 当天到期批次应能走通「提交→审核→发药」全链路。
// 这是上面两个仓储修复的业务级回归护栏：任一路径口径漂移都会在此暴露为
// 「库存不足 / 批次分配不足」。
func TestTodayExpiringBatchEndToEndDispense(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	inv := service.NewInventoryService(db)
	presc := service.NewPrescriptionService(db, inv, service.NewSpecialDrugService(db), nil, nil, service.NewClinicalService(db))
	drug := mustCreateDrug(t, db)

	today := time.Now()
	todayDate := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	if err := inv.StockIn(ctx, []service.StockEntry{{
		DrugID: drug.ID, LocationID: 2, BatchNo: "E2E-TODAY", ExpiryDate: todayDate,
		IsSplit: false, Quantity: 10, UnitPrice: drug.PurchasePrice,
	}}, 1, "当天到期全链路测试"); err != nil {
		t.Fatalf("入库失败: %v", err)
	}

	p, err := presc.Create(ctx, service.PrescriptionInput{
		PatientName: "当天到期患者",
		Items:       []service.PrescriptionItemInput{{DrugID: drug.ID, Quantity: 2}},
	})
	if err != nil {
		t.Fatalf("创建处方失败: %v", err)
	}
	if err := presc.Submit(ctx, p.ID, 1, "药师A"); err != nil {
		t.Fatalf("提交失败（当天到期批次被误判为无货）: %v", err)
	}
	for _, fn := range []func() error{
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
	after, err := repository.NewPrescriptionRepo(db).GetByID(ctx, p.ID)
	if err != nil {
		t.Fatalf("复查处方失败: %v", err)
	}
	if after.Status != "dispensed" {
		t.Fatalf("当天到期批次应可正常发药，处方状态 got %q", after.Status)
	}
	// 库存应实际扣减。断言按拆零单位（LDU）合计：mustCreateDrug 的 PackSize=24 且允许拆零，
	// 发 2 片会触发自动拆盒（整盒 10→9，拆零行 +22），总 LDU 应为 240-2=238。
	rows, _, err := repository.NewInventoryRepo(db).List(ctx,
		repository.InventoryListFilter{DrugID: drug.ID, LocationID: 2}, 0, 20)
	if err != nil {
		t.Fatalf("查询库存失败: %v", err)
	}
	var totalLDU, packSize int64
	for _, r := range rows {
		if r.BatchNo != "E2E-TODAY" {
			continue
		}
		packSize = int64(drug.PackSize)
		totalLDU += rule.ToLDU(r.IsSplit, r.Quantity, int(packSize))
	}
	if want := 10*packSize - 2; totalLDU != want {
		t.Fatalf("发药 2 片后总 LDU 应为 %d，got %d", want, totalLDU)
	}
}

// ───────────────────────── 软删 + 全局唯一约束 ─────────────────────────

// TestSoftDeletedDrugUniqueSlotIsReusable 软删后同一「一品一规一商」槽位必须可重建。
// 回归：000001 的 uq_drug 是全表唯一，软删只是标记 deleted_at、行仍在表内，
// 于是该槽位被永久占用；而创建路径的预检走默认作用域查不到软删行，预检通过
// → INSERT 撞 23505 → 非业务错误 → 接口返回 500 而非 409。
//
// 注意：药品**编码**刻意不可复用（GetByCode 走 Unscoped，编码会出现在历史处方/账目上，
// 复用会破坏可追溯性），故本用例换用新编码、只验证唯一性槽位的可复用性。
func TestSoftDeletedDrugUniqueSlotIsReusable(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	drugSvc := service.NewDrugService(db)

	d := &model.Drug{
		Code: "SFTDEL001", GenericName: "软删复用药", BrandName: "SFT", DosageForm: "片剂",
		Specification: "0.25g", Manufacturer: "软删复用药厂", ApprovalNumber: "国药准字H77777777",
		BaseUnit: "盒", SplitUnit: "片", PackSize: 20, IsSplitAllowed: true,
		RetailPrice: 2000, PurchasePrice: 1500, Status: 1,
	}
	if err := drugSvc.Create(ctx, d); err != nil {
		t.Fatalf("创建药品失败: %v", err)
	}
	if err := drugSvc.Delete(ctx, d.ID); err != nil {
		t.Fatalf("软删药品失败: %v", err)
	}

	// 同通用名/规格/厂家/剂型 + 新编码 → 一品一规一商槽位应可复用（000038 部分唯一索引）
	d2 := &model.Drug{
		Code: "SFTDEL002", GenericName: d.GenericName, BrandName: d.BrandName, DosageForm: d.DosageForm,
		Specification: d.Specification, Manufacturer: d.Manufacturer, ApprovalNumber: "国药准字H77777777",
		BaseUnit: "盒", SplitUnit: "片", PackSize: 20, IsSplitAllowed: true,
		RetailPrice: 2000, PurchasePrice: 1500, Status: 1,
	}
	if err := drugSvc.Create(ctx, d2); err != nil {
		t.Fatalf("软删后应能重建同规格药品（一品一规一商槽位应可复用），got %v", err)
	}
	if d2.ID == d.ID {
		t.Fatal("重建应产生新的药品记录")
	}
	// 旧记录仍应保留（软删不可丢历史）
	if err := db.Unscoped().Model(&model.Drug{}).Where("id = ?", d.ID).
		Take(new(model.Drug)).Error; err != nil {
		t.Fatalf("软删记录应仍可追溯: %v", err)
	}
}

// TestSoftDeletedCategoryCodeIsReusable 分类编码在软删后应可重建。
// 分类编码不出现在历史单据的可追溯关键位上，故按部分唯一索引放开复用；
// 修复前会落库撞 23505 并以 500「系统异常」暴露。
func TestSoftDeletedCategoryCodeIsReusable(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	drugSvc := service.NewDrugService(db)

	// 分类编码列宽 20
	code := fmt.Sprintf("SFC%d", time.Now().UnixNano()%1e13)
	c := &model.DrugCategory{Code: code, Name: "软删分类", Status: 1}
	if err := drugSvc.CreateCategory(ctx, c); err != nil {
		t.Fatalf("创建分类失败: %v", err)
	}
	if err := drugSvc.DeleteCategory(ctx, c.ID); err != nil {
		t.Fatalf("软删分类失败: %v", err)
	}
	c2 := &model.DrugCategory{Code: code, Name: "软删分类-重建", Status: 1}
	if err := drugSvc.CreateCategory(ctx, c2); err != nil {
		t.Fatalf("软删后应能重建同编码分类，got %v", err)
	}
	if c2.ID == c.ID {
		t.Fatal("重建应产生新的分类记录")
	}
}

// TestLiveDuplicateStillRejected 转换部分唯一索引后，在用行重复仍必须被拒（且返回业务错误而非 500）。
func TestLiveDuplicateStillRejected(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	drugSvc := service.NewDrugService(db)

	d := &model.Drug{
		Code: "LIVEDUP001", GenericName: "重复拦截药", BrandName: "LD", DosageForm: "胶囊",
		Specification: "0.3g", Manufacturer: "重复拦截药厂", ApprovalNumber: "国药准字H66666666",
		BaseUnit: "盒", SplitUnit: "粒", PackSize: 12, IsSplitAllowed: true,
		RetailPrice: 1800, PurchasePrice: 1200, Status: 1,
	}
	if err := drugSvc.Create(ctx, d); err != nil {
		t.Fatalf("创建药品失败: %v", err)
	}
	dup := &model.Drug{
		Code: "LIVEDUP002", GenericName: d.GenericName, BrandName: d.BrandName, DosageForm: d.DosageForm,
		Specification: d.Specification, Manufacturer: d.Manufacturer, ApprovalNumber: "国药准字H66666667",
		BaseUnit: "盒", SplitUnit: "粒", PackSize: 12, IsSplitAllowed: true,
		RetailPrice: 1800, PurchasePrice: 1200, Status: 1,
	}
	if err := drugSvc.Create(ctx, dup); err == nil {
		t.Fatal("在用行的一品一规一商重复必须被拒绝")
	}
}

// ───────────────────────── JWT 吊销（口令版本） ─────────────────────────

// TestChangePasswordRevokesExistingTokens 改密后旧 token 必须立即失效。
// 回归：JWT 无服务端吊销机制，改密前签发的 token 在 TTL（默认 720h=30 天）内始终有效，
// 令牌泄露后事件响应方的「改密」并不能处置该令牌。
func TestChangePasswordRevokesExistingTokens(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	svc := service.NewAuthService(db, newRevocationTestManager())

	created, err := svc.CreateUser(ctx, &model.User{
		Username: fmt.Sprintf("revoke-%d", time.Now().UnixNano()),
		Name:     "吊销测试", Role: "pharmacist", Phone: "13900000000",
	}, "Old-Passw0rd!")
	if err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}
	login := func(pw string) string {
		tok, _, err := svc.Login(ctx, created.Username, pw)
		if err != nil {
			t.Fatalf("登录失败(%s): %v", pw, err)
		}
		return tok
	}
	oldTok := login("Old-Passw0rd!")

	// 改密
	if err := svc.ChangePassword(ctx, created.ID, "Old-Passw0rd!", "New-Passw0rd!"); err != nil {
		t.Fatalf("改密失败: %v", err)
	}

	mgr := newRevocationTestManager()
	if _, err := mgr.Parse(oldTok); err != nil {
		t.Fatalf("回归前提不成立：改密后旧 token 竟无法解析（签名层应仍有效）: %v", err)
	}
	// 库里口令版本应已自增
	after, err := repository.NewUserRepo(db).GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("复查用户失败: %v", err)
	}
	if after.TokenVersion == 0 {
		t.Fatal("改密应自增 token_version（否则旧 token 不会被吊销）")
	}

	// 新口令登录得到的新 token 版本应与库中一致
	newTok := login("New-Passw0rd!")
	claims, err := mgr.Parse(newTok)
	if err != nil {
		t.Fatalf("解析新 token 失败: %v", err)
	}
	if claims.Ver != after.TokenVersion {
		t.Fatalf("新 token 携带的版本(%d)应与库中一致(%d)", claims.Ver, after.TokenVersion)
	}
	// 旧 token 携带的版本应小于库中 → 中间件将判定为已吊销
	oldClaims, err := mgr.Parse(oldTok)
	if err != nil {
		t.Fatalf("解析旧 token 失败: %v", err)
	}
	if oldClaims.Ver == after.TokenVersion {
		t.Fatal("旧 token 的版本号应与库中不同，否则吊销无效")
	}
}

// TestAdminResetPasswordBumpsTokenVersion 管理员重置口令同样要作废存量 token。
func TestAdminResetPasswordBumpsTokenVersion(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	svc := service.NewAuthService(db, newRevocationTestManager())

	created, err := svc.CreateUser(ctx, &model.User{
		Username: fmt.Sprintf("reset-%d", time.Now().UnixNano()),
		Name:     "重置测试", Role: "pharmacist", Phone: "13900000001",
	}, "Old-Passw0rd!")
	if err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}
	before, err := repository.NewUserRepo(db).GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("读取用户失败: %v", err)
	}
	// 管理员重置口令（其余字段省略）
	if err := svc.UpdateUser(ctx, created.ID, "", "", "", nil, "Reset-Passw0rd!"); err != nil {
		t.Fatalf("重置口令失败: %v", err)
	}
	after, err := repository.NewUserRepo(db).GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("复查用户失败: %v", err)
	}
	if after.TokenVersion <= before.TokenVersion {
		t.Fatalf("管理员重置口令应自增 token_version（%d → %d）", before.TokenVersion, after.TokenVersion)
	}
	// 新口令应可登录
	if _, _, err := svc.Login(ctx, created.Username, "Reset-Passw0rd!"); err != nil {
		t.Fatalf("重置后的新口令应可登录: %v", err)
	}
}

// ───────────────────────── 批量写入 / 并发 upsert ─────────────────────────

// TestBatchCreateWithEmptySliceIsNoOp 空明细批量写入应是无事发生，而非 500。
// 回归：GORM 对空切片 Create 返回 ErrEmptySlice（已在 gorm@v1.31.2 callbacks/create.go 核实），
// 会作为未知错误上报为「系统异常」。
func TestBatchCreateWithEmptySliceIsNoOp(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	if err := repository.NewPrescriptionItemRepo(db).CreateBatch(ctx, nil); err != nil {
		t.Fatalf("空明细写入应返回 nil，got %v", err)
	}
	if err := repository.NewDispenseRecordRepo(db).CreateBatch(ctx, nil); err != nil {
		t.Fatalf("空明细写入应返回 nil，got %v", err)
	}
	if err := repository.NewReceiptItemRepo(db).CreateBatch(ctx, nil); err != nil {
		t.Fatalf("空明细写入应返回 nil，got %v", err)
	}
	if err := repository.NewStocktakeItemRepo(db).CreateBatch(ctx, nil); err != nil {
		t.Fatalf("空明细写入应返回 nil，got %v", err)
	}
}

// TestStockSettingUpsertConcurrent 并发首次配置同一 (drug, location) 不得报唯一键冲突。
// 回归：「先 First 后 Create」两步在并发下双方都读到 not-found、都走 Create，一方撞
// uq_stock_setting；而 PostgreSQL 下唯一冲突会把事务置入 aborted(25P02)，整次配置以 500 收场。
func TestStockSettingUpsertConcurrent(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	drug := mustCreateDrug(t, db)
	repo := repository.NewStockSettingRepo(db)

	const workers = 8
	errCh := make(chan error, workers)
	start := make(chan struct{})
	for i := 0; i < workers; i++ {
		go func(n int) {
			<-start
			errCh <- repo.Upsert(ctx, &model.DrugStockSetting{
				DrugID: drug.ID, LocationID: 2,
				MinQuantity: int64(n), MaxQuantity: 100, ReorderQty: 10, IsEnabled: true,
			})
		}(i)
	}
	close(start)
	for i := 0; i < workers; i++ {
		if err := <-errCh; err != nil {
			t.Fatalf("并发 Upsert 出现错误（唯一键冲突/aborted）: %v", err)
		}
	}
	rows, err := repo.ListEnabled(ctx)
	if err != nil {
		t.Fatalf("查询设置失败: %v", err)
	}
	n := 0
	for _, r := range rows {
		if r.DrugID == drug.ID && r.LocationID == 2 {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("并发 Upsert 后应恰有 1 条设置，got %d", n)
	}
}

// newRevocationTestManager 构造固定密钥的 JWT 管理器（登录与校验须用同一密钥）。
func newRevocationTestManager() *auth.Manager {
	return auth.NewManager("revocation-test-secret-0123456789abcdef", time.Hour)
}
