//go:build integration

// 麻精处方（处方类型 UI 配套的后端收口）回归测试。
package service_test

import (
	"context"
	"testing"

	"gorm.io/gorm"

	"yaofang/internal/model"
	"yaofang/internal/pkg/errs"
	"yaofang/internal/repository"
	"yaofang/internal/service"
)

func specialDrug(t *testing.T, db *gorm.DB, code, name string, control, psychoLevel int) *model.Drug {
	t.Helper()
	d := &model.Drug{
		Code: code, GenericName: name, BrandName: code, DosageForm: "片剂",
		Specification: "10mg", Manufacturer: "麻精测试药厂", ApprovalNumber: "国药准字H55555555",
		BaseUnit: "盒", SplitUnit: "片", PackSize: 10, IsSplitAllowed: true,
		RetailPrice: 1000, PurchasePrice: 800, Status: 1,
		SpecialControlType: control, PsychotropicLevel: psychoLevel,
	}
	if err := repository.NewDrugRepo(db).Create(context.Background(), d); err != nil {
		t.Fatalf("创建麻精药品失败: %v", err)
	}
	return d
}

// TestSpecialPrescriptionRejectsZeroDays 「不填天数」不得成为绕过麻精限量的通路。
// 回归：限量判据是 it.Days > limit，days 留空（0）时 0 > limit 恒假直接放行。
// UI 一旦允许显式选择处方类型，这条通路就会被主动利用。
func TestSpecialPrescriptionRejectsZeroDays(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	inv := service.NewInventoryService(db)
	presc := service.NewPrescriptionService(db, inv, service.NewSpecialDrugService(db), nil, nil, service.NewClinicalService(db))
	narc := specialDrug(t, db, "NARC001", "麻醉测试药", 1, 0)

	// 显式指定为麻醉处方但不填天数 → 必须被拒
	_, err := presc.Create(ctx, service.PrescriptionInput{
		PatientName: "麻精患者", PatientCardNo: "CARD-001", Diagnosis: "术后镇痛",
		PrescriptionType: 1,
		Items:            []service.PrescriptionItemInput{{DrugID: narc.ID, Quantity: 1, Days: 0}},
	})
	if !errs.Is(err, errs.ErrSpecialDrugLimit) {
		t.Fatalf("麻醉处方 days=0 应被拒绝（ErrSpecialDrugLimit），got %v", err)
	}

	// 天数在限内 → 通过
	p, err := presc.Create(ctx, service.PrescriptionInput{
		PatientName: "麻精患者", PatientCardNo: "CARD-001", Diagnosis: "术后镇痛",
		PrescriptionType: 1,
		Items:            []service.PrescriptionItemInput{{DrugID: narc.ID, Quantity: 1, Days: 3}},
	})
	if err != nil {
		t.Fatalf("麻醉处方 3 日（限内）应通过: %v", err)
	}
	if p.PrescriptionType != 1 {
		t.Fatalf("处方类型应被记录为麻醉(1)，got %d", p.PrescriptionType)
	}
}

// TestSpecialPrescriptionRejectsOverLimit 超限量天数必须被拒绝。
func TestSpecialPrescriptionRejectsOverLimit(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	inv := service.NewInventoryService(db)
	presc := service.NewPrescriptionService(db, inv, service.NewSpecialDrugService(db), nil, nil, service.NewClinicalService(db))
	narc := specialDrug(t, db, "NARC002", "麻醉超量药", 1, 0)

	// 麻醉限 3 日
	_, err := presc.Create(ctx, service.PrescriptionInput{
		PatientName: "麻精患者", PatientCardNo: "CARD-002", Diagnosis: "术后镇痛",
		PrescriptionType: 1,
		Items:            []service.PrescriptionItemInput{{DrugID: narc.ID, Quantity: 1, Days: 4}},
	})
	if !errs.Is(err, errs.ErrSpecialDrugLimit) {
		t.Fatalf("麻醉处方 4 日（限 3）应被拒绝，got %v", err)
	}
}

