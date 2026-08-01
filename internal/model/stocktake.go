package model

import (
	"time"
)

// Stocktake 盘点单。状态：draft/counting/adjusted/completed/cancelled。
// 类型：1周期盘点 2动态盘点。
type Stocktake struct {
	ID          int64      `gorm:"primaryKey" json:"id"`
	StocktakeNo string     `gorm:"size:30;uniqueIndex;not null" json:"stocktake_no"`
	LocationID  int64      `gorm:"not null" json:"location_id"`
	Type        int        `gorm:"not null" json:"type"`
	Status      string     `gorm:"size:20;not null;default:draft" json:"status"`
	StartedBy   int64      `json:"started_by"`
	StartedAt   *time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
	Remarks     string     `gorm:"size:200" json:"remarks"`
}

func (Stocktake) TableName() string { return "stocktakes" }

// StocktakeItem 盘点明细。数量口径随 inventory 行（is_split）。
type StocktakeItem struct {
	ID              int64     `gorm:"primaryKey" json:"id"`
	StocktakeID     int64     `gorm:"not null;index" json:"stocktake_id"`
	InventoryID     int64     `gorm:"not null" json:"inventory_id"`
	DrugID          int64     `gorm:"not null" json:"drug_id"`
	BatchNo         string    `gorm:"size:50;not null" json:"batch_no"`
	ExpiryDate      time.Time `gorm:"type:date;not null" json:"expiry_date"`
	IsSplit         bool      `gorm:"not null;default:false" json:"is_split"`
	BookQuantity    int64     `gorm:"not null" json:"book_quantity"`
	CountedQuantity int64     `json:"counted_quantity"`
	Difference      int64     `gorm:"default:0" json:"difference"`
	Status          string    `gorm:"size:20;not null;default:pending" json:"status"`
	CreatedAt       time.Time `json:"created_at"`
}

func (StocktakeItem) TableName() string { return "stocktake_items" }
