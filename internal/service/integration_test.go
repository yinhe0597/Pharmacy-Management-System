//go:build integration

// 集成测试：连接真实 PostgreSQL，验证采购入库→处方全链路→退药回补。
package service_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"gorm.io/gorm"

	"yaofang/internal/config"
	"yaofang/internal/domain/enum"
	"yaofang/internal/model"
	"yaofang/internal/pkg/errs"
	"yaofang/internal/pkg/money"
	"yaofang/internal/pkg/seq"
	"yaofang/internal/repository"
	"yaofang/internal/server"
	"yaofang/internal/service"
)

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// testCleanupTables 集成测试前清空的业务表（TRUNCATE ... CASCADE）。
// 原则：种子/引用表（users/system_settings/库房/分类/参考目录/规则种子）不清，
// 其余业务表必须全覆盖；新增表同步加入（由 TestCleanupCoversAllTables 强制）。
var testCleanupTables = []string{
	"prescription_audit_logs", "prescription_dispense_records", "prescription_items",
	"prescriptions", "stock_reservations", "inventory_transactions", "inventory",
	"drug_stock_settings",
	"stocktake_items", "stocktakes", "split_orders",
	"requisition_order_items", "requisition_orders",
	"special_drug_ledgers", "ampoule_returns",
	"purchase_receipt_items", "purchase_receipts", "purchase_order_items", "purchase_orders",
	"drug_suppliers", "drug_interactions", "interaction_results", "drug_ingredients", "suppliers", "drugs", "stock_alerts",
	"patients", "patient_allergies",
	"visits", "medical_records", "medical_record_diagnoses",
	"charges", "charge_items", "charge_records", "clinical_services",
	"consultations", "adverse_reactions", "medication_guidances",
	"notifications", "notification_reads", "operation_logs",
}

// getenvPort 读取端口类环境变量（供 PG 多版本兼容性测试指定不同端口）。
func getenvPort(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			return p
		}
	}
	return def
}

// sharedDB 集成测试共用的数据库句柄（整个测试二进制只建一次连接池）。
//
// 此前每个用例各自 OpenDB，得到独立连接池且从不关闭；用例数一多，
// 累计打开的连接数会超过 PostgreSQL 的 max_connections，后续用例随机报
// "sorry, too many clients already"，把基础设施问题伪装成业务失败。
var (
	sharedDBOnce sync.Once
	sharedDB     *gorm.DB
	sharedDBErr  error
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	sharedDBOnce.Do(func() {
		// 集成测试只需要数据库连接参数；config.Load() 会校验 JWT 密钥强度（弱/占位密钥拒绝加载），
		// 而 CI 环境没有 configs/config.yaml（viper 默认值为占位密钥），故此处注入测试专用强密钥，
		// 使集成测试不依赖本地配置文件与外部环境变量。
		os.Setenv("YF_AUTH_JWT_SECRET", "integration-test-only-jwt-secret-0123456789abcdef")
		cfg, err := config.Load()
		if err != nil {
			sharedDBErr = fmt.Errorf("加载配置失败: %w", err)
			return
		}
		// 显式指定测试库连接，不依赖工作目录下的配置文件（端口可用 YF_TEST_DB_PORT 覆盖，便于 PG 多版本矩阵测试）
		cfg.Database.Host = getenv("YF_TEST_DB_HOST", "127.0.0.1")
		cfg.Database.Port = getenvPort("YF_TEST_DB_PORT", 5432)
		cfg.Database.User = getenv("YF_TEST_DB_USER", "yaofang")
		cfg.Database.Password = getenv("YF_TEST_DB_PASSWORD", "yaofang123")
		cfg.Database.Name = "yaofang"
		sharedDB, sharedDBErr = server.OpenDB(&cfg.Database)
	})
	if sharedDBErr != nil {
		t.Fatalf("初始化测试数据库失败: %v", sharedDBErr)
	}
	db := sharedDB
	// 单号号段分配器（正常由 server.NewApp 初始化；此处直接构造 service，需显式注入）
	seq.Init(db, 0)
	// 清空涉及表（CASCADE 处理外键）
	// 注意：必须覆盖所有会跨用例残留状态的业务表——尤其 stocktakes（盘点中的库房会
	// 禁止出入库，残留会使后续用例随机失败），以及拆零单/领用单/专账等。
	// testCleanupTables 提取为包变量，供 TestCleanupCoversAllTables 守护：
	// 新增业务表时必须同步加入，否则守护测试失败（docs/07 §7）。
	for _, tb := range testCleanupTables {
		if err := db.Exec("TRUNCATE TABLE " + tb + " RESTART IDENTITY CASCADE").Error; err != nil {
			t.Fatalf("清空表 %s 失败: %v", tb, err)
		}
	}
	return db
}

