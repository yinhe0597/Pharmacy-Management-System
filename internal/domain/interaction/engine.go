// Package interaction 提供药物相互作用检测的纯算法引擎。
package interaction

import (
	"sort"
	"strings"

	"yaofang/internal/domain/enum"
)

// Engine 药物相互作用检测引擎（无状态，纯算法）。
type Engine struct{}

// NewEngine 创建交互检测引擎。
func NewEngine() *Engine { return &Engine{} }

// CheckPrescription 执行完整的处方相互作用与禁忌检查。
func (eng *Engine) CheckPrescription(
	drugs []DrugProfile,
	items []PrescriptionItemInfo,
	patient *PatientProfile,
	explicitPairs []ExplicitPairRule,
	ingredientRules []IngredientRule,
	classRules []ClassRule,
	tagRules []TagRule,
) *AuditResult {
	result := &AuditResult{Passed: true}

	// 0. 患者级禁忌检查
	if patient != nil {
		eng.checkPatientContraindications(drugs, patient, result)
	}

	// 1-4. 药品对交互检查（四种策略）
	findings := eng.checkAllStrategies(drugs, explicitPairs, ingredientRules, classRules, tagRules)

	// 去重
	findings = eng.deduplicate(findings)

	// 分类：Level=1 → 拦截，Level≥2 → 提醒
	for _, f := range findings {
		result.AddInteraction(f)
		if f.Level == enum.InteractionLevelContraindication {
			result.AddBlock(f.ToBlock().Code, f.ToBlock().Message, f.ToBlock().Detail)
		} else {
			result.AddWarning(f.ToPairWarning())
		}
	}

	// 5. 极量检查
	eng.checkOverdose(items, drugs, result)

	// 6. 重复用药检查
	eng.checkDuplicateDrugs(items, drugs, result)

	return result
}

// checkAllStrategies 执行四种交互检测策略。
func (eng *Engine) checkAllStrategies(
	drugs []DrugProfile,
	explicitPairs []ExplicitPairRule,
	ingredientRules []IngredientRule,
	classRules []ClassRule,
	tagRules []TagRule,
) []InteractionFinding {
	var findings []InteractionFinding

	// 构建 ID→Profile 映射
	profileByID := make(map[int64]*DrugProfile, len(drugs))
	for i := range drugs {
		profileByID[drugs[i].ID] = &drugs[i]
	}

	// 策略1：显式药品对
	findings = append(findings, eng.checkExplicit(drugs, explicitPairs, profileByID)...)

	// 策略2：成分匹配
	findings = append(findings, eng.checkIngredient(drugs, ingredientRules)...)

	// 策略3：分类匹配
	findings = append(findings, eng.checkClass(drugs, classRules)...)

	// 策略4：标签匹配
	findings = append(findings, eng.checkTag(drugs, tagRules)...)

	return findings
}

// checkExplicit 显式药品对交互检查。
func (eng *Engine) checkExplicit(
	drugs []DrugProfile,
	rules []ExplicitPairRule,
	profileByID map[int64]*DrugProfile,
) []InteractionFinding {
	var findings []InteractionFinding
	drugIDs := make(map[int64]bool, len(drugs))
	for _, d := range drugs {
		drugIDs[d.ID] = true
	}
	for _, r := range rules {
		if drugIDs[r.DrugAID] && drugIDs[r.DrugBID] {
			nameA, nameB := "", ""
			if p, ok := profileByID[r.DrugAID]; ok {
				nameA = p.GenericName
			}
			if p, ok := profileByID[r.DrugBID]; ok {
				nameB = p.GenericName
			}
			findings = append(findings, InteractionFinding{
				DrugAID:       r.DrugAID,
				DrugBID:       r.DrugBID,
				DrugAName:     nameA,
				DrugBName:     nameB,
				Strategy:      enum.InteractionStrategyExplicit,
				Level:         r.Level,
				Mechanism:     r.Mechanism,
				EvidenceLevel: r.EvidenceLevel,
				Description:   r.Description,
			})
		}
	}
	return findings
}

