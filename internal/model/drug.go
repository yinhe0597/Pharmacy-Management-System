package model

import (
	"time"

	"gorm.io/gorm"
)

// DrugCategory 药品分类（支持树）。
type DrugCategory struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	Code      string    `gorm:"size:20;uniqueIndex;not null" json:"code"`
	Name      string    `gorm:"size:50;not null" json:"name"`
	ParentID  int64     `gorm:"not null;default:0" json:"parent_id"`
	SortOrder int       `gorm:"not null;default:0" json:"sort_order"`
	Status    int       `gorm:"not null;default:1" json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (DrugCategory) TableName() string { return "drug_categories" }

// Drug 药品主数据。
type Drug struct {
	ID                 int64          `gorm:"primaryKey" json:"id"`
	Code               string         `gorm:"size:20;uniqueIndex;not null" json:"code"`
	GenericName        string         `gorm:"size:100;not null;index" json:"generic_name"`
	BrandName          string         `gorm:"size:100" json:"brand_name"`
	DosageForm         string         `gorm:"size:30;not null" json:"dosage_form"`
	Specification      string         `gorm:"size:100;not null" json:"specification"`
	Manufacturer       string         `gorm:"size:100;not null" json:"manufacturer"`
	ApprovalNumber     string         `gorm:"size:50" json:"approval_number"`
	Barcode            string         `gorm:"size:50" json:"barcode"`
	CategoryID         *int64         `gorm:"index" json:"category_id"`
	BaseUnit           string         `gorm:"size:20;not null" json:"base_unit"`
	SplitUnit          string         `gorm:"size:20" json:"split_unit"`
	PackSize           int            `gorm:"not null;default:1" json:"pack_size"`
	IsSplitAllowed     bool           `gorm:"not null;default:false" json:"is_split_allowed"`
	RetailPrice        int64          `gorm:"not null;default:0" json:"retail_price"`
	PurchasePrice      int64          `gorm:"not null;default:0" json:"purchase_price"`
	SplitRetailPrice   int64          `gorm:"not null;default:0" json:"split_retail_price"`
	SplitPurchasePrice int64          `gorm:"not null;default:0" json:"split_purchase_price"`
	AntibioticLevel    int            `gorm:"not null;default:0" json:"antibiotic_level"`           // 0非抗生素 1非限制 2限制 3特殊
	SpecialControlType int            `gorm:"not null;default:0;index" json:"special_control_type"` // 0无 1麻醉 2精神 3毒性 4放射性
	PsychotropicLevel  int            `gorm:"not null;default:0" json:"psychotropic_level"`
	MaxSingleDose         int64          `json:"max_single_dose"`
	MaxDailyDose          int64          `json:"max_daily_dose"`
	ExpiryWarningDays     int            `gorm:"not null;default:90" json:"expiry_warning_days"`
	ActiveIngredient      string         `gorm:"size:200" json:"active_ingredient"`
	ATCCode               string         `gorm:"size:10" json:"atc_code"`
	PharmacologicalGroup  string         `gorm:"size:100;index" json:"pharmacological_group"`
	PregnancyCategory     string         `gorm:"size:1" json:"pregnancy_category"`
	AgeMinYears           *int           `json:"age_min_years"`
	AgeMaxYears           *int           `json:"age_max_years"`
	InteractionTags       string         `gorm:"size:500" json:"interaction_tags"`
	LactationSafe         *bool          `json:"lactation_safe"`
	ContraindicationNotes string         `gorm:"type:text" json:"contraindication_notes"`
	PyCode                string         `gorm:"size:50;index" json:"py_code"`
	Status                int            `gorm:"not null;default:1" json:"status"`
	IsFrozen              bool           `gorm:"not null;default:false" json:"is_frozen"`
	Remarks               string         `gorm:"type:text" json:"remarks"`
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
	DeletedAt             gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Drug) TableName() string { return "drugs" }

// DrugInteraction 药品配伍禁忌。
type DrugInteraction struct {
	ID              int64     `gorm:"primaryKey" json:"id"`
	DrugAID         int64     `gorm:"not null;index" json:"drug_a_id"`
	DrugBID         int64     `gorm:"not null;index" json:"drug_b_id"`
	Level           int       `gorm:"not null" json:"level"` // 1禁忌 2慎用 3注意
	Description     string    `gorm:"size:500" json:"description"`
	Mechanism       string    `gorm:"size:500" json:"mechanism"`
	EvidenceLevel   string    `gorm:"size:1;default:E" json:"evidence_level"`
	SourceReference string    `gorm:"size:500" json:"source_reference"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (DrugInteraction) TableName() string { return "drug_interactions" }

