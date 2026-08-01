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
	MaxSingleDose      int64          `json:"max_single_dose"`
	MaxDailyDose       int64          `json:"max_daily_dose"`
	ExpiryWarningDays  int            `gorm:"not null;default:90" json:"expiry_warning_days"`
	PyCode             string         `gorm:"size:50;index" json:"py_code"`
	Status             int            `gorm:"not null;default:1" json:"status"`
	IsFrozen           bool           `gorm:"not null;default:false" json:"is_frozen"`
	Remarks            string         `gorm:"type:text" json:"remarks"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Drug) TableName() string { return "drugs" }

// DrugInteraction 药品配伍禁忌。
type DrugInteraction struct {
	ID          int64     `gorm:"primaryKey" json:"id"`
	DrugAID     int64     `gorm:"not null;index" json:"drug_a_id"`
	DrugBID     int64     `gorm:"not null;index" json:"drug_b_id"`
	Level       int       `gorm:"not null" json:"level"` // 1禁忌 2慎用 3注意
	Description string    `gorm:"size:500" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

func (DrugInteraction) TableName() string { return "drug_interactions" }
