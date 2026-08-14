package model

import (
	"time"
)

// Prescription 处方。
// 状态机：pending_review → reviewed_passed/reviewed_rejected；
// reviewed_passed → dispensing → dispensed → returned；可 cancel。
// 处方类型：0普通 1麻醉 2精神一类 3精神二类 4毒性 5放射性。
type Prescription struct {
	ID                       int64      `gorm:"primaryKey" json:"id"`
	PrescriptionNo           string     `gorm:"size:30;uniqueIndex;not null" json:"prescription_no"`
	PatientID                int64      `gorm:"index" json:"patient_id"` // 二期关联 patients
	PatientName              string     `gorm:"size:50;not null" json:"patient_name"`
	PatientGender            string     `gorm:"size:10" json:"patient_gender"`
	PatientAge               string     `gorm:"size:10" json:"patient_age"`
	PatientCardNo            string     `gorm:"size:50" json:"patient_card_no"`
	IsPregnant               bool       `gorm:"not null;default:false" json:"is_pregnant"`
	IsLactating              bool       `gorm:"not null;default:false" json:"is_lactating"`
	DiagnosisCode            string     `gorm:"size:10;index" json:"diagnosis_code"` // ICD-10 结构化诊断编码（可选）
	Diagnosis                string     `gorm:"type:text" json:"diagnosis"`
	Department               string     `gorm:"size:50" json:"department"`
	DoctorName               string     `gorm:"size:50" json:"doctor_name"`
	PrescriptionType         int        `gorm:"not null;default:0" json:"prescription_type"`
	SpecialControlType       int        `gorm:"not null;default:0" json:"special_control_type"`
	Source                   string     `gorm:"size:20;not null;default:manual" json:"source"`
	VisitID                  *int64     `gorm:"index" json:"visit_id"` // 二期：关联就诊（docs/20 S5）；nil=未关联
	Status                   string     `gorm:"size:20;not null;default:pending_review;index" json:"status"`
	TotalAmount              int64      `gorm:"not null;default:0" json:"total_amount"`
	AuditorID                int64      `json:"auditor_id"`
	AuditorName              string     `gorm:"size:50" json:"auditor_name"`
	DispensingPharmacistID   int64      `json:"dispensing_pharmacist_id"`
	DispensingPharmacistName string     `gorm:"size:50" json:"dispensing_pharmacist_name"`
	CheckerID                int64      `json:"checker_id"`
	CheckerName              string     `gorm:"size:50" json:"checker_name"`
	ReviewedAt               *time.Time `json:"reviewed_at"`
	DispensedAt              *time.Time `json:"dispensed_at"`
	Version                  int        `gorm:"not null;default:0" json:"version"` // 乐观锁
	Remarks                  string     `gorm:"type:text" json:"remarks"`
	CreatedAt                time.Time  `json:"created_at"`
	UpdatedAt                time.Time  `json:"updated_at"`
}

func (Prescription) TableName() string { return "prescriptions" }

// PrescriptionItem 处方明细（开方快照）。quantity 以拆零单位计。
type PrescriptionItem struct {
	ID                int64     `gorm:"primaryKey" json:"id"`
	PrescriptionID    int64     `gorm:"not null;index" json:"prescription_id"`
	LineNo            int       `gorm:"not null" json:"line_no"`
	DrugID            int64     `gorm:"not null;index" json:"drug_id"`
	DrugName          string    `gorm:"size:100;not null" json:"drug_name"`
	Specification     string    `gorm:"size:100" json:"specification"`
	Manufacturer      string    `gorm:"size:100" json:"manufacturer"`
	DosageForm        string    `gorm:"size:30" json:"dosage_form"`
	BaseUnit          string    `gorm:"size:20;not null" json:"base_unit"`
	SplitUnit         string    `gorm:"size:20" json:"split_unit"`
	PackSize          int       `gorm:"not null;default:1" json:"pack_size"`
	IsSplitAllowed    bool      `gorm:"not null;default:false" json:"is_split_allowed"`
	IsSplit           bool      `gorm:"not null;default:false" json:"is_split"` // true=强制拆零发药
	Quantity          int64     `gorm:"not null" json:"quantity"`               // LDU（拆零单位）
	UnitPrice         int64     `gorm:"not null" json:"unit_price"`             // 拆零单价快照（分/拆零单位）
	RetailPrice       int64     `gorm:"not null;default:0" json:"retail_price"` // 盒价快照（分/基本单位）
	Amount            int64     `gorm:"not null" json:"amount"`                 // 精确混合金额
	UsageText         string    `gorm:"size:200" json:"usage_text"`
	Frequency         string    `gorm:"size:50" json:"frequency"`
	Route             string    `gorm:"size:20" json:"route"`       // 给药途径（oral/external/iv/im/iv_drip/inhale/other）
	BatchGroup        string    `gorm:"size:20" json:"batch_group"` // 分批组（如 口服组/输液组1）
	SingleDose        int64     `json:"single_dose"`
	TotalDailyDose    int64     `json:"total_daily_dose"`
	Days              int       `json:"days"`
	DispensedQuantity int64     `gorm:"not null;default:0" json:"dispensed_quantity"`
	ReturnedQuantity  int64     `gorm:"not null;default:0" json:"returned_quantity"`
	Status            string    `gorm:"size:20;not null;default:pending" json:"status"`
	CreatedAt         time.Time `json:"created_at"`
}

func (PrescriptionItem) TableName() string { return "prescription_items" }

// PrescriptionDispenseRecord 发药记录（批次级）。
type PrescriptionDispenseRecord struct {
	ID              int64     `gorm:"primaryKey" json:"id"`
	PrescriptionID  int64     `gorm:"not null;index" json:"prescription_id"`
	ItemID          int64     `gorm:"not null;index" json:"item_id"`
	InventoryID     int64     `gorm:"not null;index" json:"inventory_id"`
	DrugID          int64     `gorm:"not null" json:"drug_id"`
	BatchNo         string    `gorm:"size:50;not null" json:"batch_no"`
	ExpiryDate      time.Time `gorm:"type:date" json:"expiry_date"`
	IsSplit         bool      `gorm:"not null;default:false" json:"is_split"`
	Quantity        int64     `gorm:"not null" json:"quantity"`
	UnitPrice       int64     `gorm:"not null" json:"unit_price"`
	Amount          int64     `gorm:"not null" json:"amount"`
	ReturnQuantity  int64     `gorm:"not null;default:0" json:"return_quantity"`
	DispensedBy     int64     `json:"dispensed_by"`
	DispensedByName string    `gorm:"size:50" json:"dispensed_by_name"`
	CreatedAt       time.Time `json:"created_at"`
}

func (PrescriptionDispenseRecord) TableName() string { return "prescription_dispense_records" }

// PrescriptionAuditLog 处方状态流转日志。
type PrescriptionAuditLog struct {
	ID             int64     `gorm:"primaryKey" json:"id"`
	PrescriptionID int64     `gorm:"not null;index" json:"prescription_id"`
	Action         string    `gorm:"size:30;not null" json:"action"`
	FromStatus     string    `gorm:"size:20" json:"from_status"`
	ToStatus       string    `gorm:"size:20" json:"to_status"`
	OperatorID     int64     `json:"operator_id"`
	OperatorName   string    `gorm:"size:50" json:"operator_name"`
	Remarks        string    `gorm:"size:500" json:"remarks"`
	CreatedAt      time.Time `json:"created_at"`
}

func (PrescriptionAuditLog) TableName() string { return "prescription_audit_logs" }
