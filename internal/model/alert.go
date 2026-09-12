package model

import (
	"time"
)

// StockAlert 预警/提醒记录。
// alert_type：expiry(近效期) / expired(已过期) / below_min(低于下限)。
type StockAlert struct {
	ID             int64      `gorm:"primaryKey" json:"id"`
	AlertType      string     `gorm:"size:20;not null;index" json:"alert_type"`
	DrugID         int64      `gorm:"not null" json:"drug_id"`
	LocationID     int64      `gorm:"not null;default:0" json:"location_id"`
	BatchNo        string     `gorm:"size:50;not null;default:''" json:"batch_no"` // 无批次概念（如低于下限）时为空串，参与 open 去重键
	ExpiryDate     *time.Time `gorm:"type:date" json:"expiry_date"`
	Quantity       int64      `json:"quantity"`
	Message        string     `gorm:"size:300" json:"message"`
	Status         string     `gorm:"size:20;not null;default:open;index" json:"status"` // open/resolved/ignored
	CreatedAt      time.Time  `json:"created_at"`
	ResolvedAt     *time.Time `json:"resolved_at"`
	ResolvedBy     int64      `json:"resolved_by"`
	ResolvedByName string     `gorm:"size:50" json:"resolved_by_name"`
}

func (StockAlert) TableName() string { return "stock_alerts" }
