package model

import "time"

// SplitOrder 拆零操作单（docs/13 F4）。记录拆零来源行/结果行与操作/复核人。
type SplitOrder struct {
	ID            int64      `gorm:"primaryKey" json:"id"`
	SplitNo       string     `gorm:"size:30;uniqueIndex;not null" json:"split_no"`
	InventoryID   int64      `gorm:"not null" json:"inventory_id"`
	DrugID        int64      `gorm:"not null" json:"drug_id"`
	LocationID    int64      `gorm:"not null" json:"location_id"`
	BatchNo       string     `gorm:"size:50;not null" json:"batch_no"`
	ExpiryDate    *time.Time `gorm:"type:date" json:"expiry_date"`
	Boxes         int64      `gorm:"not null" json:"boxes"`
	Units         int64      `gorm:"not null" json:"units"`
	Damaged       int64      `gorm:"not null;default:0" json:"damaged"`
	SplitUnitCost int64      `gorm:"not null;default:0" json:"split_unit_cost"` // 分/拆零单位
	OperatorID    int64      `json:"operator_id"`
	OperatorName  string     `gorm:"size:50" json:"operator_name"`
	ReviewerID    int64      `json:"reviewer_id"` // 麻精双人复核人
	ReviewerName  string     `gorm:"size:50" json:"reviewer_name"`
	Remarks       string     `gorm:"size:200" json:"remarks"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func (SplitOrder) TableName() string { return "split_orders" }