// checkIngredient 成分级交互检查。
func (eng *Engine) checkIngredient(drugs []DrugProfile, rules []IngredientRule) []InteractionFinding {
	if len(rules) == 0 {
		return nil
	}
	// 构建 ingredient → []drugs 索引
	ingIndex := make(map[string][]DrugProfile)
	for _, d := range drugs {
		// 主成分
		if d.ActiveIngredient != "" {
			ingIndex[d.ActiveIngredient] = append(ingIndex[d.ActiveIngredient], d)
		}
		// drug_ingredients 中的成分
		for _, ing := range d.Ingredients {
			ingIndex[ing] = append(ingIndex[ing], d)
		}
	}
	var findings []InteractionFinding
	for _, r := range rules {
		drugsA := ingIndex[r.IngredientA]
		drugsB := ingIndex[r.IngredientB]
		if len(drugsA) == 0 || len(drugsB) == 0 {
			continue
		}
		for _, da := range drugsA {
			for _, db := range drugsB {
				if da.ID == db.ID {
					continue
				}
				findings = append(findings, InteractionFinding{
					DrugAID:       da.ID,
					DrugBID:       db.ID,
					DrugAName:     da.GenericName,
					DrugBName:     db.GenericName,
					Strategy:      enum.InteractionStrategyIngredient,
					Level:         r.Level,
					Mechanism:     r.Mechanism,
					EvidenceLevel: r.EvidenceLevel,
					Description:   r.Description,
				})
			}
		}
	}
	return findings
}

// checkClass 分类级交互检查。
func (eng *Engine) checkClass(drugs []DrugProfile, rules []ClassRule) []InteractionFinding {
	if len(rules) == 0 {
		return nil
	}
	classIndex := make(map[string][]DrugProfile)
	for _, d := range drugs {
		g := strings.TrimSpace(d.PharmacologicalGroup)
		if g != "" {
			classIndex[g] = append(classIndex[g], d)
		}
	}
	var findings []InteractionFinding
	for _, r := range rules {
		if !r.IsActive {
			continue
		}
		drugsA := classIndex[r.ClassA]
		drugsB := classIndex[r.ClassB]
		if len(drugsA) == 0 || len(drugsB) == 0 {
			continue
		}
		for _, da := range drugsA {
			for _, db := range drugsB {
				if da.ID == db.ID {
					continue
				}
				findings = append(findings, InteractionFinding{
					DrugAID:       da.ID,
					DrugBID:       db.ID,
					DrugAName:     da.GenericName,
					DrugBName:     db.GenericName,
					Strategy:      enum.InteractionStrategyClass,
					Level:         r.Level,
					Mechanism:     r.Mechanism,
					EvidenceLevel: r.EvidenceLevel,
					Description:   r.Description,
				})
			}
		}
	}
	return findings
}

// checkTag 标签级交互检查。
func (eng *Engine) checkTag(drugs []DrugProfile, rules []TagRule) []InteractionFinding {
	if len(rules) == 0 {
		return nil
	}
	tagIndex := make(map[string][]DrugProfile)
	for _, d := range drugs {
		for _, tag := range d.InteractionTags {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tagIndex[tag] = append(tagIndex[tag], d)
			}
		}
	}
	var findings []InteractionFinding
	for _, r := range rules {
		if !r.IsActive {
			continue
		}
		drugsA := tagIndex[r.TagA]
		drugsB := tagIndex[r.TagB]
		if len(drugsA) == 0 || len(drugsB) == 0 {
			continue
		}
		for _, da := range drugsA {
			for _, db := range drugsB {
				if da.ID == db.ID {
					continue
				}
				findings = append(findings, InteractionFinding{
					DrugAID:       da.ID,
					DrugBID:       db.ID,
					DrugAName:     da.GenericName,
					DrugBName:     db.GenericName,
					Strategy:      enum.InteractionStrategyTag,
					Level:         r.Level,
					Mechanism:     r.Mechanism,
					EvidenceLevel: r.EvidenceLevel,
					Description:   r.Description,
				})
			}
		}
	}
	return findings
}

