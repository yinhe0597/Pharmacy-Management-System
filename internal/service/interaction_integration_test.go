//go:build integration

// 交互引擎集成测试：验证多策略匹配与患者禁忌在处方审核中的端到端行为。
package service_test

import (
	"context"
	"testing"

	"gorm.io/gorm"

	"yaofang/internal/model"
	"yaofang/internal/pkg/errs"
	"yaofang/internal/service"
)

// setupInteractionTest 构建带交互引擎的 PrescriptionService 和基础测试药品。
func setupInteractionTest(t *testing.T) (*gorm.DB, *service.PrescriptionService, *model.Drug, *model.Drug) {
	t.Helper()
	db := setupTestDB(t)
	inv := service.NewInventoryService(db)
	interSvc := service.NewInteractionService(db)
	presc := service.NewPrescriptionService(db, inv, service.NewSpecialDrugService(db), interSvc, nil, service.NewClinicalService(db))

	drugSvc := service.NewDrugService(db)
	// 药品 A：NSAID（布洛芬），用于分类/标签交互
	drugA := &model.Drug{
		Code: "INT-A", GenericName: "布洛芬", BrandName: "IT", DosageForm: "片剂",
		Specification: "0.2g", Manufacturer: "测试药厂", ApprovalNumber: "X1",
		BaseUnit: "盒", PackSize: 24, IsSplitAllowed: true,
		RetailPrice: 1200, PurchasePrice: 900,
		ActiveIngredient: "布洛芬", PharmacologicalGroup: "NSAID",
		InteractionTags: "nsaid", Status: 1,
	}
	if err := drugSvc.Create(context.Background(), drugA); err != nil {
		t.Fatalf("创建药品A失败: %v", err)
	}

	// 药品 B：抗凝药（华法林），用于分类/标签交互
	drugB := &model.Drug{
		Code: "INT-B", GenericName: "华法林", BrandName: "IT", DosageForm: "片剂",
		Specification: "2.5mg", Manufacturer: "测试药厂", ApprovalNumber: "X2",
		BaseUnit: "盒", PackSize: 30, IsSplitAllowed: true,
		RetailPrice: 3000, PurchasePrice: 2400,
		ActiveIngredient: "华法林", PharmacologicalGroup: "anticoagulant",
		InteractionTags: "anticoagulant", Status: 1,
	}
	if err := drugSvc.Create(context.Background(), drugB); err != nil {
		t.Fatalf("创建药品B失败: %v", err)
	}

	return db, presc, drugA, drugB
}

// TestReviewBlockedByExplicitInteraction 显式药品对 level=1 禁忌 → 审核被 3003 拦截。
func TestReviewBlockedByExplicitInteraction(t *testing.T) {
	db, presc, drugA, drugB := setupInteractionTest(t)
	ctx := context.Background()

	// 创建显式配伍禁忌（level=1）
	if err := db.Create(&model.DrugInteraction{
		DrugAID: drugA.ID, DrugBID: drugB.ID, Level: 1,
		Description: "布洛芬+华法林禁忌",
	}).Error; err != nil {
		t.Fatalf("创建交互规则失败: %v", err)
	}

	// 创建含两药的处方
	p, err := presc.Create(ctx, service.PrescriptionInput{
		PatientName: "张三",
		PatientAge:  "35岁",
		Items: []service.PrescriptionItemInput{
			{DrugID: drugA.ID, Quantity: 12, IsSplit: true, SingleDose: 2, TotalDailyDose: 4, Days: 3, Frequency: "bid"},
			{DrugID: drugB.ID, Quantity: 30, IsSplit: true, SingleDose: 5, TotalDailyDose: 10, Days: 3, Frequency: "bid"},
		},
	})
	if err != nil {
		t.Fatalf("创建处方失败: %v", err)
	}

	// 提交 → 审核被拦截
	if err := presc.Submit(ctx, p.ID, 1, "操作员"); err != nil {
		t.Fatalf("提交失败: %v", err)
	}
	_, err = presc.Review(ctx, p.ID, service.AuditInput{Action: "pass"}, 2, "药师B")
	if err == nil {
		t.Fatal("期望审核被配伍禁忌拦截，但审核通过了")
	}
	if !errs.Is(err, errs.ErrInteraction) {
		t.Fatalf("期望 ErrInteraction(3003)，实际: %v", err)
	}
}