func mustCreateDrug(t *testing.T, db *gorm.DB) *model.Drug {
	t.Helper()
	drugSvc := service.NewDrugService(db)
	d := &model.Drug{
		Code: "IT001", GenericName: "阿莫西林胶囊", BrandName: "IT", DosageForm: "胶囊剂",
		Specification: "0.5g", Manufacturer: "集成测试药厂", ApprovalNumber: "国药准字H99999999",
		BaseUnit: "盒", SplitUnit: "粒", PackSize: 24, IsSplitAllowed: true,
		RetailPrice: 2400, PurchasePrice: 1800,
		AntibioticLevel: 1, Status: 1,
	}
	if err := drugSvc.Create(context.Background(), d); err != nil {
		t.Fatalf("创建药品失败: %v", err)
	}
	if d.SplitRetailPrice != 100 {
		t.Fatalf("拆零价计算错误: got %d want 100", d.SplitRetailPrice)
	}
	return d
}

func mustCreatePatient(t *testing.T, db *gorm.DB) *model.Patient {
	t.Helper()
	pat := &model.Patient{CardNo: fmt.Sprintf("IT-PAT-%d", time.Now().UnixNano()), Name: "集成测试患者", Gender: "女", Age: "35岁"}
	if err := repository.NewPatientRepo(db).Create(context.Background(), pat); err != nil {
		t.Fatalf("创建患者失败: %v", err)
	}
	return pat
}

func TestFullChain(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	inv := service.NewInventoryService(db)
	presc := service.NewPrescriptionService(db, inv, service.NewSpecialDrugService(db), nil, nil, service.NewClinicalService(db))

	drug := mustCreateDrug(t, db)
	today := time.Now()

	// 1. 其他入库 10 盒（批次 A，效期一年后）
	expiry := today.AddDate(1, 0, 0)
	err := inv.StockIn(ctx, []service.StockEntry{{
		DrugID: drug.ID, LocationID: 2, BatchNo: "BATCH-A",
		ExpiryDate: expiry, IsSplit: false, Quantity: 10, UnitPrice: drug.PurchasePrice,
	}}, 1, "集成测试")
	if err != nil {
		t.Fatalf("入库失败: %v", err)
	}

	// 2. 拆零 1 盒 → 24 粒
	wholeRows, _, err := repository.NewInventoryRepo(db).List(ctx, repository.InventoryListFilter{DrugID: drug.ID, LocationID: 2}, 0, 10)
	if err != nil || len(wholeRows) == 0 {
		t.Fatalf("查询整盒库存失败: %v", err)
	}
	if err := inv.Split(ctx, service.SplitRequest{InventoryID: wholeRows[0].ID, Packs: 1}, 1, "集成测试"); err != nil {
		t.Fatalf("拆零失败: %v", err)
	}
	splitRows, _, _ := repository.NewInventoryRepo(db).List(ctx, repository.InventoryListFilter{DrugID: drug.ID, LocationID: 2}, 0, 10)
	var splitQty int64
	for _, r := range splitRows {
		if r.IsSplit {
			splitQty = r.Quantity
		}
	}
	if splitQty != 24 {
		t.Fatalf("拆零后拆零库存应为24粒, got %d", splitQty)
	}

	// 3. 处方：拆零发 12 粒
	p, err := presc.Create(ctx, service.PrescriptionInput{
		PatientName: "张三", PatientAge: "35岁",
		Items: []service.PrescriptionItemInput{
			{DrugID: drug.ID, Quantity: 12, IsSplit: true, SingleDose: 2, TotalDailyDose: 4, Days: 3, Frequency: "bid"},
		},
	})
	if err != nil {
		t.Fatalf("创建处方失败: %v", err)
	}

	// 4. 提交（预占 12 粒）
	if err := presc.Submit(ctx, p.ID, 1, "药师A"); err != nil {
		t.Fatalf("提交失败: %v", err)
	}
	resvs, _ := repository.NewStockReservationRepo(db).ListActiveByRef(ctx, "prescription", p.ID)
	var reserved int64
	for _, r := range resvs {
		reserved += r.Quantity
	}
	if reserved != 12 {
		t.Fatalf("预占数量应为12, got %d", reserved)
	}

	// 5. 审核通过
	if _, err := presc.Review(ctx, p.ID, service.AuditInput{Action: "pass"}, 2, "药师B"); err != nil {
		t.Fatalf("审核失败: %v", err)
	}

	// 6. 调配
	if err := presc.Dispense(ctx, p.ID, 3, "调配员", "pharmacist"); err != nil {
		t.Fatalf("调配失败: %v", err)
	}

	// 7. 发药确认（核对）
	if err := presc.ConfirmDispense(ctx, p.ID, 4, "核对员", "pharmacist"); err != nil {
		t.Fatalf("发药确认失败: %v", err)
	}
	// 拆零库存应扣减 12 → 12
	splitRows, _, _ = repository.NewInventoryRepo(db).List(ctx, repository.InventoryListFilter{DrugID: drug.ID, LocationID: 2}, 0, 10)
	for _, r := range splitRows {
		if r.IsSplit && r.Quantity != 12 {
			t.Fatalf("发药后拆零库存应为12, got %d", r.Quantity)
		}
	}

	// 8. 部分退药 6 粒 → 拆零库存回补到 18
	items, _ := repository.NewPrescriptionItemRepo(db).ListByPrescription(ctx, p.ID)
	if len(items) == 0 {
		t.Fatal("处方应含明细")
	}
	if err := presc.Return(ctx, p.ID, []service.ReturnItemInput{{ItemID: items[0].ID, ReturnQuantity: 6}}, 5, "药师C"); err != nil {
		t.Fatalf("退药失败: %v", err)
	}
	splitRows, _, _ = repository.NewInventoryRepo(db).List(ctx, repository.InventoryListFilter{DrugID: drug.ID, LocationID: 2}, 0, 10)
	for _, r := range splitRows {
		if r.IsSplit && r.Quantity != 18 {
			t.Fatalf("退药后拆零库存应为18, got %d", r.Quantity)
		}
	}
}

