package model

import (
	"time"
)

// InventoryLocation 库存地点：1药库 2药房 3科室。
type InventoryLocation struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	Code      string    `gorm:"size:20;uniqueIndex;not null" json:"code"`
	Name      string    `gorm:"size:50;not null" json:"name"`
	Type      int       `gorm:"not null" json:"type"`
	ParentID  int64     `gorm:"not null;default:0" json:"parent_id"`
	IsActive  bool      `gorm:"not null;default:true" json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (InventoryLocation) TableName() string { return "inventory_locations" }

// DrugStockSetting 库存上下限设置，数量以拆零单位（LDU）计。
type DrugStockSetting struct {
	ID          int64     `gorm:"primaryKey" json:"id"`
	DrugID      int64     `gorm:"not null;uniqueIndex:uq_stock_setting" json:"drug_id"`
	LocationID  int64     `gorm:"not null;uniqueIndex:uq_stock_setting" json:"location_id"`
	MinQuantity int64     `gorm:"not null;default:0" json:"min_quantity"`
	MaxQuantity int64     `gorm:"not null;default:0" json:"max_quantity"`
	ReorderQty  int64     `gorm:"not null;default:0" json:"reorder_qty"`
	IsEnabled   bool      `gorm:"not null;default:true" json:"is_enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (DrugStockSetting) TableName() string { return "drug_stock_settings" }

// Inventory 库存批次。is_split=false 数量按基本单位，true 按拆零单位。
type Inventory struct {
	ID               int64     `gorm:"primaryKey" json:"id"`
	DrugID           int64     `gorm:"not null;index:idx_inventory_drug_loc" json:"drug_id"`
	LocationID       int64     `gorm:"not null;index:idx_inventory_drug_loc" json:"location_id"`
	BatchNo          string    `gorm:"size:50;not null" json:"batch_no"`
	ExpiryDate       time.Time `gorm:"type:date" json:"expiry_date"`
	Quantity         int64     `gorm:"not null;default:0" json:"quantity"`
	ReservedQuantity int64     `gorm:"not null;default:0" json:"reserved_quantity"`
	IsSplit          bool      `gorm:"not null;default:false" json:"is_split"`
	UnitPrice        int64     `gorm:"not null;default:0" json:"unit_price"`
	ReceivedAt       time.Time `json:"received_at"`
	Status           int       `gorm:"not null;default:1" json:"status"` // 1正常 2过期锁定 3手动锁定
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (Inventory) TableName() string { return "inventory" }

// Available 可用数量 = 数量 - 已预占。
func (i *Inventory) Available() int64 { return i.Quantity - i.ReservedQuantity }

// IsNormal 是否正常可用批次。
func (i *Inventory) IsNormal() bool { return i.Status == 1 && i.Quantity > i.ReservedQuantity }

// InventoryTransaction 库存流水。quantity 有符号，正=入，负=出。
type InventoryTransaction struct {
	ID             int64      `gorm:"primaryKey" json:"id"`
	TransactionNo  string     `gorm:"size:30;uniqueIndex;not null" json:"transaction_no"`
	DrugID         int64      `gorm:"not null" json:"drug_id"`
	LocationID     int64      `gorm:"not null" json:"location_id"`
	BatchNo        string     `gorm:"size:50" json:"batch_no"`
	ExpiryDate     *time.Time `gorm:"type:date" json:"expiry_date"`
	Quantity       int64      `gorm:"not null" json:"quantity"`
	IsSplit        bool       `gorm:"not null;default:false" json:"is_split"`
	BeforeQuantity int64      `json:"before_quantity"`
	AfterQuantity  int64      `json:"after_quantity"`
	TxnType        string     `gorm:"size:20;not null" json:"txn_type"`
	RefType        string     `gorm:"size:30" json:"ref_type"`
	RefID          int64      `json:"ref_id"`
	OperatorID     int64      `json:"operator_id"`
	OperatorName   string     `gorm:"size:50" json:"operator_name"`
	Remarks        string     `gorm:"size:200" json:"remarks"`
	CreatedAt      time.Time  `gorm:"index" json:"created_at"`
}

func (InventoryTransaction) TableName() string { return "inventory_transactions" }

// StockReservation 库存预占记录。
// NeedSplit=true 表示该整盒预占在发药时需拆零发放（自动拆零）；SplitUnits 为该盒用于处方的片数。
type StockReservation struct {
	ID            int64      `gorm:"primaryKey" json:"id"`
	ReservationNo string     `gorm:"size:30;uniqueIndex;not null" json:"reservation_no"`
	RefType       string     `gorm:"size:30;not null;index" json:"ref_type"`
	RefID         int64      `gorm:"not null;index" json:"ref_id"`
	ItemID        int64      `json:"item_id"`
	InventoryID   int64      `gorm:"not null;index" json:"inventory_id"`
	DrugID        int64      `gorm:"not null" json:"drug_id"`
	LocationID    int64      `gorm:"not null" json:"location_id"`
	BatchNo       string     `gorm:"size:50;not null" json:"batch_no"`
	ExpiryDate    time.Time  `gorm:"type:date" json:"expiry_date"`
	IsSplit       bool       `gorm:"not null;default:false" json:"is_split"`
	Quantity      int64      `gorm:"not null" json:"quantity"`
	NeedSplit     bool       `gorm:"not null;default:false" json:"need_split"`
	SplitUnits    int64      `gorm:"not null;default:0" json:"split_units"`
	Status        string     `gorm:"size:20;not null;default:active" json:"status"` // active/consumed/released
	CreatedAt     time.Time  `json:"created_at"`
	ReleasedAt    *time.Time `json:"released_at"`
}

func (StockReservation) TableName() string { return "stock_reservations" }
