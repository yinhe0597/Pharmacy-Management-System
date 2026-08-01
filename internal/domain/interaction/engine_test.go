// Package interaction 交互引擎单元测试。
package interaction

import (
	"testing"
)

// ---- 辅助函数 ----

func ptrInt(v int) *int { return &v }
func ptrBool(v bool) *bool { return &v }

// ---- 测试用例 ----

func TestEngineCheckExplicit(t *testing.T) {
	engine := NewEngine()
	drugs := []DrugProfile{
		{ID: 1, GenericName: "布洛芬", ActiveIngredient: "布洛芬", PharmacologicalGroup: "NSAID", InteractionTags: []string{"nsaid", "anti-inflammatory"}},
		{ID: 2, GenericName: "华法林", ActiveIngredient: "华法林", PharmacologicalGroup: "anticoagulant", InteractionTags: []string{"anticoagulant"}},
		{ID: 3, GenericName: "阿莫西林", ActiveIngredient: "阿莫西林", PharmacologicalGroup: "antibiotic"},
	}

	rules := []ExplicitPairRule{
		{DrugAID: 1, DrugBID: 2, Level: 1, Mechanism: "NSAID增加出血风险", Description: "布洛芬与华法林存在禁忌"},
	}

	result := engine.CheckPrescription(drugs, nil, nil, rules, nil, nil, nil)
	if !result.HasBlocks() {
		t.Error("expected blocks for level=1 explicit interaction")
	}
	if len(result.Interactions) == 0 {
		t.Error("expected at least one interaction finding")
	}
}

func TestEngineNoInteraction(t *testing.T) {
	engine := NewEngine()
	drugs := []DrugProfile{
		{ID: 1, GenericName: "阿莫西林", ActiveIngredient: "阿莫西林"},
		{ID: 2, GenericName: "对乙酰氨基酚", ActiveIngredient: "对乙酰氨基酚"},
	}

	result := engine.CheckPrescription(drugs, nil, nil, nil, nil, nil, nil)
	if result.HasBlocks() {
		t.Error("expected no blocks for drugs without interactions")
	}
	if len(result.Interactions) != 0 {
		t.Errorf("expected 0 interactions, got %d", len(result.Interactions))
	}
}

func TestEngineIngredientInteraction(t *testing.T) {
	engine := NewEngine()
	drugs := []DrugProfile{
		{ID: 1, GenericName: "复合感冒灵", ActiveIngredient: "对乙酰氨基酚", Ingredients: []string{"对乙酰氨基酚", "伪麻黄碱"}},
		{ID: 2, GenericName: "泰诺", ActiveIngredient: "对乙酰氨基酚", Ingredients: []string{"对乙酰氨基酚"}},
	}

	// 对乙酰氨基酚与对乙酰氨基酚不触发（同成分药品累加在重复用药中检测）
	ingredientRules := []IngredientRule{
		{IngredientA: "布洛芬", IngredientB: "华法林", Level: 1, Description: "NSAID与抗凝药"},
	}

	result := engine.CheckPrescription(drugs, nil, nil, nil, ingredientRules, nil, nil)
	if result.HasBlocks() {
		t.Error("expected no blocks: ingredient rules don't match these drugs")
	}
}

func TestEngineClassInteraction(t *testing.T) {
	engine := NewEngine()
	drugs := []DrugProfile{
		{ID: 1, GenericName: "布洛芬", PharmacologicalGroup: "NSAID"},
		{ID: 2, GenericName: "华法林", PharmacologicalGroup: "anticoagulant"},
	}

	classRules := []ClassRule{
		{ClassA: "NSAID", ClassB: "anticoagulant", Level: 2, Description: "NSAID与抗凝药慎用", IsActive: true},
	}

	result := engine.CheckPrescription(drugs, nil, nil, nil, nil, classRules, nil)
	if result.HasBlocks() {
		t.Error("expected no blocks for level=2 class interaction")
	}
	if len(result.Warnings) == 0 {
		t.Error("expected warnings for level=2 class interaction")
	}
}