func TestCancelReleasesReservation(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	inv := service.NewInventoryService(db)
	presc := service.NewPrescriptionService(db, inv, service.NewSpecialDrugService(db), nil, nil, service.NewClinicalService(db))
	drug := mustCreateDrug(t, db)

	if err := inv.StockIn(ctx, []service.StockEntry{{
		DrugID: drug.ID, LocationID: 2, BatchNo: "BATCH-B",
		ExpiryDate: time.Now().AddDate(1, 0, 0), IsSplit: false, Quantity: 5, UnitPrice: drug.PurchasePrice,
	}}, 1, "集成测试"); err != nil {
		t.Fatalf("入库失败: %v", err)
	}
	// 混合发药：Quantity 按 LDU（48 = 2 整盒），整盒部分走整盒库存
	p, err := presc.Create(ctx, service.PrescriptionInput{
		PatientName: "李四",
		Items:       []service.PrescriptionItemInput{{DrugID: drug.ID, Quantity: 48}},
	})
	if err != nil {
		t.Fatalf("创建处方失败: %v", err)
	}
	if err := presc.Submit(ctx, p.ID, 1, "药师A"); err != nil {
		t.Fatalf("提交失败: %v", err)
	}
	if err := presc.Cancel(ctx, p.ID, "药师A"); err != nil {
		t.Fatalf("作废失败: %v", err)
	}
	resvs, _ := repository.NewStockReservationRepo(db).ListActiveByRef(ctx, "prescription", p.ID)
	if len(resvs) != 0 {
		t.Fatalf("作废后应无 active 预占, got %d", len(resvs))
	}
	rows, _, _ := repository.NewInventoryRepo(db).List(ctx, repository.InventoryListFilter{DrugID: drug.ID, LocationID: 2}, 0, 10)
	for _, r := range rows {
		if r.ReservedQuantity != 0 {
			t.Fatalf("作废后预占应归零, got %d", r.ReservedQuantity)
		}
	}
	_ = money.Cents(0) // keep import
	_ = enum.TxnPurchaseIn
}

// TestSubmitIdempotent 重复提交不得重复预占。
func TestSubmitIdempotent(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	inv := service.NewInventoryService(db)
	presc := service.NewPrescriptionService(db, inv, service.NewSpecialDrugService(db), nil, nil, service.NewClinicalService(db))
	drug := mustCreateDrug(t, db)
	if err := inv.StockIn(ctx, []service.StockEntry{{
		DrugID: drug.ID, LocationID: 2, BatchNo: "BATCH-D",
		ExpiryDate: time.Now().AddDate(1, 0, 0), IsSplit: false, Quantity: 5, UnitPrice: drug.PurchasePrice,
	}}, 1, "集成测试"); err != nil {
		t.Fatalf("入库失败: %v", err)
	}
	p, err := presc.Create(ctx, service.PrescriptionInput{
		PatientName: "幂等患者",
		Items:       []service.PrescriptionItemInput{{DrugID: drug.ID, Quantity: 72}},
	})
	if err != nil {
		t.Fatalf("创建处方失败: %v", err)
	}
	for i := 0; i < 2; i++ {
		if err := presc.Submit(ctx, p.ID, 1, "药师"); err != nil {
			t.Fatalf("第%d次提交失败: %v", i+1, err)
		}
	}
	resvs, _ := repository.NewStockReservationRepo(db).ListActiveByRef(ctx, "prescription", p.ID)
	var total int64
	for _, r := range resvs {
		total += r.Quantity
	}
	if total != 3 {
		t.Fatalf("重复提交后总预占应为3, got %d", total)
	}
}

