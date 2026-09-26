//go:build integration

package service_test

import (
	"testing"
)

// testStaticTables 不清表的静态表（种子/引用/基础设施），附原因。
// 用例残留只可能来自 testCleanupTables 之外的业务表；新增业务表必须进清理清单。
var testStaticTables = map[string]string{
	"schema_migrations":         "迁移版本表",
	"users":                     "种子账号（handler 测试用 admin 登录）",
	"system_settings":           "种子默认配置（契约测试读取）",
	"inventory_locations":       "种子库房（用例硬编码 ID 1/2）",
	"drug_categories":           "种子分类",
	"diagnosis_codes":           "ICD-10 种子",
	"vbp_drug_catalog":          "集采种子",
	"nhsa_drug_catalog":         "医保种子",
	"medical_consumables":       "耗材种子",
	"non_insurance_drugs":       "非医保种子",
	"ingredient_interactions":   "交互规则种子",
	"class_interaction_rules":   "交互规则种子",
	"tag_interactions":          "交互规则种子",
	"patient_contraindications": "患者禁忌种子",
	"operation_logs_archive":    "只追加归档表，测试不写入",
	"spatial_ref_sys":           "PostGIS 残留（若装扩展），非业务表",
	"geography_columns":         "PostGIS 残留（若装扩展），非业务表",
	"geometry_columns":          "PostGIS 残留（若装扩展），非业务表",
}

// TestCleanupCoversAllTables 守护：库里真实存在的业务表必须被覆盖——
// 要么在 testCleanupTables（每用例前清空），要么在 testStaticTables（有理由不清）。
// 新增 model/迁移建表后若两边都没登记，本用例失败，强制补登记。
func TestCleanupCoversAllTables(t *testing.T) {
	db := setupTestDB(t)
	var tables []string
	if err := db.Raw(`SELECT tablename FROM pg_tables WHERE schemaname = 'public'`).Scan(&tables).Error; err != nil {
		t.Fatalf("查询表清单失败: %v", err)
	}
	cleaned := make(map[string]bool, len(testCleanupTables))
	for _, tb := range testCleanupTables {
		cleaned[tb] = true
	}
	var missing []string
	for _, tb := range tables {
		if cleaned[tb] {
			continue
		}
		if _, ok := testStaticTables[tb]; ok {
			continue
		}
		missing = append(missing, tb)
	}
	if len(missing) > 0 {
		t.Fatalf("以下表未登记清理/静态清单，请加入 testCleanupTables 或 testStaticTables（附原因）: %v", missing)
	}
}