func TestEngineClassInteractionInactive(t *testing.T) {
	engine := NewEngine()
	drugs := []DrugProfile{
		{ID: 1, GenericName: "布洛芬", PharmacologicalGroup: "NSAID"},
		{ID: 2, GenericName: "华法林", PharmacologicalGroup: "anticoagulant"},
	}

	classRules := []ClassRule{
		{ClassA: "NSAID", ClassB: "anticoagulant", Level: 1, Description: "已禁用规则", IsActive: false},
	}

	result := engine.CheckPrescription(drugs, nil, nil, nil, nil, classRules, nil)
	if result.HasBlocks() {
		t.Error("expected no blocks: inactive rule should be ignored")
	}
}

func TestEngineTagInteraction(t *testing.T) {
	engine := NewEngine()
	drugs := []DrugProfile{
		{ID: 1, GenericName: "舍曲林", InteractionTags: []string{"ssri", "serotonergic"}},
		{ID: 2, GenericName: "曲马多", InteractionTags: []string{"opioid", "serotonergic"}},
	}

	tagRules := []TagRule{
		{TagA: "ssri", TagB: "serotonergic", Level: 1, Description: "5-HT综合征风险", IsActive: true},
	}

	result := engine.CheckPrescription(drugs, nil, nil, nil, nil, nil, tagRules)
	if !result.HasBlocks() {
		t.Error("expected blocks for level=1 tag interaction (ssri + serotonergic)")
	}
}

func TestEngineDeduplication(t *testing.T) {
	engine := NewEngine()
	drugs := []DrugProfile{
		{ID: 1, GenericName: "布洛芬", ActiveIngredient: "布洛芬", PharmacologicalGroup: "NSAID", InteractionTags: []string{"nsaid"}},
		{ID: 2, GenericName: "华法林", ActiveIngredient: "华法林", PharmacologicalGroup: "anticoagulant", InteractionTags: []string{"anticoagulant"}},
	}

	// 同时匹配显式对、成分级、分类级和标签级 — 只保留显式对
	explicitRules := []ExplicitPairRule{
		{DrugAID: 1, DrugBID: 2, Level: 1, Mechanism: "显式", Description: "布洛芬+华法林禁忌"},
	}
	ingredientRules := []IngredientRule{
		{IngredientA: "布洛芬", IngredientB: "华法林", Level: 2, Description: "成分交互"},
	}
	classRules := []ClassRule{
		{ClassA: "NSAID", ClassB: "anticoagulant", Level: 3, Description: "分类交互", IsActive: true},
	}
	tagRules := []TagRule{
		{TagA: "nsaid", TagB: "anticoagulant", Level: 2, Description: "标签交互", IsActive: true},
	}

	result := engine.CheckPrescription(drugs, nil, nil, explicitRules, ingredientRules, classRules, tagRules)

	// 去重后只应保留一条（显式优先级最高）
	if len(result.Interactions) != 1 {
		t.Errorf("expected 1 deduplicated interaction, got %d", len(result.Interactions))
	}
	if len(result.Interactions) > 0 && result.Interactions[0].Strategy != "explicit" {
		t.Errorf("expected 'explicit' strategy, got '%s'", result.Interactions[0].Strategy)
	}
}

func TestEngineOverdose(t *testing.T) {
	engine := NewEngine()
	drugs := []DrugProfile{
		{ID: 1, GenericName: "地高辛", MaxSingleDose: 500, MaxDailyDose: 1500},
	}

	items := []PrescriptionItemInfo{
		{DrugID: 1, DrugName: "地高辛", SingleDose: 1000, TotalDailyDose: 1500},
	}

	result := engine.CheckPrescription(drugs, items, nil, nil, nil, nil, nil)
	if !result.HasBlocks() {
		t.Error("expected blocks for single dose exceeded")
	}
}

func TestEngineOverdoseDailyExceeded(t *testing.T) {
	engine := NewEngine()
	drugs := []DrugProfile{
		{ID: 1, GenericName: "地高辛", MaxSingleDose: 500, MaxDailyDose: 1500},
	}

	items := []PrescriptionItemInfo{
		{DrugID: 1, DrugName: "地高辛", SingleDose: 250, TotalDailyDose: 2000},
	}

	result := engine.CheckPrescription(drugs, items, nil, nil, nil, nil, nil)
	if !result.HasBlocks() {
		t.Error("expected blocks for daily dose exceeded")
	}
}