// deduplicate 去重：高优先级策略覆盖低优先级。
// 优先级：explicit > ingredient > class > tag
func (eng *Engine) deduplicate(findings []InteractionFinding) []InteractionFinding {
	if len(findings) <= 1 {
		return findings
	}

	// 按药品对分组
	type pairKey struct{ a, b int64 }
	groups := make(map[pairKey][]InteractionFinding)

	for _, f := range findings {
		a, b := f.DrugAID, f.DrugBID
		if a > b {
			a, b = b, a
		}
		key := pairKey{a, b}
		groups[key] = append(groups[key], f)
	}

	strategyRank := map[string]int{
		enum.InteractionStrategyExplicit:   1,
		enum.InteractionStrategyIngredient: 2,
		enum.InteractionStrategyClass:      3,
		enum.InteractionStrategyTag:        4,
	}

	var result []InteractionFinding
	for _, group := range groups {
		// 找出最高优先级策略
		bestRank := 5
		for _, f := range group {
			if r, ok := strategyRank[f.Strategy]; ok && r < bestRank {
				bestRank = r
			}
		}
		// 收集最高优先级策略的发现，同时记录添加数量
		added := 0
		for _, f := range group {
			if strategyRank[f.Strategy] == bestRank {
				result = append(result, f)
				added++
			}
		}
		// 同策略内按严重程度去重（level 越小越严重），只保留最严重的
		if added > 1 {
			lastGroup := result[len(result)-added:]
			bestLevel := 4
			for _, f := range lastGroup {
				if f.Level < bestLevel {
					bestLevel = f.Level
				}
			}
			// 过滤出最严重级别的
			filtered := result[:len(result)-added]
			for _, f := range lastGroup {
				if f.Level == bestLevel {
					filtered = append(filtered, f)
				}
			}
			result = filtered
		}
	}
	return result
}

// checkPatientContraindications 患者级禁忌检查。
func (eng *Engine) checkPatientContraindications(drugs []DrugProfile, patient *PatientProfile, result *AuditResult) {
	for _, d := range drugs {
		// 年龄检查
		if patient.Age > 0 {
			if d.AgeMinYears != nil && patient.Age < *d.AgeMinYears {
				result.AddBlock(enum.ErrAgeContraindicationCode, "年龄禁忌",
					d.GenericName+"：患者年龄低于该药品最低适用年龄（"+itoa(*d.AgeMinYears)+"岁）")
			}
			if d.AgeMaxYears != nil && *d.AgeMaxYears > 0 && patient.Age > *d.AgeMaxYears {
				result.AddBlock(enum.ErrAgeContraindicationCode, "年龄禁忌",
					d.GenericName+"：患者年龄超过该药品最高适用年龄（"+itoa(*d.AgeMaxYears)+"岁）")
			}
		}

		// 妊娠检查
		if patient.IsPregnant {
			cat := strings.ToUpper(d.PregnancyCategory)
			if cat == "X" || cat == "D" {
				result.AddBlock(enum.ErrPregnancyContraindicationCode, "妊娠禁忌",
					d.GenericName+"：该药品妊娠分级为"+cat+"级，孕期禁用")
			}
		}

		// 哺乳检查
		if patient.IsLactating && d.LactationSafe != nil && !*d.LactationSafe {
			result.AddDrugWarning(DrugWarning{
				DrugID:   d.ID,
				DrugName: d.GenericName,
				Code:     enum.ErrLactationWarningCode,
				Message:  "哺乳期慎用",
				Detail:   d.GenericName + "：该药品在哺乳期安全性未知或不建议使用",
			})
		}

		// 过敏检查
		for _, allergy := range patient.Allergies {
			if matchesAllergy(d, allergy) {
				result.AddBlock(enum.ErrAllergyContraindicationCode, "过敏史禁忌",
					d.GenericName+"：患者有"+allergy.DrugName+"过敏史（"+allergy.Reaction+"）")
				break
			}
		}
	}
}

// matchesAllergy 检查药品是否匹配过敏信息。
func matchesAllergy(d DrugProfile, allergy AllergyInfo) bool {
	// 通用名匹配
	if strings.EqualFold(d.GenericName, allergy.DrugName) {
		return true
	}
	// 成分匹配
	if strings.EqualFold(d.ActiveIngredient, allergy.DrugName) {
		return true
	}
	for _, ing := range d.Ingredients {
		if strings.EqualFold(ing, allergy.DrugName) {
			return true
		}
	}
	return false
}