// TestReviewWarningForCautionInteraction level=2 慎用 → 审核通过但有 warnings。
func TestReviewWarningForCautionInteraction(t *testing.T) {
	db, presc, drugA, drugB := setupInteractionTest(t)
	ctx := context.Background()

	// 创建显式配伍慎用（level=2）
	if err := db.Create(&model.DrugInteraction{
		DrugAID: drugA.ID, DrugBID: drugB.ID, Level: 2,
		Description: "布洛芬+华法林慎用",
	}).Error; err != nil {
		t.Fatalf("创建交互规则失败: %v", err)
	}

	p, err := presc.Create(ctx, service.PrescriptionInput{
		PatientName: "李四",
		PatientAge:  "50岁",
		Items: []service.PrescriptionItemInput{
			{DrugID: drugA.ID, Quantity: 12, IsSplit: true, SingleDose: 2, TotalDailyDose: 4, Days: 3, Frequency: "bid"},
			{DrugID: drugB.ID, Quantity: 30, IsSplit: true, SingleDose: 5, TotalDailyDose: 10, Days: 3, Frequency: "bid"},
		},
	})
	if err != nil {
		t.Fatalf("创建处方失败: %v", err)
	}
	if err := presc.Submit(ctx, p.ID, 1, "操作员"); err != nil {
		t.Fatalf("提交失败: %v", err)
	}

	result, err := presc.Review(ctx, p.ID, service.AuditInput{Action: "pass"}, 2, "药师B")
	if err != nil {
		t.Fatalf("审核失败（level=2 慎用应通过）: %v", err)
	}
	if !result.Passed {
		t.Error("期望 passed=true（慎用仅警告不拦截）")
	}
	if len(result.Warnings) == 0 {
		t.Error("期望有 warnings（慎用级别提醒）")
	}
}

// TestReviewPassesNoInteraction 无交互药品 → 审核通过无 warnings。
func TestReviewPassesNoInteraction(t *testing.T) {
	db, presc, drugA, _ := setupInteractionTest(t)
	ctx := context.Background()

	// 创建另一个无交互的药品
	drugSvc := service.NewDrugService(db)
	drugC := &model.Drug{
		Code: "INT-C", GenericName: "维生素C", BrandName: "IT", DosageForm: "片剂",
		Specification: "100mg", Manufacturer: "测试药厂", ApprovalNumber: "X3",
		BaseUnit: "盒", PackSize: 60, IsSplitAllowed: true,
		RetailPrice: 1800, PurchasePrice: 1200, Status: 1,
	}
	if err := drugSvc.Create(ctx, drugC); err != nil {
		t.Fatalf("创建药品C失败: %v", err)
	}

	p, err := presc.Create(ctx, service.PrescriptionInput{
		PatientName: "王五",
		PatientAge:  "28岁",
		Items: []service.PrescriptionItemInput{
			{DrugID: drugA.ID, Quantity: 12, IsSplit: true, SingleDose: 2, TotalDailyDose: 4, Days: 3, Frequency: "bid"},
			{DrugID: drugC.ID, Quantity: 30, IsSplit: true, SingleDose: 1, TotalDailyDose: 2, Days: 15, Frequency: "qd"},
		},
	})
	if err != nil {
		t.Fatalf("创建处方失败: %v", err)
	}
	if err := presc.Submit(ctx, p.ID, 1, "操作员"); err != nil {
		t.Fatalf("提交失败: %v", err)
	}

	result, err := presc.Review(ctx, p.ID, service.AuditInput{Action: "pass"}, 2, "药师B")
	if err != nil {
		t.Fatalf("审核失败: %v", err)
	}
	if !result.Passed {
		t.Error("期望 passed=true")
	}
	if len(result.Warnings) > 0 {
		t.Logf("warnings: %+v", result.Warnings)
	}
}