func TestEngineNoOverdose(t *testing.T) {
	engine := NewEngine()
	drugs := []DrugProfile{
		{ID: 1, GenericName: "地高辛", MaxSingleDose: 500, MaxDailyDose: 1500},
	}

	items := []PrescriptionItemInfo{
		{DrugID: 1, DrugName: "地高辛", SingleDose: 250, TotalDailyDose: 750},
	}

	result := engine.CheckPrescription(drugs, items, nil, nil, nil, nil, nil)
	if result.HasBlocks() {
		t.Error("expected no blocks for normal dose")
	}
}

func TestEngineDuplicateDrug(t *testing.T) {
	engine := NewEngine()
	drugs := []DrugProfile{
		{ID: 1, GenericName: "对乙酰氨基酚片", ActiveIngredient: "对乙酰氨基酚"},
		{ID: 2, GenericName: "复方感冒灵", ActiveIngredient: "对乙酰氨基酚"},
	}

	items := []PrescriptionItemInfo{
		{DrugID: 1, DrugName: "对乙酰氨基酚片"},
		{DrugID: 2, DrugName: "复方感冒灵"},
	}

	result := engine.CheckPrescription(drugs, items, nil, nil, nil, nil, nil)
	if !result.HasWarnings() {
		t.Error("expected warnings for duplicate drugs (same generic name = same ingredient)")
	}
}

func TestEngineDuplicatePharmGroup(t *testing.T) {
	engine := NewEngine()
	drugs := []DrugProfile{
		{ID: 1, GenericName: "布洛芬", PharmacologicalGroup: "NSAID"},
		{ID: 2, GenericName: "双氯芬酸", PharmacologicalGroup: "NSAID"},
	}

	items := []PrescriptionItemInfo{
		{DrugID: 1, DrugName: "布洛芬"},
		{DrugID: 2, DrugName: "双氯芬酸"},
	}

	result := engine.CheckPrescription(drugs, items, nil, nil, nil, nil, nil)
	if !result.HasWarnings() {
		t.Error("expected warnings for same pharmacological group")
	}
}

func TestEngineAgeContraindication(t *testing.T) {
	engine := NewEngine()
	drugs := []DrugProfile{
		{ID: 1, GenericName: "阿司匹林", AgeMinYears: ptrInt(12)},
	}

	patient := &PatientProfile{Age: 5}

	result := engine.CheckPrescription(drugs, nil, patient, nil, nil, nil, nil)
	if !result.HasBlocks() {
		t.Error("expected blocks for age below minimum")
	}
}

func TestEngineAgeOK(t *testing.T) {
	engine := NewEngine()
	drugs := []DrugProfile{
		{ID: 1, GenericName: "阿司匹林", AgeMinYears: ptrInt(12)},
	}

	patient := &PatientProfile{Age: 30}

	result := engine.CheckPrescription(drugs, nil, patient, nil, nil, nil, nil)
	if result.HasBlocks() {
		t.Error("expected no blocks for age within allowed range")
	}
}

func TestEnginePregnancyContraindication(t *testing.T) {
	engine := NewEngine()
	drugs := []DrugProfile{
		{ID: 1, GenericName: "异维A酸", PregnancyCategory: "X"},
	}

	patient := &PatientProfile{IsPregnant: true}

	result := engine.CheckPrescription(drugs, nil, patient, nil, nil, nil, nil)
	if !result.HasBlocks() {
		t.Error("expected blocks for pregnancy category X")
	}
}

func TestEnginePregnancyNoContraindication(t *testing.T) {
	engine := NewEngine()
	drugs := []DrugProfile{
		{ID: 1, GenericName: "对乙酰氨基酚", PregnancyCategory: "B"},
	}

	patient := &PatientProfile{IsPregnant: true}

	result := engine.CheckPrescription(drugs, nil, patient, nil, nil, nil, nil)
	if result.HasBlocks() {
		t.Error("expected no blocks for pregnancy category B")
	}
}

func TestEngineAllergyCheck(t *testing.T) {
	engine := NewEngine()
	drugs := []DrugProfile{
		{ID: 1, GenericName: "青霉素", ActiveIngredient: "青霉素"},
	}

	patient := &PatientProfile{
		Allergies: []AllergyInfo{
			{DrugName: "青霉素", Reaction: "过敏性休克", Severity: 3},
		},
	}

	result := engine.CheckPrescription(drugs, nil, patient, nil, nil, nil, nil)
	if !result.HasBlocks() {
		t.Error("expected blocks for known allergy")
	}
}

