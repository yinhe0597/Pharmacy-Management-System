// Package service 药物相互作用检测服务（编排层）。
package service

import (
	"context"
	"strings"
	"sync"

	"gorm.io/gorm"

	"yaofang/internal/domain/enum"
	"yaofang/internal/domain/interaction"
	"yaofang/internal/model"
	"yaofang/internal/repository"
	"yaofang/internal/service/port"
)

// InteractionService 药物相互作用检测编排服务。
type InteractionService struct {
	db     *gorm.DB
	engine *interaction.Engine
	cache  *interactionCache
}

// interactionCache 交互规则缓存（成分/分类/标签）。
type interactionCache struct {
	mu          sync.RWMutex
	ingredients []model.IngredientInteraction
	classes     []model.ClassInteractionRule
	tags        []model.TagInteraction
	loaded      bool
}

// NewInteractionService 创建交互服务。
func NewInteractionService(db *gorm.DB) *InteractionService {
	return &InteractionService{
		db:     db,
		engine: interaction.NewEngine(),
		cache:  &interactionCache{},
	}
}

// InvalidateCache 使缓存失效（规则 CRUD 后调用）。
func (s *InteractionService) InvalidateCache() {
	s.cache.mu.Lock()
	defer s.cache.mu.Unlock()
	s.cache.loaded = false
}

// warmCache 预热/刷新缓存。原子加载：全部成功才更新缓存，避免部分失败导致不一致。
func (s *InteractionService) warmCache(ctx context.Context) error {
	s.cache.mu.Lock()
	defer s.cache.mu.Unlock()
	if s.cache.loaded {
		return nil
	}
	// 先加载到局部变量，全部成功再原子赋值
	ingredients, err := repository.NewIngredientInteractionRepo(s.db).GetAll(ctx)
	if err != nil {
		return err
	}
	classes, err := repository.NewClassInteractionRepo(s.db).GetAllActive(ctx)
	if err != nil {
		return err
	}
	tags, err := repository.NewTagInteractionRepo(s.db).GetAllActive(ctx)
	if err != nil {
		return err
	}
	s.cache.ingredients = ingredients
	s.cache.classes = classes
	s.cache.tags = tags
	s.cache.loaded = true
	return nil
}

