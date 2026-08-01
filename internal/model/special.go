package model

import (
	"time"
)

// SpecialDrugLedger 麻精药品专账。
type SpecialDrugLedger struct {
	ID             int64     `gorm:"primaryKey" json:"id"`
	DrugID         int64     `gorm:"not null;index" json:"drug_id"`
	BatchNo        string    `gorm:"size:50;not null" json:"batch_no"`
	PrescriptionID int64     `json:"prescription_id"`
	LogType        string    `gorm:"size:20;not null" json:"log_type"` // prescribe/dispense/return/ampoule_return/usage
	Quantity       int64     `gorm:"not null" json:"quantity"`         // 有符号
	PatientName    string    `gorm:"size:50" json:"patient_name"`
	PatientCardNo  string    `gorm:"size:50" json:"patient_card_no"`
	OperatorID     int64     `json:"operator_id"`
	OperatorName   string    `gorm:"size:50" json:"operator_name"`
	Notes          string    `gorm:"size:200" json:"notes"`
	CreatedAt      time.Time `json:"created_at"`
}

func (SpecialDrugLedger) TableName() string { return "special_drug_ledgers" }

// AmpouleReturn 空安瓿回收。
type AmpouleReturn struct {
	ID             int64     `gorm:"primaryKey" json:"id"`
	DrugID         int64     `gorm:"not null" json:"drug_id"`
	BatchNo        string    `gorm:"size:50" json:"batch_no"`
	PrescriptionID int64     `json:"prescription_id"`
	PatientName    string    `gorm:"size:50" json:"patient_name"`
	Quantity       int       `gorm:"not null" json:"quantity"`
	ReturnedBy     string    `gorm:"size:50" json:"returned_by"`
	VerifiedBy     string    `gorm:"size:50" json:"verified_by"`
	ReturnDate     time.Time `gorm:"type:date;not null" json:"return_date"`
	Status         string    `gorm:"size:20;not null;default:pending" json:"status"` // pending/verified
	CreatedAt      time.Time `json:"created_at"`
}

func (AmpouleReturn) TableName() string { return "ampoule_returns" }
