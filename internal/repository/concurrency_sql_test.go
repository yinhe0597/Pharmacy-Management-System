package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"yaofang/internal/model"
)

// newDryRunDB 构造一个「不连接数据库」的 PostgreSQL 方言 DB，用于断言生成的 SQL。
// sql.Open 是惰性的，且该驱动 Initialize 不做版本探测，因此无需真实数据库即可构建语句。
func newDryRunDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(
		postgres.New(postgres.Config{
			DSN: "host=127.0.0.1 user=test password=test dbname=test sslmode=disable",
		}),
		&gorm.Config{DryRun: true, DisableAutomaticPing: true, Logger: logger.Discard},
	)
	if err != nil {
		t.Fatalf("构造 DryRun DB 失败: %v", err)
	}
	return db.Session(&gorm.Session{DryRun: true, SkipDefaultTransaction: true})
}

// captureSQL 在指定回调阶段捕获生成的 SQL（DryRun 下仅构建语句、不执行）。
func captureSQL(t *testing.T, db *gorm.DB, kind string, fn func() error) string {
	t.Helper()
	var captured string
	name := "test:capture_" + kind

	register := func(cb interface {
		Register(string, func(*gorm.DB)) error
	}) {
		if err := cb.Register(name, func(tx *gorm.DB) {
			if tx.Statement != nil {
				captured = tx.Statement.SQL.String()
			}
		}); err != nil {
			t.Fatalf("注册回调失败: %v", err)
		}
	}
	switch kind {
	case "create":
		register(db.Callback().Create().After("gorm:create"))
	case "update":
		register(db.Callback().Update().After("gorm:update"))
	default:
		t.Fatalf("未知回调类型: %s", kind)
	}
	defer func() {
		switch kind {
		case "create":
			_ = db.Callback().Create().Remove(name)
		case "update":
			_ = db.Callback().Update().Remove(name)
		}
	}()

	if err := fn(); err != nil {
		t.Fatalf("DryRun 执行返回错误: %v", err)
	}
	if captured == "" {
		t.Fatal("未捕获到 SQL 语句（DryRun 下应仍会构建 SQL）")
	}
	return strings.Join(strings.Fields(captured), " ")
}

// TestUpsertAddQuantitySQL 断言并发入库使用的 upsert 生成
// 「INSERT ... ON CONFLICT (五键) DO UPDATE ... RETURNING」，而不是「先查后插 + 冲突回退」。
// 后者在 PostgreSQL 唯一键冲突后事务进入 aborted 状态（25P02），回退查询必然失败。
func TestUpsertAddQuantitySQL(t *testing.T) {
	db := newDryRunDB(t)
	repo := NewInventoryRepo(db)
	inv := &model.Inventory{
		DrugID: 1, LocationID: 2, BatchNo: "B20260912",
		ExpiryDate: time.Date(2027, 9, 30, 0, 0, 0, 0, time.UTC),
		Quantity:   10, IsSplit: false, UnitPrice: 100, Status: 1,
	}

	sql := captureSQL(t, db, "create", func() error {
		_, err := repo.UpsertAddQuantity(context.Background(), inv)
		return err
	})

	for _, want := range []string{"INSERT INTO \"inventory\"", "ON CONFLICT", "DO UPDATE SET", "RETURNING"} {
		if !strings.Contains(sql, want) {
			t.Errorf("upsert SQL 缺少 %q：%s", want, sql)
		}
	}
	t.Logf("upsert SQL: %s", sql)
	for _, key := range []string{"drug_id", "location_id", "batch_no", "expiry_date", "is_split"} {
		if !strings.Contains(sql, key) {
			t.Errorf("upsert 冲突键缺少列 %q：%s", key, sql)
		}
	}
	if strings.Contains(sql, "SELECT") {
		t.Errorf("upsert 不应包含先查后插（SELECT）：%s", sql)
	}
}

// TestUpdateReceivedWithinLimitSQL 断言采购收货累加是「同一条 UPDATE 内条件 + 累加」，
// 而不是无锁读校验加盲累加（后者并发完成多张收货单会超收）。
func TestUpdateReceivedWithinLimitSQL(t *testing.T) {
	db := newDryRunDB(t)
	repo := NewPOItemRepo(db)

	sql := captureSQL(t, db, "update", func() error {
		_, err := repo.UpdateReceivedWithinLimit(context.Background(), 7, 5)
		return err
	})

	if !strings.Contains(sql, "received_quantity + ") {
		t.Errorf("条件累加 SQL 缺少累加表达式：%s", sql)
	}
	if !strings.Contains(sql, "<= quantity") && !strings.Contains(sql, "<= \"quantity\"") {
		t.Errorf("条件累加 SQL 缺少超收护栏（received_quantity + ? <= quantity）：%s", sql)
	}
	if strings.Contains(sql, "SELECT") {
		t.Errorf("条件累加不应先 SELECT 再 UPDATE：%s", sql)
	}
}

// TestUpdateCountedSQL 断言盘点差异写入口径为「实盘 - 账面」，而非直接写实盘数。
func TestUpdateCountedSQL(t *testing.T) {
	db := newDryRunDB(t)
	repo := NewStocktakeItemRepo(db)

	sql := captureSQL(t, db, "update", func() error {
		return repo.UpdateCounted(context.Background(), 3, 42)
	})

	if !strings.Contains(sql, "difference") || !strings.Contains(sql, "book_quantity") {
		t.Errorf("盘点差异未按「实盘 - 账面」计算：%s", sql)
	}
	if !strings.Contains(sql, "counted_quantity") {
		t.Errorf("盘点录入 SQL 缺少 counted_quantity：%s", sql)
	}
}