// DrugIngredient 药品-成分映射。
type DrugIngredient struct {
	ID             int64     `gorm:"primaryKey" json:"id"`
	DrugID         int64     `gorm:"not null;index" json:"drug_id"`
	IngredientName string    `gorm:"size:200;not null;index" json:"ingredient_name"`
	Strength       string    `gorm:"size:50" json:"strength"`
	CreatedAt      time.Time `json:"created_at"`
}

func (DrugIngredient) TableName() string { return "drug_ingredients" }

// IngredientInteraction 成分-成分相互作用规则。
type IngredientInteraction struct {
	ID              int64     `gorm:"primaryKey" json:"id"`
	IngredientA     string    `gorm:"size:200;not null;index" json:"ingredient_a"`
	IngredientB     string    `gorm:"size:200;not null;index" json:"ingredient_b"`
	Level           int       `gorm:"not null" json:"level"` // 1禁忌 2慎用 3注意
	Mechanism       string    `gorm:"size:500" json:"mechanism"`
	EvidenceLevel   string    `gorm:"size:1;default:C" json:"evidence_level"`
	SourceReference string    `gorm:"size:500" json:"source_reference"`
	Description     string    `gorm:"size:500" json:"description"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (IngredientInteraction) TableName() string { return "ingredient_interactions" }

// ClassInteractionRule 分类-分类相互作用规则。
type ClassInteractionRule struct {
	ID              int64     `gorm:"primaryKey" json:"id"`
	ClassA          string    `gorm:"size:100;not null;index" json:"class_a"`
	ClassB          string    `gorm:"size:100;not null;index" json:"class_b"`
	Level           int       `gorm:"not null" json:"level"`
	Mechanism       string    `gorm:"size:500" json:"mechanism"`
	EvidenceLevel   string    `gorm:"size:1;default:C" json:"evidence_level"`
	SourceReference string    `gorm:"size:500" json:"source_reference"`
	Description     string    `gorm:"size:500" json:"description"`
	IsActive        bool      `gorm:"not null;default:true" json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (ClassInteractionRule) TableName() string { return "class_interaction_rules" }

// TagInteraction 交互标签-标签相互作用规则。
type TagInteraction struct {
	ID              int64     `gorm:"primaryKey" json:"id"`
	TagA            string    `gorm:"size:50;not null" json:"tag_a"`
	TagB            string    `gorm:"size:50;not null" json:"tag_b"`
	Level           int       `gorm:"not null" json:"level"`
	Mechanism       string    `gorm:"size:500" json:"mechanism"`
	EvidenceLevel   string    `gorm:"size:1;default:C" json:"evidence_level"`
	SourceReference string    `gorm:"size:500" json:"source_reference"`
	Description     string    `gorm:"size:500" json:"description"`
	IsActive        bool      `gorm:"not null;default:true" json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (TagInteraction) TableName() string { return "tag_interactions" }

// PatientContraindication 患者禁忌。
type PatientContraindication struct {
	ID                   int64     `gorm:"primaryKey" json:"id"`
	DrugID               *int64    `json:"drug_id"`
	Ingredient           string    `gorm:"size:200" json:"ingredient"`
	ContraindicationType string    `gorm:"size:50;not null;index" json:"contraindication_type"`
	ConditionValue       string    `gorm:"size:200" json:"condition_value"`
	Level                int       `gorm:"not null;default:1" json:"level"`
	Description          string    `gorm:"size:500" json:"description"`
	CreatedAt            time.Time `json:"created_at"`
}

func (PatientContraindication) TableName() string { return "patient_contraindications" }

// InteractionResult 交互检测结果快照。
type InteractionResult struct {
	ID               int64     `gorm:"primaryKey" json:"id"`
	PrescriptionID   int64     `gorm:"not null;index" json:"prescription_id"`
	Strategy         string    `gorm:"size:20;not null" json:"strategy"`
	DrugAID          *int64    `json:"drug_a_id"`
	DrugBID          *int64    `json:"drug_b_id"`
	DrugAName        string    `gorm:"size:100" json:"drug_a_name"`
	DrugBName        string    `gorm:"size:100" json:"drug_b_name"`
	Level            int       `gorm:"not null" json:"level"`
	Severity         string    `gorm:"size:20;not null;default:warning" json:"severity"`
	Mechanism        string    `gorm:"size:500" json:"mechanism"`
	EvidenceLevel    string    `gorm:"size:1" json:"evidence_level"`
	Resolved         bool      `gorm:"not null;default:false" json:"resolved"`
	ResolvedBy       *int64    `json:"resolved_by"`
	ResolvedRemarks  string    `gorm:"size:500" json:"resolved_remarks"`
	CreatedAt        time.Time `json:"created_at"`
}

func (InteractionResult) TableName() string { return "interaction_results" }
