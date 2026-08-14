package model

import "time"

// Patient 患者档案（二期完整实现，对应 port.Patient）。
type Patient struct {
	ID          int64     `gorm:"primaryKey" json:"id"`
	CardNo      string    `gorm:"size:50;uniqueIndex;not null" json:"card_no"`
	Name        string    `gorm:"size:50;not null" json:"name"`
	Gender      string    `gorm:"size:10" json:"gender"`
	Age         string    `gorm:"size:10" json:"age"`
	Phone       string    `gorm:"size:20" json:"phone"`
	IsLactating bool      `gorm:"not null;default:false" json:"is_lactating"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Patient) TableName() string { return "patients" }

// PatientAllergy 患者过敏史（对应 port.Allergy）。
type PatientAllergy struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	PatientID int64     `gorm:"not null;index" json:"patient_id"`
	DrugName  string    `gorm:"size:100;not null" json:"drug_name"`
	Reaction  string    `gorm:"size:200" json:"reaction"`
	Severity  int       `gorm:"not null;default:1" json:"severity"` // 1轻 2中 3重
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (PatientAllergy) TableName() string { return "patient_allergies" }
