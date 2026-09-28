// Package model 临床服务与计费相关数据模型。
package model

import (
	"time"

	"gorm.io/gorm"
)

// ClinicalService 诊疗项目（手法复位/静脉注射等），不入药房库存，独立计价。
type ClinicalService struct {
	ID        int64          `gorm:"primaryKey" json:"id"`
	Code      string         `gorm:"size:20;uniqueIndex;not null" json:"code"`
	Name      string         `gorm:"size:100;not null" json:"name"`
	Category  string         `gorm:"size:50" json:"category"`
	UnitPrice int64          `gorm:"not null;default:0" json:"unit_price"`
	Unit      string         `gorm:"size:20;not null;default:次" json:"unit"`
	Status    int            `gorm:"not null;default:1" json:"status"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (ClinicalService) TableName() string { return "clinical_services" }

// ChargeRecord 应收计费项目源（药品/耗材/诊疗项目统一入口）。
//
// 定位：**不是记账凭证**。唯一记账凭证是 charges（结算单）+ charge_items（费用行）。
// 本表只承载「已产生、待归集到结算单的费用项」，由结算单按 visit_id 精确消费；
// 任何报表都不应直接统计本表，否则会与已结算凭证重复计费。
type ChargeRecord struct {
	ID            int64     `gorm:"primaryKey" json:"id"`
	VisitID       *int64    `gorm:"index" json:"visit_id"` // 关联就诊（结算归集依据；000040）
	PatientID     int64     `gorm:"index" json:"patient_id"`
	PatientName   string    `gorm:"size:50;not null" json:"patient_name"`
	PatientCardNo string    `gorm:"size:50" json:"patient_card_no"`
	ItemType      string    `gorm:"size:20;not null" json:"item_type"` // drug / consumable / clinical_service
	ItemID        *int64    `json:"item_id"`
	ItemName      string    `gorm:"size:100;not null" json:"item_name"` // 快照名称
	Quantity      int       `gorm:"not null;default:1" json:"quantity"`
	UnitPrice     int64     `gorm:"not null" json:"unit_price"`
	Amount        int64     `gorm:"not null" json:"amount"`               // 正=收费，负=退费冲正
	Voided        bool      `gorm:"not null;default:false" json:"voided"` // 是否已红冲
	RefType       string    `gorm:"size:20;index" json:"ref_type"`        // 来源单据类型（prescription/charge_void/...）
	RefID         int64     `gorm:"index" json:"ref_id"`                  // 来源单据 ID
	OperatorID    *int64    `json:"operator_id"`
	OperatorName  string    `gorm:"size:50" json:"operator_name"`
	Remarks       string    `gorm:"type:text" json:"remarks"`
	CreatedAt     time.Time `json:"created_at"`
}

func (ChargeRecord) TableName() string { return "charge_records" }