// TestSplitReservedBlocked 已预占的整盒不可拆零。
func TestSplitReservedBlocked(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	inv := service.NewInventoryService(db)
	presc := service.NewPrescriptionService(db, inv, service.NewSpecialDrugService(db), nil, nil, service.NewClinicalService(db))
	drug := mustCreateDrug(t, db)
	if err := inv.StockIn(ctx, []service.StockEntry{{
		DrugID: drug.ID, LocationID: 2, BatchNo: "BATCH-E",
		ExpiryDate: time.Now().AddDate(1, 0, 0), IsSplit: false, Quantity: 5, UnitPrice: drug.PurchasePrice,
	}}, 1, "集成测试"); err != nil {
		t.Fatalf("入库失败: %v", err)
	}
	p, err := presc.Create(ctx, service.PrescriptionInput{
		PatientName: "拆零患者",
		Items:       []service.PrescriptionItemInput{{DrugID: drug.ID, Quantity: 48}},
	})
	if err != nil {
		t.Fatalf("创建处方失败: %v", err)
	}
	if err := presc.Submit(ctx, p.ID, 1, "药师"); err != nil {
		t.Fatalf("提交失败: %v", err)
	}
	rows, _, _ := repository.NewInventoryRepo(db).List(ctx, repository.InventoryListFilter{DrugID: drug.ID, LocationID: 2}, 0, 10)
	if len(rows) == 0 {
		t.Fatal("无库存行")
	}
	// 预占 2 盒后可用 3 盒，拆 4 盒应被拦截（2001 库存不足）
	err = inv.Split(ctx, service.SplitRequest{InventoryID: rows[0].ID, Packs: 4}, 1, "集成测试")
	if err == nil {
		t.Fatal("拆零已预占库存应被拦截")
	}
	var e *errs.Error
	if errors.As(err, &e) && e.Code == 2001 {
		return
	}
	t.Fatalf("期望库存不足(2001), got %v", err)
}

// TestConcurrentReserveNoOversell 并发预占不超卖：单批 5 盒，两个处方并发各预占 3 盒，
// 总预占必须 ≤5 盒（条件更新兜底），且无负可用量。
func TestConcurrentReserveNoOversell(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	inv := service.NewInventoryService(db)
	presc := service.NewPrescriptionService(db, inv, service.NewSpecialDrugService(db), nil, nil, service.NewClinicalService(db))
	drug := mustCreateDrug(t, db)

	if err := inv.StockIn(ctx, []service.StockEntry{{
		DrugID: drug.ID, LocationID: 2, BatchNo: "BATCH-C",
		ExpiryDate: time.Now().AddDate(1, 0, 0), IsSplit: false, Quantity: 5, UnitPrice: drug.PurchasePrice,
	}}, 1, "集成测试"); err != nil {
		t.Fatalf("入库失败: %v", err)
	}

	const n = 2
	ids := make([]int64, n)
	for i := 0; i < n; i++ {
		p, err := presc.Create(ctx, service.PrescriptionInput{
			PatientName: "并发患者",
			Items:       []service.PrescriptionItemInput{{DrugID: drug.ID, Quantity: 72}},
		})
		if err != nil {
			t.Fatalf("创建处方 %d 失败: %v", i, err)
		}
		ids[i] = p.ID
	}

	start := make(chan struct{})
	errs := make([]error, n)
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(idx int) {
			defer wg.Done()
			<-start
			errs[idx] = presc.Submit(ctx, ids[idx], 1, "并发药师")
		}(i)
	}
	close(start)
	wg.Wait() // 等待全部并发提交完成后再校验

	totalReserved := int64(0)
	for i := 0; i < n; i++ {
		if errs[i] != nil {
			t.Fatalf("处方 %d 提交失败: %v", i, errs[i])
		}
		resvs, _ := repository.NewStockReservationRepo(db).ListActiveByRef(ctx, "prescription", ids[i])
		for _, r := range resvs {
			totalReserved += r.Quantity
		}
	}
	if totalReserved > 5 {
		t.Fatalf("并发预占超卖：总预占 %d > 可用 5", totalReserved)
	}
	// 总预占必然为 5（5盒可全部被预占，3+2 或 2+3）
	if totalReserved != 5 {
		t.Fatalf("总预占应为 5（短缺方仍完成部分预占）, got %d", totalReserved)
	}
	// 无负可用量
	var row model.Inventory
	if err := db.Where("drug_id = ? AND location_id = 2 AND batch_no = 'BATCH-C'", drug.ID).First(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.Available() < 0 {
		t.Fatalf("出现负可用量: %d", row.Available())
	}
}
