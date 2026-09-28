// Package model 二期就诊模块数据模型（docs/20 S1）：就诊/病历/合并结算。
package model

import (
	"time"

	"gorm.io/gorm"
)

// Visit 就诊登记（挂号/分诊）。
type Visit struct {
	ID           int64          `gorm:"primaryKey" json:"id"`
	VisitNo      string         `gorm:"size:30;uniqueIndex;not null" json:"visit_no"`
	PatientID    int64          `gorm:"not null;index" json:"patient_id"`
	PatientName  string         `gorm:"size:50;not null" json:"patient_name"` // 姓名快照
	Department   string         `gorm:"size:50" json:"department"`
	DoctorID     *int64         `gorm:"index" json:"doctor_id"` // 接诊医生（分诊后指定）
	DoctorName   string         `gorm:"size:50" json:"doctor_name"`
	VisitType    string         `gorm:"size:20;not null;default:outpatient" json:"visit_type"` // outpatient/inpatient/refill
	Status       string         `gorm:"size:20;not null;default:waiting;index" json:"status"`  // waiting/visiting/finished/cancelled
	RegisteredBy string         `gorm:"size:50" json:"registered_by"`
	RegisteredAt time.Time      `json:"registered_at"`
	VisitedAt    *time.Time     `json:"visited_at"`
	FinishedAt   *time.Time     `json:"finished_at"`
	Remarks      string         `gorm:"size:500" json:"remarks"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Visit) TableName() string { return "visits" }

// MedicalRecord 就诊病历（一就诊一病历，可更新）。
type MedicalRecord struct {
	ID                int64                    `gorm:"primaryKey" json:"id"`
	VisitID           int64                    `gorm:"not null;uniqueIndex" json:"visit_id"`
	PatientID         int64                    `gorm:"not null;index" json:"patient_id"`
	ChiefComplaint    string                   `gorm:"type:text" json:"chief_complaint"` // 主诉
	PresentIllness    string                   `gorm:"type:text" json:"present_illness"` // 现病史
	PastHistory       string                   `gorm:"type:text" json:"past_history"`    // 既往史
	PhysicalExam      string                   `gorm:"type:text" json:"physical_exam"`   // 体格检查
	Temperature       *float64                 `json:"temperature"`                      // 体温 ℃
	SystolicPressure  *int                     `json:"systolic_pressure"`                // 收缩压 mmHg
	DiastolicPressure *int                     `json:"diastolic_pressure"`               // 舒张压 mmHg
	Pulse             *int                     `json:"pulse"`                            // 脉搏 次/分
	Diagnosis         string                   `gorm:"type:text" json:"diagnosis"`       // 诊断描述
	DiagnosisCode     string                   `gorm:"size:10" json:"diagnosis_code"`    // 主诊断 ICD-10
	CreatedBy         string                   `gorm:"size:50" json:"created_by"`
	CreatedAt         time.Time                `json:"created_at"`
	UpdatedAt         time.Time                `json:"updated_at"`
	DeletedAt         gorm.DeletedAt           `gorm:"index" json:"-"`
	Diagnoses         []MedicalRecordDiagnosis `gorm:"foreignKey:MedicalRecordID" json:"diagnoses,omitempty"`
}

func (MedicalRecord) TableName() string { return "medical_records" }

// MedicalRecordDiagnosis 病历结构化诊断（关联 ICD-10，可多诊断）。
type MedicalRecordDiagnosis struct {
	ID              int64     `gorm:"primaryKey" json:"id"`
	MedicalRecordID int64     `gorm:"not null;index" json:"medical_record_id"`
	DiagnosisCode   string    `gorm:"size:10;not null" json:"diagnosis_code"`
	DiagnosisName   string    `gorm:"size:200" json:"diagnosis_name"` // 诊断名称快照
	IsPrimary       bool      `gorm:"not null;default:false" json:"is_primary"`
	SortOrder       int       `gorm:"not null;default:0" json:"sort_order"`
	CreatedAt       time.Time `json:"created_at"`
}

func (MedicalRecordDiagnosis) TableName() string { return "medical_record_diagnoses" }

// Charge 合并结算单（挂号+诊查+治疗+检查+药费合并收费）。
type Charge struct {
	ID             int64          `gorm:"primaryKey" json:"id"`
	ChargeNo       string         `gorm:"size:30;uniqueIndex;not null" json:"charge_no"`
	VisitID        int64          `gorm:"not null;index" json:"visit_id"`
	PatientID      int64          `gorm:"not null;index" json:"patient_id"`
	PatientName    string         `gorm:"size:50;not null" json:"patient_name"`
	TotalAmount    int64          `gorm:"not null;default:0" json:"total_amount"`               // 合计（分）
	DiscountAmount int64          `gorm:"not null;default:0" json:"discount_amount"`            // 优惠（分）
	PayableAmount  int64          `gorm:"not null;default:0" json:"payable_amount"`             // 应收（分）
	PaidAmount     int64          `gorm:"not null;default:0" json:"paid_amount"`                // 实收（分）
	Status         string         `gorm:"size:20;not null;default:pending;index" json:"status"` // pending/paid/refunded
	OperatorName   string         `gorm:"size:50" json:"operator_name"`
	PaidAt         *time.Time     `json:"paid_at"`
	RefundedAt     *time.Time     `json:"refunded_at"`
	Remarks        string         `gorm:"size:500" json:"remarks"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
	Items          []ChargeItem   `gorm:"foreignKey:ChargeID" json:"items,omitempty"`
}

func (Charge) TableName() string { return "charges" }

// ChargeItem 结算单明细（多费用项）。
// 凭证行：与 Charge 共同构成唯一记账凭证，所有收入/计费报表只统计本表。
type ChargeItem struct {
	ID int64 `gorm:"primaryKey" json:"id"`
	// ChargeID 所属结算单。visit_id 同一结算单内恒等于 Charge.VisitID。
	ChargeID int64  `gorm:"not null;index" json:"charge_id"`
	VisitID  *int64 `gorm:"index" json:"visit_id"`
	ItemType string `gorm:"size:20;not null" json:"item_type"` // registration/consultation/treatment/examination/drug/consumable/clinical_service
	// ItemID 业务对象 ID（drugs.id / clinical_services.id …），含义随 ItemType 而定。
	ItemID *int64 `json:"item_id"`
	// SourceRecordID 本行对应的应收计费项目源（charge_records.id，000040）。
	// ItemID 是多态引用、缺少判别字段，追溯回源须用本列显式记录。
	SourceRecordID *int64    `gorm:"index" json:"source_record_id"`
	ItemName       string    `gorm:"size:200;not null" json:"item_name"`
	Quantity       int       `gorm:"not null;default:1" json:"quantity"`
	UnitPrice      int64     `gorm:"not null;default:0" json:"unit_price"`
	Amount         int64     `gorm:"not null;default:0" json:"amount"`
	SortOrder      int       `gorm:"not null;default:0" json:"sort_order"`
	CreatedAt      time.Time `json:"created_at"`
}

func (ChargeItem) TableName() string { return "charge_items" }