// checkOverdose 极量检查。
func (eng *Engine) checkOverdose(items []PrescriptionItemInfo, drugs []DrugProfile, result *AuditResult) {
	profileByID := make(map[int64]*DrugProfile, len(drugs))
	for i := range drugs {
		profileByID[drugs[i].ID] = &drugs[i]
	}
	for _, item := range items {
		d, ok := profileByID[item.DrugID]
		if !ok {
			continue
		}
		if d.MaxSingleDose > 0 && item.SingleDose > 0 && item.SingleDose > d.MaxSingleDose {
			result.AddBlock(enum.ErrDoseExceededCode, "极量超限",
				d.GenericName+"：单次剂量超过最大单次剂量")
		}
		if d.MaxDailyDose > 0 && item.TotalDailyDose > 0 && item.TotalDailyDose > d.MaxDailyDose {
			result.AddBlock(enum.ErrDoseExceededCode, "极量超限",
				d.GenericName+"：日总剂量超过最大日剂量")
		}
	}
}

// checkDuplicateDrugs 重复用药检查。
func (eng *Engine) checkDuplicateDrugs(items []PrescriptionItemInfo, drugs []DrugProfile, result *AuditResult) {
	profileByID := make(map[int64]*DrugProfile, len(drugs))
	for i := range drugs {
		profileByID[drugs[i].ID] = &drugs[i]
	}

	// 按通用名分组
	byName := make(map[string][]PrescriptionItemInfo)
	for _, item := range items {
		d, ok := profileByID[item.DrugID]
		if !ok {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(d.GenericName))
		byName[name] = append(byName[name], item)
	}

	for name, group := range byName {
		if len(group) <= 1 {
			continue
		}
		var names []string
		for _, it := range group {
			names = append(names, it.DrugName)
		}
		result.AddWarning(PairWarning{
			Code:    enum.ErrDuplicateDrugCode,
			Message: "重复用药提醒",
			Detail:  "处方中存在同通用名药品：" + name + "（" + strings.Join(names, "、") + "）",
		})
	}

	// 按活性成分分组（检测不同通用名但含相同成分的复方制剂）
	byIngredient := make(map[string][]PrescriptionItemInfo)
	for _, item := range items {
		d, ok := profileByID[item.DrugID]
		if !ok || d.ActiveIngredient == "" {
			continue
		}
		ing := strings.ToLower(strings.TrimSpace(d.ActiveIngredient))
		byIngredient[ing] = append(byIngredient[ing], item)
	}

	for ing, group := range byIngredient {
		if len(group) <= 1 {
			continue
		}
		var names []string
		for _, it := range group {
			names = append(names, it.DrugName)
		}
		result.AddWarning(PairWarning{
			Code:    enum.ErrDuplicateDrugCode,
			Message: "重复用药提醒（同成分）",
			Detail:  "处方中多种药品含相同成分 " + ing + "：" + strings.Join(names, "、"),
		})
	}

	// 按药理分组检查
	byPharmGroup := make(map[string][]PrescriptionItemInfo)
	for _, item := range items {
		d, ok := profileByID[item.DrugID]
		if !ok || d.PharmacologicalGroup == "" {
			continue
		}
		group := strings.ToLower(strings.TrimSpace(d.PharmacologicalGroup))
		byPharmGroup[group] = append(byPharmGroup[group], item)
	}

	for group, members := range byPharmGroup {
		if len(members) <= 1 {
			continue
		}
		// 检查是否已经通过通用名检测到（避免重复提醒）
		var names []string
		for _, it := range members {
			names = append(names, it.DrugName)
		}
		result.AddWarning(PairWarning{
			Code:    enum.ErrDuplicateDrugCode,
			Message: "同药理分类重复用药",
			Detail:  "处方中" + group + "类药物存在多种：" + strings.Join(names, "、"),
		})
	}
}

// itoa 简单的 int → string 转换（避免导入 strconv）。
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	if neg {
		s = "-" + s
	}
	return s
}

// sortFindings 按严重程度排序（level 小 → 大）。
func sortFindings(findings []InteractionFinding) {
	sort.Slice(findings, func(i, j int) bool {
		return findings[i].Level < findings[j].Level
	})
}