// CheckPrescription 执行处方交互检测。
// prescription 用于从处方记录中提取患者年龄/性别/妊娠状态来构建患者画像。
// db 为可选的数据库句柄：在事务内调用时应传入 tx，使检测读到的库存/药品数据与事务一致
// 并避免持锁期间在另一连接上查询（缩短行锁持有时间）；传 nil 时回退到服务自身连接。
func (s *InteractionService) CheckPrescription(
	ctx context.Context,
	db *gorm.DB,
	items []model.PrescriptionItem,
	prescription *model.Prescription,
	patientService port.IPatientService,
) (*interaction.AuditResult, error) {
	if db == nil {
		db = s.db
	}
	// 1. 提取药品 ID
	drugIDs := make([]int64, 0, len(items))
	for _, it := range items {
		drugIDs = append(drugIDs, it.DrugID)
	}

	// 2. 批量加载药品画像
	drugs, err := repository.NewDrugRepo(db).BatchGetDrugProfiles(ctx, drugIDs)
	if err != nil {
		return nil, err
	}

	// 3. 加载成分映射
	ingredients, err := repository.NewDrugIngredientRepo(db).ListByDrugIDs(ctx, drugIDs)
	if err != nil {
		return nil, err
	}
	ingByDrug := make(map[int64][]string)
	for _, ing := range ingredients {
		ingByDrug[ing.DrugID] = append(ingByDrug[ing.DrugID], ing.IngredientName)
	}

	// 4. 构建 DrugProfile 列表
	profiles := make([]interaction.DrugProfile, 0, len(drugs))
	for _, d := range drugs {
		tags := parseTags(d.InteractionTags)
		profiles = append(profiles, interaction.DrugProfile{
			ID:                   d.ID,
			GenericName:          d.GenericName,
			BrandName:            d.BrandName,
			ActiveIngredient:     d.ActiveIngredient,
			PharmacologicalGroup: d.PharmacologicalGroup,
			Ingredients:          ingByDrug[d.ID],
			InteractionTags:      tags,
			AgeMinYears:          d.AgeMinYears,
			AgeMaxYears:          d.AgeMaxYears,
			PregnancyCategory:    d.PregnancyCategory,
			LactationSafe:        d.LactationSafe,
			MaxSingleDose:        d.MaxSingleDose,
			MaxDailyDose:         d.MaxDailyDose,
		})
	}

	// 5. 构建处方明细
	itemInfos := make([]interaction.PrescriptionItemInfo, 0, len(items))
	for _, it := range items {
		drugName := ""
		for _, d := range drugs {
			if d.ID == it.DrugID {
				drugName = d.GenericName
				break
			}
		}
		itemInfos = append(itemInfos, interaction.PrescriptionItemInfo{
			DrugID:         it.DrugID,
			DrugName:       drugName,
			SingleDose:     it.SingleDose,
			TotalDailyDose: it.TotalDailyDose,
		})
	}

	// 6. 加载显式药品对交互
	explicitInteractions, err := repository.NewInteractionRepo(db).ListByDrugIDs(ctx, drugIDs)
	if err != nil {
		return nil, err
	}
	explicitRules := make([]interaction.ExplicitPairRule, 0, len(explicitInteractions))
	for _, ei := range explicitInteractions {
		explicitRules = append(explicitRules, interaction.ExplicitPairRule{
			DrugAID:       ei.DrugAID,
			DrugBID:       ei.DrugBID,
			Level:         ei.Level,
			Mechanism:     ei.Mechanism,
			EvidenceLevel: ei.EvidenceLevel,
			Description:   ei.Description,
		})
	}

	// 7. 预热缓存并加载规则
	if err := s.warmCache(ctx); err != nil {
		return nil, err
	}
	s.cache.mu.RLock()
	ingredientRules := make([]interaction.IngredientRule, 0, len(s.cache.ingredients))
	for _, r := range s.cache.ingredients {
		ingredientRules = append(ingredientRules, interaction.IngredientRule{
			IngredientA:   r.IngredientA,
			IngredientB:   r.IngredientB,
			Level:         r.Level,
			Mechanism:     r.Mechanism,
			EvidenceLevel: r.EvidenceLevel,
			Description:   r.Description,
			IsActive:      true, // ingredient_interactions 表无 is_active 列，始终启用
		})
	}
	classRules := make([]interaction.ClassRule, 0, len(s.cache.classes))
	for _, r := range s.cache.classes {
		classRules = append(classRules, interaction.ClassRule{
			ClassA:        r.ClassA,
			ClassB:        r.ClassB,
			Level:         r.Level,
			Mechanism:     r.Mechanism,
			EvidenceLevel: r.EvidenceLevel,
			Description:   r.Description,
			IsActive:      r.IsActive,
		})
	}
	tagRules := make([]interaction.TagRule, 0, len(s.cache.tags))
	for _, r := range s.cache.tags {
		tagRules = append(tagRules, interaction.TagRule{
			TagA:          r.TagA,
			TagB:          r.TagB,
			Level:         r.Level,
			Mechanism:     r.Mechanism,
			EvidenceLevel: r.EvidenceLevel,
			Description:   r.Description,
			IsActive:      r.IsActive,
		})
	}
	s.cache.mu.RUnlock()

	// 8. 构建患者画像（优先从处方记录提取，其次从 IPatientService）
	var patientProfile *interaction.PatientProfile
	if prescription != nil {
		patientProfile = s.buildPatientProfileFromPrescription(prescription, ctx, patientService)
	} else if patientService != nil {
		// 回退：通过患者服务获取（二期实现）
		patientProfile = &interaction.PatientProfile{}
	}

	// 9. 调用引擎
	result := s.engine.CheckPrescription(profiles, itemInfos, patientProfile,
		explicitRules, ingredientRules, classRules, tagRules)

	return result, nil
}

// buildPatientProfileFromPrescription 从处方记录构建患者画像（绕过 二期 stub）。
func (s *InteractionService) buildPatientProfileFromPrescription(
	p *model.Prescription,
	ctx context.Context,
	ps port.IPatientService,
) *interaction.PatientProfile {
	profile := &interaction.PatientProfile{
		Gender:      p.PatientGender,
		IsPregnant:  p.IsPregnant,
		IsLactating: p.IsLactating,
	}
	// 从年龄字符串解析年龄数值
	if p.PatientAge != "" {
		for _, c := range p.PatientAge {
			if c >= '0' && c <= '9' {
				profile.Age = profile.Age*10 + int(c-'0')
			} else {
				break
			}
		}
	}
	// 若有患者 ID，尝试从 IPatientService 加载过敏史（二期）
	if pid := idOrZero(p.PatientID); ps != nil && pid > 0 {
		allergies, err := ps.GetAllergies(ctx, pid)
		if err == nil {
			for _, a := range allergies {
				profile.Allergies = append(profile.Allergies, interaction.AllergyInfo{
					DrugName: a.DrugName,
					Reaction: a.Reaction,
					Severity: a.Severity,
				})
			}
		}
	}
	return profile
}

