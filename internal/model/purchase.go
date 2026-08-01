package model

import (
	"time"
)

// PurchaseOrder 采购单。状态：draft/submitted/partial/received/cancelled。
type PurchaseOrder struct {
	ID          int64      `gorm:"primaryKey" json:"id"`
	PurchaseNo  string     `gorm:"size:30;uniqueIndex;not null" json:"purchase_no"`
	SupplierID  int64      `gorm:"not null;index" json:"supplier_id"`
	Status      string     `gorm:"size:20;not null;default:draft" json:"status"`
	ExpectedAt  *time.Time `gorm:"type:date" json:"expected_at"`
	TotalAmount int64      `gorm:"not null;default:0" json:"total_amount"`
	CreatedBy   int64      `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	ReceivedAt  *time.Time `json:"received_at"`
	Remarks     string     `gorm:"type:text" json:"remarks"`
}

func (PurchaseOrder) TableName() string { return "purchase_orders" }

// PurchaseOrderItem 采购单明细。
type PurchaseOrderItem struct {
	ID               int64 `gorm:"primaryKey" json:"id"`
	PurchaseOrderID  int64 `gorm:"not null;index" json:"purchase_order_id"`
	DrugID           int64 `gorm:"not null" json:"drug_id"`
	Quantity         int64 `gorm:"not null" json:"quantity"`
	UnitPrice        int64 `gorm:"not null" json:"unit_price"`
	Amount           int64 `gorm:"not null;default:0" json:"amount"`
	ReceivedQuantity int64 `gorm:"not null;default:0" json:"received_quantity"`
}

func (PurchaseOrderItem) TableName() string { return "purchase_order_items" }

// PurchaseReceipt 采购收货（质检）单。状态：pending_quality/partially_received/received/qc_failed。
type PurchaseReceipt struct {
	ID              int64      `gorm:"primaryKey" json:"id"`
	ReceiptNo       string     `gorm:"size:30;uniqueIndex;not null" json:"receipt_no"`
	PurchaseOrderID int64      `gorm:"index" json:"purchase_order_id"`
	SupplierID      int64      `gorm:"not null" json:"supplier_id"`
	Status          string     `gorm:"size:20;not null;default:pending_quality" json:"status"`
	TotalAmount     int64      `gorm:"not null;default:0" json:"total_amount"`
	ReceivedAt      *time.Time `json:"received_at"`
	ReceivedBy      int64      `json:"received_by"`
	CreatedAt       time.Time  `json:"created_at"`
}

func (PurchaseReceipt) TableName() string { return "purchase_receipts" }

// PurchaseReceiptItem 收货明细，批次绑定（入库即绑定批号效期）。
type PurchaseReceiptItem struct {
	ID               int64     `gorm:"primaryKey" json:"id"`
	ReceiptID        int64     `gorm:"not null;index" json:"receipt_id"`
	OrderItemID      int64     `gorm:"index" json:"order_item_id"`
	DrugID           int64     `gorm:"not null" json:"drug_id"`
	OrderedQuantity  int64     `gorm:"not null" json:"ordered_quantity"`
	ReceivedQuantity int64     `gorm:"not null" json:"received_quantity"`
	BatchNo          string    `gorm:"size:50;not null" json:"batch_no"`
	ExpiryDate       time.Time `gorm:"type:date;not null" json:"expiry_date"`
	UnitPrice        int64     `gorm:"not null" json:"unit_price"`
	QCResult         int       `gorm:"not null;default:1" json:"qc_result"` // 1合格 2不合格
	QCNotes          string    `gorm:"size:200" json:"qc_notes"`
}

func (PurchaseReceiptItem) TableName() string { return "purchase_receipt_items" }
