package model

import (
	"time"

	"gorm.io/gorm"
)

// Supplier 供应商。
type Supplier struct {
	ID              int64          `gorm:"primaryKey" json:"id"`
	Code            string         `gorm:"size:20;uniqueIndex;not null" json:"code"`
	Name            string         `gorm:"size:100;not null" json:"name"`
	ContactPerson   string         `gorm:"size:50" json:"contact_person"`
	Phone           string         `gorm:"size:20" json:"phone"`
	Address         string         `gorm:"size:200" json:"address"`
	QualificationNo string         `gorm:"size:50" json:"qualification_no"`
	Status          int            `gorm:"not null;default:1" json:"status"`
	Remarks         string         `gorm:"type:text" json:"remarks"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Supplier) TableName() string { return "suppliers" }

// DrugSupplier 药品-供应商供货关系。
type DrugSupplier struct {
	ID             int64     `gorm:"primaryKey" json:"id"`
	DrugID         int64     `gorm:"not null;uniqueIndex:uq_drug_supplier" json:"drug_id"`
	SupplierID     int64     `gorm:"not null;uniqueIndex:uq_drug_supplier" json:"supplier_id"`
	IsDefault      bool      `gorm:"not null;default:false" json:"is_default"`
	PurchasePrice  int64     `json:"purchase_price"`
	LastPurchaseAt time.Time `json:"last_purchase_at"`
	Status         int       `gorm:"not null;default:1" json:"status"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

func (DrugSupplier) TableName() string { return "drug_suppliers" }