// parseTags 解析逗号分隔的标签字符串。
func parseTags(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// ---- 成分交互规则管理 ----

// CreateIngredientInteraction 创建成分交互规则。
func (s *InteractionService) CreateIngredientInteraction(ctx context.Context, ii *model.IngredientInteraction) error {
	if err := repository.NewIngredientInteractionRepo(s.db).Create(ctx, ii); err != nil {
		return err
	}
	s.InvalidateCache()
	return nil
}

// ListIngredientInteractions 分页查询成分交互规则。
func (s *InteractionService) ListIngredientInteractions(ctx context.Context, offset, limit int) ([]model.IngredientInteraction, int64, error) {
	return repository.NewIngredientInteractionRepo(s.db).List(ctx, offset, limit)
}

// UpdateIngredientInteraction 更新成分交互规则。
func (s *InteractionService) UpdateIngredientInteraction(ctx context.Context, ii *model.IngredientInteraction) error {
	if err := repository.NewIngredientInteractionRepo(s.db).Update(ctx, ii); err != nil {
		return err
	}
	s.InvalidateCache()
	return nil
}

// DeleteIngredientInteraction 删除成分交互规则。
func (s *InteractionService) DeleteIngredientInteraction(ctx context.Context, id int64) error {
	if err := repository.NewIngredientInteractionRepo(s.db).Delete(ctx, id); err != nil {
		return err
	}
	s.InvalidateCache()
	return nil
}

// ---- 分类交互规则管理 ----

// CreateClassInteraction 创建分类交互规则。
func (s *InteractionService) CreateClassInteraction(ctx context.Context, ci *model.ClassInteractionRule) error {
	if err := repository.NewClassInteractionRepo(s.db).Create(ctx, ci); err != nil {
		return err
	}
	s.InvalidateCache()
	return nil
}

// ListClassInteractions 分页查询分类交互规则。
func (s *InteractionService) ListClassInteractions(ctx context.Context, offset, limit int) ([]model.ClassInteractionRule, int64, error) {
	return repository.NewClassInteractionRepo(s.db).List(ctx, offset, limit)
}

// UpdateClassInteraction 更新分类交互规则。
func (s *InteractionService) UpdateClassInteraction(ctx context.Context, ci *model.ClassInteractionRule) error {
	if err := repository.NewClassInteractionRepo(s.db).Update(ctx, ci); err != nil {
		return err
	}
	s.InvalidateCache()
	return nil
}

// DeleteClassInteraction 删除分类交互规则。
func (s *InteractionService) DeleteClassInteraction(ctx context.Context, id int64) error {
	if err := repository.NewClassInteractionRepo(s.db).Delete(ctx, id); err != nil {
		return err
	}
	s.InvalidateCache()
	return nil
}

// ---- 标签交互规则管理 ----

// CreateTagInteraction 创建标签交互规则。
func (s *InteractionService) CreateTagInteraction(ctx context.Context, ti *model.TagInteraction) error {
	if err := repository.NewTagInteractionRepo(s.db).Create(ctx, ti); err != nil {
		return err
	}
	s.InvalidateCache()
	return nil
}

// ListTagInteractions 分页查询标签交互规则。
func (s *InteractionService) ListTagInteractions(ctx context.Context, offset, limit int) ([]model.TagInteraction, int64, error) {
	return repository.NewTagInteractionRepo(s.db).List(ctx, offset, limit)
}

// UpdateTagInteraction 更新标签交互规则。
func (s *InteractionService) UpdateTagInteraction(ctx context.Context, ti *model.TagInteraction) error {
	if err := repository.NewTagInteractionRepo(s.db).Update(ctx, ti); err != nil {
		return err
	}
	s.InvalidateCache()
	return nil
}

// DeleteTagInteraction 删除标签交互规则。
func (s *InteractionService) DeleteTagInteraction(ctx context.Context, id int64) error {
	if err := repository.NewTagInteractionRepo(s.db).Delete(ctx, id); err != nil {
		return err
	}
	s.InvalidateCache()
	return nil
}

// ---- 药品成分管理 ----

// AddDrugIngredient 添加药品成分。
func (s *InteractionService) AddDrugIngredient(ctx context.Context, di *model.DrugIngredient) error {
	return repository.NewDrugIngredientRepo(s.db).Create(ctx, di)
}

// RemoveDrugIngredient 删除药品成分。
func (s *InteractionService) RemoveDrugIngredient(ctx context.Context, id int64) error {
	return repository.NewDrugIngredientRepo(s.db).Delete(ctx, id)
}

// ListDrugIngredients 查询药品成分。
func (s *InteractionService) ListDrugIngredients(ctx context.Context, drugID int64) ([]model.DrugIngredient, error) {
	return repository.NewDrugIngredientRepo(s.db).ListByDrugID(ctx, drugID)
}

// ---- 辅助方法 ----

// SaveInteractionResults 保存交互检测结果快照。
func (s *InteractionService) SaveInteractionResults(
	ctx context.Context, tx *gorm.DB, prescriptionID int64, result *interaction.AuditResult,
) error {
	if result == nil {
		return nil
	}
	var records []model.InteractionResult
	for _, f := range result.Interactions {
		severity := enum.InteractionSeverityWarning
		if f.Level == enum.InteractionLevelContraindication {
			severity = enum.InteractionSeverityBlock
		}
		records = append(records, model.InteractionResult{
			PrescriptionID: prescriptionID,
			Strategy:       f.Strategy,
			DrugAID:        &f.DrugAID,
			DrugBID:        &f.DrugBID,
			DrugAName:      f.DrugAName,
			DrugBName:      f.DrugBName,
			Level:          f.Level,
			Severity:       severity,
			Mechanism:      f.Mechanism,
			EvidenceLevel:  f.EvidenceLevel,
		})
	}
	return repository.NewInteractionResultRepo(tx).BatchCreate(ctx, records)
}