func TestEngineAllergyIngredientMatch(t *testing.T) {
	engine := NewEngine()
	drugs := []DrugProfile{
		{ID: 1, GenericName: "头孢拉定", ActiveIngredient: "头孢拉定", Ingredients: []string{"头孢拉定"}},
	}

	patient := &PatientProfile{
		Allergies: []AllergyInfo{
			{DrugName: "头孢拉定", Reaction: "皮疹", Severity: 2},
		},
	}

	result := engine.CheckPrescription(drugs, nil, patient, nil, nil, nil, nil)
	if !result.HasBlocks() {
		t.Error("expected blocks for ingredient-matched allergy")
	}
}

func TestEngineLactationWarning(t *testing.T) {
	engine := NewEngine()
	drugs := []DrugProfile{
		{ID: 1, GenericName: "甲硝唑", LactationSafe: ptrBool(false)},
	}

	patient := &PatientProfile{IsLactating: true}

	result := engine.CheckPrescription(drugs, nil, patient, nil, nil, nil, nil)
	if !result.HasWarnings() {
		t.Error("expected warnings for lactation safety concern")
	}
}

func TestEngineSingleDrug(t *testing.T) {
	engine := NewEngine()
	drugs := []DrugProfile{
		{ID: 1, GenericName: "布洛芬"},
	}

	result := engine.CheckPrescription(drugs, nil, nil, nil, nil, nil, nil)
	if result.HasBlocks() {
		t.Error("expected no blocks for single drug")
	}
	if len(result.Interactions) != 0 {
		t.Errorf("expected 0 interactions for single drug, got %d", len(result.Interactions))
	}
}

func TestEngineMultiDrugPrescription(t *testing.T) {
	engine := NewEngine()
	drugs := []DrugProfile{
		{ID: 1, GenericName: "布洛芬", ActiveIngredient: "布洛芬", PharmacologicalGroup: "NSAID", InteractionTags: []string{"nsaid"}},
		{ID: 2, GenericName: "华法林", ActiveIngredient: "华法林", PharmacologicalGroup: "anticoagulant", InteractionTags: []string{"anticoagulant"}},
		{ID: 3, GenericName: "阿莫西林", ActiveIngredient: "阿莫西林", PharmacologicalGroup: "antibiotic"},
		{ID: 4, GenericName: "奥美拉唑", ActiveIngredient: "奥美拉唑", PharmacologicalGroup: "PPI"},
		{ID: 5, GenericName: "对乙酰氨基酚", ActiveIngredient: "对乙酰氨基酚"},
	}

	explicitRules := []ExplicitPairRule{
		{DrugAID: 1, DrugBID: 2, Level: 1, Description: "布洛芬+华法林禁忌"},
	}
	classRules := []ClassRule{
		{ClassA: "NSAID", ClassB: "anticoagulant", Level: 1, Description: "NSAID类与抗凝药类禁忌", IsActive: true},
	}

	result := engine.CheckPrescription(drugs, nil, nil, explicitRules, nil, classRules, nil)

	// 显式规则覆盖分类规则，保留1条
	if len(result.Interactions) != 1 {
		t.Errorf("expected 1 deduplicated interaction, got %d", len(result.Interactions))
	}
	if !result.HasBlocks() {
		t.Error("expected blocks for level=1 interaction")
	}
}

func TestEngineNoDoseLimit(t *testing.T) {
	engine := NewEngine()
	// MaxSingleDose=0 表示不检查单次极量
	drugs := []DrugProfile{
		{ID: 1, GenericName: "维生素C", MaxSingleDose: 0, MaxDailyDose: 0},
	}

	items := []PrescriptionItemInfo{
		{DrugID: 1, DrugName: "维生素C", SingleDose: 10000, TotalDailyDose: 30000},
	}

	result := engine.CheckPrescription(drugs, items, nil, nil, nil, nil, nil)
	if result.HasBlocks() {
		t.Error("expected no blocks: dose limits are zero (unset)")
	}
}

func TestEngineDuplicateNotTriggeredAlone(t *testing.T) {
	engine := NewEngine()
	drugs := []DrugProfile{
		{ID: 1, GenericName: "布洛芬"},
	}

	items := []PrescriptionItemInfo{
		{DrugID: 1, DrugName: "布洛芬"},
	}

	result := engine.CheckPrescription(drugs, items, nil, nil, nil, nil, nil)
	if result.HasWarnings() {
		t.Error("expected no duplicate warning for single drug")
	}
}
