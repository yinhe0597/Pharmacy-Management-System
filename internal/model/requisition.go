package model

import "time"

// RequisitionOrder 耗材领用/补发登记单（docs/18：药房护士核心职能）。
type RequisitionOrder struct {
	ID            int64     `gorm:"primaryKey" json:"id"`
	RequisitionNo string    `gorm:"size:30;uniqueIndex;not null" json:"requisition_no"`
	LocationID    int64     `gorm:"not null" json:"location_id"`
	Purpose       string    `gorm:"size:20;not null;default:supplement" json:"purpose"` // clinical/supplement/other
	Reason        string    `gorm:"size:200" json:"reason"`
	OperatorID    int64     `json:"operator_id"`
	OperatorName  string    `gorm:"size:50" json:"operator_name"`
	Status        string    `gorm:"size:20;not null;default:completed" json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (RequisitionOrder) TableName() string { return "requisition_orders" }

// RequisitionOrderItem 领用/补发登记单明细。
type RequisitionOrderItem struct {
	ID                 int64     `gorm:"primaryKey" json:"id"`
	RequisitionOrderID int64     `gorm:"not null;index" json:"requisition_order_id"`
	DrugID             int64     `gorm:"not null;index" json:"drug_id"`
	BatchNo            string    `gorm:"size:50" json:"batch_no"`
	IsSplit            bool      `gorm:"not null;default:false" json:"is_split"`
	Quantity           int64     `gorm:"not null" json:"quantity"`
	UnitPrice          int64     `gorm:"not null;default:0" json:"unit_price"`
	CreatedAt          time.Time `json:"created_at"`
}

func (RequisitionOrderItem) TableName() string { return "requisition_order_items" }
