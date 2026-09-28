//go:build integration

// 预警列表行 DTO 回归：前端预警表按 drug_name / location_name / days_left 渲染，
// 此前 StockAlertRepo.List 是裸 Find(model.StockAlert{})，三列皆无 → 整表三列空白。
//
// 另锁定一个已实测的 SQL 陷阱：date - date 在 PG 返回 integer 而非 interval，
// 套 EXTRACT(DAY FROM ...) 会因缺少 extract(unknown, integer) 重载导致整个接口运行期报错。
package service_test

import (
	"context"
	"testing"
	"time"

	"yaofang/internal/model"
	"yaofang/internal/repository"
	"yaofang/internal/service"
)

// TestAlertListRowCarriesDisplayFields 预警列表必须带出药品名、库房名与剩余效期天数。
func TestAlertListRowCarriesDisplayFields(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	drug := mustCreateDrug(t, db)

	// 库房 2 由迁移种子保证存在（与既有 StockIn 测试一致）
	locName := "主药库"
	var loc model.InventoryLocation
	if err := db.WithContext(ctx).Where("id = ?", 2).First(&loc).Error; err != nil {
		t.Fatalf("读取库房 2 失败: %v", err)
	}
	locName = loc.Name

	// 30 天后到期：days_left 期望 30
	expiry := time.Now().Truncate(24*time.Hour).AddDate(0, 0, 30)
	alert := &model.StockAlert{
		DrugID:     drug.ID,
		LocationID: 2,
		BatchNo:    "ALERTROW-1",
		ExpiryDate: &expiry,
		AlertType:  "expiry",
		Message:    "行 DTO 回归",
		Status:     "open",
	}
	if err := repository.NewStockAlertRepo(db).Create(ctx, alert); err != nil {
		t.Fatalf("创建预警失败: %v", err)
	}

	rows, total, err := repository.NewStockAlertRepo(db).List(ctx, "expiry", "open", 0, 50)
	if err != nil {
		t.Fatalf("查询预警列表失败（若为 extract 重载错误则本用例即回归守卫）: %v", err)
	}
	if total == 0 {
		t.Fatal("预警列表为空")
	}

	var got *repository.StockAlertRow
	for i := range rows {
		if rows[i].ID == alert.ID {
			got = &rows[i]
			break
		}
	}
	if got == nil {
		t.Fatalf("预警 %d 未出现在列表中", alert.ID)
	}
	if got.DrugName == "" {
		t.Error("drug_name 为空：预警行 DTO 未 JOIN drugs，前端「药品」列整列空白")
	}
	if got.LocationName != locName {
		t.Errorf("location_name = %q, 期望 %q：未 JOIN inventory_locations", got.LocationName, locName)
	}
	if got.DaysLeft == nil {
		t.Fatal("days_left 为 NULL：效期预警必须带出剩余天数")
	}
	if *got.DaysLeft < 29 || *got.DaysLeft > 31 {
		t.Errorf("days_left = %d, 期望约 30（date-date 应为整天数）", *got.DaysLeft)
	}
}

// TestAlertListDaysLeftNullForLowStock 低库存预警无关联效期，days_left 应为 NULL 而非 0，
// 前端据此留空；返回 0 会被误读为「今天到期」。
func TestAlertListDaysLeftNullForLowStock(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	drug := mustCreateDrug(t, db)

	alert := &model.StockAlert{
		DrugID:     drug.ID,
		LocationID: 2,
		AlertType:  "low_stock",
		Message:    "低库存",
		Status:     "open",
	}
	if err := repository.NewStockAlertRepo(db).Create(ctx, alert); err != nil {
		t.Fatalf("创建低库存预警失败: %v", err)
	}

	rows, _, err := repository.NewStockAlertRepo(db).List(ctx, "low_stock", "open", 0, 50)
	if err != nil {
		t.Fatalf("查询低库存预警失败: %v", err)
	}
	for i := range rows {
		if rows[i].ID == alert.ID {
			if rows[i].DaysLeft != nil {
				t.Errorf("无 expiry_date 的预警 days_left = %d, 期望 NULL", *rows[i].DaysLeft)
			}
			if rows[i].DrugName == "" {
				t.Error("低库存预警 drug_name 为空")
			}
			return
		}
	}
	t.Fatalf("低库存预警 %d 未出现在列表中", alert.ID)
}

// TestServiceListAlertsReturnsRowDTO 服务层签名须同步为行 DTO，
// 否则 handler 返回的仍是 model.StockAlert，字段在链路上再次丢失。
func TestServiceListAlertsReturnsRowDTO(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	inv := service.NewInventoryService(db)

	if _, _, err := inv.ListAlerts(ctx, "expiry", "open", 1, 10); err != nil {
		t.Fatalf("服务层查询效期预警失败: %v", err)
	}
	if _, _, err := inv.ListAlerts(ctx, "low_stock", "open", 1, 10); err != nil {
		t.Fatalf("服务层查询低库存预警失败: %v", err)
	}
}
