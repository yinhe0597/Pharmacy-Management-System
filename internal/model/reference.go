package model

import "time"

// 以下为 v1.3 引入的参考数据只读表（ICD-10 / 集采 / 医保 / 耗材 / 非医保），
// 对应迁移 000015-000019，无软删除、无写接口。

// DiagnosisCode ICD-10 诊断编码（迁移 000015）。
type DiagnosisCode struct {
	ID           int64     `gorm:"primaryKey" json:"id"`
	Code         string    `gorm:"size:10;not null" json:"code"`
	DiseaseName  string    `gorm:"size:200;not null" json:"disease_name"`
	ChapterCode  string    `gorm:"size:20" json:"chapter_code"`
	ChapterName  string    `gorm:"size:100" json:"chapter_name"`
	CategoryCode string    `gorm:"size:20" json:"category_code"`
	CategoryName string    `gorm:"size:100" json:"category_name"`
	PyCode       string    `gorm:"size:50" json:"py_code"`
	IsCommon     bool      `gorm:"not null;default:false" json:"is_common"`
	Status       int       `gorm:"not null;default:1" json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (DiagnosisCode) TableName() string { return "diagnosis_codes" }

// VBPDrug 国家集采药品目录（迁移 000016）。
type VBPDrug struct {
	ID             int64     `gorm:"primaryKey" json:"id"`
	GenericName    string    `gorm:"size:150;not null" json:"generic_name"`
	DosageForm     string    `gorm:"size:40;not null" json:"dosage_form"`
	DosageCategory string    `gorm:"size:30" json:"dosage_category"`
	VPBBatch       int       `gorm:"column:vbp_batch;not null" json:"vbp_batch"` // 字段名 VPBBatch 会被 GORM 误转为 vpb_batch，故显式指定列名
	PyCode         string    `gorm:"size:50" json:"py_code"`
	IsCommon       bool      `gorm:"not null;default:false" json:"is_common"`
	Status         int       `gorm:"not null;default:1" json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (VBPDrug) TableName() string { return "vbp_drug_catalog" }

// NHSADrug 国家医保药品目录（迁移 000017）。
type NHSADrug struct {
	ID             int64     `gorm:"primaryKey" json:"id"`
	DrugName       string    `gorm:"size:200;not null" json:"drug_name"`
	DosageForm     string    `gorm:"size:80" json:"dosage_form"`
	InsuranceClass string    `gorm:"size:4;not null" json:"insurance_class"`
	DrugCategory   string    `gorm:"size:100" json:"drug_category"`
	SubCategory    string    `gorm:"size:100" json:"sub_category"`
	IsEssential    bool      `gorm:"not null;default:false" json:"is_essential"`
	PyCode         string    `gorm:"size:80" json:"py_code"`
	Notes          string    `gorm:"size:500" json:"notes"`
	Status         int       `gorm:"not null;default:1" json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (NHSADrug) TableName() string { return "nhsa_drug_catalog" }

// MedicalConsumable 医用耗材参考目录（迁移 000018）。
type MedicalConsumable struct {
	ID          int64     `gorm:"primaryKey" json:"id"`
	ItemName    string    `gorm:"size:200;not null" json:"item_name"`
	SubCategory string    `gorm:"size:50" json:"sub_category"`
	Category    string    `gorm:"size:50;not null" json:"category"`
	NMPAClass   string    `gorm:"size:20" json:"nmpa_class"`
	Description string    `gorm:"size:300" json:"description"`
	PyCode      string    `gorm:"size:80" json:"py_code"`
	IsCommon    bool      `gorm:"not null;default:false" json:"is_common"`
	Status      int       `gorm:"not null;default:1" json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (MedicalConsumable) TableName() string { return "medical_consumables" }

// NonInsuranceDrug 非医保常用药品参考目录（迁移 000019）。
type NonInsuranceDrug struct {
	ID          int64     `gorm:"primaryKey" json:"id"`
	DrugName    string    `gorm:"size:200;not null" json:"drug_name"`
	DosageForm  string    `gorm:"size:60" json:"dosage_form"`
	RxOtcClass  string    `gorm:"size:20" json:"rx_otc_class"`
	Category    string    `gorm:"size:50;not null" json:"category"`
	SubCategory string    `gorm:"size:50" json:"sub_category"`
	Description string    `gorm:"size:300" json:"description"`
	PyCode      string    `gorm:"size:80" json:"py_code"`
	IsCommon    bool      `gorm:"not null;default:true" json:"is_common"`
	Status      int       `gorm:"not null;default:1" json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (NonInsuranceDrug) TableName() string { return "non_insurance_drugs" }