// TestReviewBlockedByClassInteraction 药品 PharmacologicalGroup=NSAID+anticoagulant → 种子规则触发分类级禁忌。
func TestReviewBlockedByClassInteraction(t *testing.T) {
	_, presc, drugA, drugB := setupInteractionTest(t)
	ctx := context.Background()

	// 种子数据已有 class_interaction_rules: NSAID+anticoagulant level=1
	// 药品 A(NSAID) + B(anticoagulant) 应自动触发分类级交互

	p, err := presc.Create(ctx, service.PrescriptionInput{
		PatientName: "赵六",
		PatientAge:  "45岁",
		Items: []service.PrescriptionItemInput{
			{DrugID: drugA.ID, Quantity: 12, IsSplit: true, SingleDose: 2, TotalDailyDose: 4, Days: 3, Frequency: "bid"},
			{DrugID: drugB.ID, Quantity: 30, IsSplit: true, SingleDose: 5, TotalDailyDose: 10, Days: 3, Frequency: "bid"},
		},
	})
	if err != nil {
		t.Fatalf("创建处方失败: %v", err)
	}
	if err := presc.Submit(ctx, p.ID, 1, "操作员"); err != nil {
		t.Fatalf("提交失败: %v", err)
	}

	_, err = presc.Review(ctx, p.ID, service.AuditInput{Action: "pass"}, 2, "药师B")
	if err == nil {
		t.Fatal("期望审核被分类级禁忌拦截（NSAID+anticoagulant），但审核通过了")
	}
	if !errs.Is(err, errs.ErrInteraction) {
		t.Fatalf("期望 ErrInteraction(3003)，实际: %v", err)
	}
}

// TestReviewBlockedByAgeContraindication 年龄禁忌检查。
func TestReviewBlockedByAgeContraindication(t *testing.T) {
	db, presc, drugA, _ := setupInteractionTest(t)
	ctx := context.Background()

	// 给药品 A 设置最低适用年龄 12 岁
	db.Model(&model.Drug{}).Where("id = ?", drugA.ID).Update("age_min_years", 12)

	// 创建处方，患者年龄 5 岁（低于最低适用年龄）
	p, err := presc.Create(ctx, service.PrescriptionInput{
		PatientName: "小患者",
		PatientAge:  "5岁",
		Items: []service.PrescriptionItemInput{
			{DrugID: drugA.ID, Quantity: 6, IsSplit: true, SingleDose: 1, TotalDailyDose: 2, Days: 3, Frequency: "bid"},
		},
	})
	if err != nil {
		t.Fatalf("创建处方失败: %v", err)
	}
	if err := presc.Submit(ctx, p.ID, 1, "操作员"); err != nil {
		t.Fatalf("提交失败: %v", err)
	}

	_, err = presc.Review(ctx, p.ID, service.AuditInput{Action: "pass"}, 2, "药师B")
	if err == nil {
		t.Fatal("期望审核被年龄禁忌拦截（5岁 < 12岁最小适用年龄），但审核通过了")
	}
	if !errs.Is(err, errs.ErrInteraction) {
		t.Fatalf("期望被拦截(配伍禁忌汇总)，实际: %v", err)
	}
}

// TestReviewBlockedByPregnancyContraindication 妊娠禁忌检查。
func TestReviewBlockedByPregnancyContraindication(t *testing.T) {
	db, presc, drugA, _ := setupInteractionTest(t)
	ctx := context.Background()

	// 给药品 A 设置妊娠分级 X（绝对禁忌）
	db.Model(&model.Drug{}).Where("id = ?", drugA.ID).Update("pregnancy_category", "X")

	// 创建处方，标记患者妊娠
	p, err := presc.Create(ctx, service.PrescriptionInput{
		PatientName: "孕妇",
		PatientAge:  "28岁",
		IsPregnant:  true,
		Items: []service.PrescriptionItemInput{
			{DrugID: drugA.ID, Quantity: 6, IsSplit: true, SingleDose: 1, TotalDailyDose: 2, Days: 3, Frequency: "bid"},
		},
	})
	if err != nil {
		t.Fatalf("创建处方失败: %v", err)
	}
	if err := presc.Submit(ctx, p.ID, 1, "操作员"); err != nil {
		t.Fatalf("提交失败: %v", err)
	}

	_, err = presc.Review(ctx, p.ID, service.AuditInput{Action: "pass"}, 2, "药师B")
	if err == nil {
		t.Fatal("期望审核被妊娠禁忌拦截（PregnancyCategory=X + IsPregnant=true），但审核通过了")
	}
	if !errs.Is(err, errs.ErrInteraction) {
		t.Fatalf("期望被拦截(配伍禁忌汇总)，实际: %v", err)
	}
}
