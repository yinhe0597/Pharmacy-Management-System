package model

import (
	"time"

	"gorm.io/gorm"
)

// Consultation 用药咨询记录。
type Consultation struct {
	ID          int64     `gorm:"primaryKey" json:"id"`
	PatientName string    `gorm:"size:50" json:"patient_name"`
	PatientID   int64     `json:"patient_id"`
	DrugID      int64     `gorm:"index" json:"drug_id"`
	Question    string    `gorm:"type:text;not null" json:"question"`
	Answer      string    `gorm:"type:text" json:"answer"`
	Consultant  string    `gorm:"size:50" json:"consultant"`
	Contact     string    `gorm:"size:50" json:"contact"`
	ConsultedAt time.Time      `json:"consulted_at"`
	CreatedAt   time.Time      `json:"created_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Consultation) TableName() string { return "consultations" }

// AdverseReaction 不良反应登记。
type AdverseReaction struct {
	ID           int64     `gorm:"primaryKey" json:"id"`
	PatientName  string    `gorm:"size:50" json:"patient_name"`
	PatientID    int64     `json:"patient_id"`
	DrugID       int64     `gorm:"index" json:"drug_id"`
	BatchNo      string    `gorm:"size:50" json:"batch_no"`
	ReactionDesc string    `gorm:"type:text;not null" json:"reaction_desc"`
	Severity     int       `gorm:"not null;default:1" json:"severity"` // 1轻 2中 3重
	Outcome      string    `gorm:"size:20" json:"outcome"`
	Reporter     string    `gorm:"size:50" json:"reporter"`
	ReportDate   time.Time      `gorm:"type:date;not null" json:"report_date"`
	CreatedAt    time.Time      `json:"created_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (AdverseReaction) TableName() string { return "adverse_reactions" }

// MedicationGuidance 患者用药指导记录。
type MedicationGuidance struct {
	ID             int64     `gorm:"primaryKey" json:"id"`
	PrescriptionID int64     `json:"prescription_id"`
	PatientName    string    `gorm:"size:50" json:"patient_name"`
	DrugID         int64     `gorm:"index" json:"drug_id"`
	Content        string    `gorm:"type:text;not null" json:"content"`
	Pharmacist     string    `gorm:"size:50" json:"pharmacist"`
	GuidedAt       time.Time      `json:"guided_at"`
	CreatedAt      time.Time      `json:"created_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

func (MedicationGuidance) TableName() string { return "medication_guidances" }
