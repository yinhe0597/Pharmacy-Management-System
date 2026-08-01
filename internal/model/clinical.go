// Package model 临床服务与计费相关数据模型。
package model

import (
	"time"

	"gorm.io/gorm"
)

// ClinicalService 诊疗项目（手法复位/静脉注射等），不入药房库存，独立计价。
type ClinicalService struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	Code      string    `gorm:"size:20;uniqueIndex;not null" json:"code"`
	Name      string    `gorm:"size:100;not null" json:"name"`
	Category  string    `gorm:"size:50" json:"category"`
	UnitPrice int64     `gorm:"not null;default:0" json:"unit_price"`
	Unit      string    `gorm:"size:20;not null;default:次" json:"unit"`
	Status    int       `gorm:"not null;default:1" json:"status"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (ClinicalService) TableName() string { return "clinical_services" }

// ChargeRecord 计费记录（药品/耗材/诊疗项目统一入口）。
type ChargeRecord struct {
	ID            int64     `gorm:"primaryKey" json:"id"`
	PatientName   string    `gorm:"size:50;not null" json:"patient_name"`
	PatientCardNo string    `gorm:"size:50" json:"patient_card_no"`
	ItemType      string    `gorm:"size:20;not null" json:"item_type"`   // drug / consumable / clinical_service
	ItemID        *int64    `json:"item_id"`
	ItemName      string    `gorm:"size:100;not null" json:"item_name"` // 快照名称
	Quantity      int       `gorm:"not null;default:1" json:"quantity"`
	UnitPrice     int64     `gorm:"not null" json:"unit_price"`
	Amount        int64     `gorm:"not null" json:"amount"`
	OperatorID    *int64    `json:"operator_id"`
	OperatorName  string    `gorm:"size:50" json:"operator_name"`
	Remarks       string    `gorm:"type:text" json:"remarks"`
	CreatedAt     time.Time `json:"created_at"`
}

func (ChargeRecord) TableName() string { return "charge_records" }