// TestDeriveSpecialTypePicksStrictest 混合开方时按「最严」归类。
// 回归：SpecialControlNarcotic=1 < SpecialControlPsycho=2，原先用 max 取编号最大，
// 「麻醉 + 二类精神」会被判为二类精神（限 7 日），麻醉的 3 日限量被跳过。
func TestDeriveSpecialTypePicksStrictest(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	inv := service.NewInventoryService(db)
	presc := service.NewPrescriptionService(db, inv, service.NewSpecialDrugService(db), nil, nil, service.NewClinicalService(db))
	narc := specialDrug(t, db, "NARC003", "麻醉混合药", 1, 0)
	psycho2 := specialDrug(t, db, "PSY2001", "二类精神混合药", 2, 2)

	// 不显式指定类型（走自动推导），7 日：只有按「麻醉」归类才会被拒
	_, err := presc.Create(ctx, service.PrescriptionInput{
		PatientName: "混合麻精患者", PatientCardNo: "CARD-003", Diagnosis: "混合管制",
		Items: []service.PrescriptionItemInput{
			{DrugID: narc.ID, Quantity: 1, Days: 7},
			{DrugID: psycho2.ID, Quantity: 1, Days: 7},
		},
	})
	if !errs.Is(err, errs.ErrSpecialDrugLimit) {
		t.Fatalf("含麻醉药品的处方 7 日应按麻醉限(3)拒绝，got %v（说明被宽松归类为二类精神）", err)
	}

	// 3 日应通过，且类型为麻醉
	p, err := presc.Create(ctx, service.PrescriptionInput{
		PatientName: "混合麻精患者", PatientCardNo: "CARD-003", Diagnosis: "混合管制",
		Items: []service.PrescriptionItemInput{
			{DrugID: narc.ID, Quantity: 1, Days: 3},
			{DrugID: psycho2.ID, Quantity: 1, Days: 3},
		},
	})
	if err != nil {
		t.Fatalf("含麻醉药品的处方 3 日应通过: %v", err)
	}
	if p.PrescriptionType != 1 {
		t.Fatalf("混合开方应按最严归类为麻醉(1)，got %d", p.PrescriptionType)
	}
}

// TestPrescriptionTypeOutOfRangeRejected 非法处方类型必须被拒绝。
// 回归：未知类型会落入 CheckPrescriptionLimit 的 default 分支跳过限量校验。
func TestPrescriptionTypeOutOfRangeRejected(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	inv := service.NewInventoryService(db)
	presc := service.NewPrescriptionService(db, inv, service.NewSpecialDrugService(db), nil, nil, service.NewClinicalService(db))
	drug := mustCreateDrug(t, db)

	_, err := presc.Create(ctx, service.PrescriptionInput{
		PatientName: "越界类型患者", PatientCardNo: "CARD-004", Diagnosis: "测试",
		PrescriptionType: 99,
		Items:            []service.PrescriptionItemInput{{DrugID: drug.ID, Quantity: 1, Days: 999}},
	})
	if !errs.Is(err, errs.ErrBadRequest) {
		t.Fatalf("prescription_type=99 应被拒绝（ErrBadRequest），got %v", err)
	}
}

// TestSpecialPrescriptionRequiresCardAndDiagnosis 麻精处方必须填卡号与诊断。
func TestSpecialPrescriptionRequiresCardAndDiagnosis(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	inv := service.NewInventoryService(db)
	presc := service.NewPrescriptionService(db, inv, service.NewSpecialDrugService(db), nil, nil, service.NewClinicalService(db))
	narc := specialDrug(t, db, "NARC004", "麻醉必填药", 1, 0)

	for _, c := range []struct {
		name       string
		card, diag string
	}{
		{"缺卡号", "", "诊断"},
		{"缺诊断", "CARD-005", ""},
		{"两者皆缺", "", ""},
	} {
		_, err := presc.Create(ctx, service.PrescriptionInput{
			PatientName: "麻精患者", PatientCardNo: c.card, Diagnosis: c.diag,
			PrescriptionType: 1,
			Items:            []service.PrescriptionItemInput{{DrugID: narc.ID, Quantity: 1, Days: 1}},
		})
		if !errs.Is(err, errs.ErrSpecialDrugRequired) {
			t.Fatalf("%s 应返回 ErrSpecialDrugRequired，got %v", c.name, err)
		}
	}
}
