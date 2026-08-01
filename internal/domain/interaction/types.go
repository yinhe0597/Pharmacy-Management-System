// Package interaction 定义药物相互作用检测的核心类型。
package interaction

import "yaofang/internal/domain/enum"

// DrugProfile 药品用于交互检测的完整画像（预加载，不查 DB）。
type DrugProfile struct {
	ID                   int64
	GenericName          string
	BrandName            string
	ActiveIngredient     string
	PharmacologicalGroup string
	Ingredients          []string // 从 drug_ingredients 加载
	InteractionTags      []string // 从 interaction_tags 字段解析
	AgeMinYears          *int
	AgeMaxYears          *int
	PregnancyCategory    string
	LactationSafe        *bool
	MaxSingleDose        int64
	MaxDailyDose         int64
}

// PatientProfile 患者画像（用于个体化禁忌检查）。
type PatientProfile struct {
	Age         int
	Gender      string
	IsPregnant  bool
	IsLactating bool
	Allergies   []AllergyInfo
}

// AllergyInfo 过敏信息。
type AllergyInfo struct {
	DrugName  string
	Reaction  string
	Severity  int // 1轻 2中 3重
}

// PrescriptionItemInfo 处方明细（用于极量/重复用药检查）。
type PrescriptionItemInfo struct {
	DrugID         int64
	DrugName       string
	SingleDose     int64
	TotalDailyDose int64
}

// InteractionFinding 单条交互检测结果。
type InteractionFinding struct {
	DrugAID        int64  `json:"drug_a_id"`
	DrugBID        int64  `json:"drug_b_id"`
	DrugAName      string `json:"drug_a_name"`
	DrugBName      string `json:"drug_b_name"`
	Strategy       string `json:"strategy"` // explicit / ingredient / class / tag
	Level          int    `json:"level"`    // 1禁忌 2慎用 3注意
	Mechanism      string `json:"mechanism"`
	EvidenceLevel  string `json:"evidence_level"`
	Description    string `json:"description"`
}

// PairWarning 药品对级别的提醒。
type PairWarning struct {
	DrugAID   int64  `json:"drug_a_id"`
	DrugBID   int64  `json:"drug_b_id"`
	DrugAName string `json:"drug_a_name"`
	DrugBName string `json:"drug_b_name"`
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Detail    string `json:"detail"`
}

// DrugWarning 单药品级别的提醒（如年龄、妊娠禁忌）。
type DrugWarning struct {
	DrugID   int64  `json:"drug_id"`
	DrugName string `json:"drug_name"`
	Code     int    `json:"code"`
	Message  string `json:"message"`
	Detail   string `json:"detail"`
}

// AuditResult 完整的处方审核结果。
type AuditResult struct {
	Interactions []InteractionFinding `json:"interactions,omitempty"`
	Warnings     []PairWarning        `json:"warnings,omitempty"`
	DrugWarnings []DrugWarning        `json:"drug_warnings,omitempty"`
	Blocks       []Block              `json:"blocks,omitempty"`
	Passed       bool                 `json:"passed"`
}

// Block 拦截项。
type Block struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail"`
}

// HasBlocks 是否存在拦截项。
func (r *AuditResult) HasBlocks() bool { return len(r.Blocks) > 0 }

// HasWarnings 是否存在提醒项。
func (r *AuditResult) HasWarnings() bool { return len(r.Warnings) > 0 || len(r.DrugWarnings) > 0 }

// AddBlock 添加拦截项。
func (r *AuditResult) AddBlock(code int, msg, detail string) {
	r.Blocks = append(r.Blocks, Block{Code: code, Message: msg, Detail: detail})
	r.Passed = false
}

// AddWarning 添加药品对提醒。
func (r *AuditResult) AddWarning(w PairWarning) {
	r.Warnings = append(r.Warnings, w)
}

// AddDrugWarning 添加单药品提醒。
func (r *AuditResult) AddDrugWarning(w DrugWarning) {
	r.DrugWarnings = append(r.DrugWarnings, w)
}

// AddInteraction 添加交互发现。
func (r *AuditResult) AddInteraction(f InteractionFinding) {
	r.Interactions = append(r.Interactions, f)
}

// FindingToBlock 将交互发现转为拦截项。
func (f InteractionFinding) ToBlock() Block {
	return Block{
		Code:    enum.ErrInteractionCode,
		Message: "配伍禁忌",
		Detail:  f.Description,
	}
}

// FindingToWarning 将交互发现转为提醒项。
func (f InteractionFinding) ToPairWarning() PairWarning {
	code := enum.ErrInteractionNoteCode // 未知级别默认归入"注意"
	switch f.Level {
	case enum.InteractionLevelContraindication:
		code = enum.ErrInteractionCode
	case enum.InteractionLevelCaution:
		code = enum.ErrInteractionCautionCode
	case enum.InteractionLevelNote:
		code = enum.ErrInteractionNoteCode
	}
	return PairWarning{
		DrugAID:   f.DrugAID,
		DrugBID:   f.DrugBID,
		DrugAName: f.DrugAName,
		DrugBName: f.DrugBName,
		Code:      code,
		Message:   f.Description,
		Detail:    f.Mechanism,
	}
}
