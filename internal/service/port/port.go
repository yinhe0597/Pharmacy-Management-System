// Package port 定义一期与二期（诊疗模块）之间的能力接口。
// 一期提供简易实现，二期替换为完整实现，接口签名保持稳定（见 docs/05）。
package port

import (
	"context"
	"time"
)

// Patient 患者档案（二期完整对象；一期简易实现仅保留标识信息）。
type Patient struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Gender    string    `json:"gender"`
	Age       string    `json:"age"`
	CardNo    string    `json:"card_no"`
	Phone     string    `json:"phone"`
	CreatedAt time.Time `json:"created_at"`
}

// Allergy 过敏信息。
type Allergy struct {
	DrugName string `json:"drug_name"`
	Reaction string `json:"reaction"`
	Severity int    `json:"severity"` // 1轻 2中 3重
}

// MedicationRecord 用药史。
type MedicationRecord struct {
	DrugName  string `json:"drug_name"`
	Dosage    string `json:"dosage"`
	BeginDate string `json:"begin_date"`
	EndDate   string `json:"end_date"`
}

// IPatientService 患者服务。一期为 SimplePatientService（仅存储标识字符串）。
type IPatientService interface {
	Register(ctx context.Context, p *Patient) (int64, error)
	GetPatient(ctx context.Context, patientID int64) (*Patient, error)
	// 二期：处方审核时自动带入过敏史/用药史提醒。
	GetAllergies(ctx context.Context, patientID int64) ([]Allergy, error)
	GetMedicationHistory(ctx context.Context, patientID int64) ([]MedicationRecord, error)
}

// ReserveItem 预占项。
// 分配模式：
//   - IsSplit=true            ：强制拆零，数量按拆零单位，仅从拆零库存分配
//   - IsSplit=false, Mixed=false：整盒口径，数量按基本单位，仅从整盒库存分配
//   - IsSplit=false, Mixed=true ：混合发药，数量按 LDU（拆零单位），
//     自动拆为「整盒部分（按盒）× 整盒库存 + 零头部分 × 拆零库存」
//
// ItemID 为可选业务明细 ID（处方明细），用于预占与发药记录的批次归属。
type ReserveItem struct {
	DrugID     int64 `json:"drug_id"`
	LocationID int64 `json:"location_id"`
	IsSplit    bool  `json:"is_split"`
	Mixed      bool  `json:"mixed"`
	Quantity   int64 `json:"quantity"`
	ItemID     int64 `json:"item_id"`
}

// ReservationResult 预占结果。
type ReservationResult struct {
	DrugID   int64 `json:"drug_id"`
	Quantity int64 `json:"quantity"`
	Reserved int64 `json:"reserved"`
	Shortage int64 `json:"shortage"` // >0 表示不足量
}

// BatchQty 指定批次的分配数量。
type BatchQty struct {
	InventoryID int64  `json:"inventory_id"`
	BatchNo     string `json:"batch_no"`
	ExpiryDate  string `json:"expiry_date"`
	IsSplit     bool   `json:"is_split"`
	Quantity    int64  `json:"quantity"`
}

// DispenseItem 发药项。Quantity 以 IsSplit 对应口径计。
type DispenseItem struct {
	DrugID          int64      `json:"drug_id"`
	LocationID      int64      `json:"location_id"`
	IsSplit         bool       `json:"is_split"`
	Quantity        int64      `json:"quantity"`
	BatchAllocation []BatchQty `json:"batch_allocation"` // 指定批次（可选，默认 FEFO）
}

// Availability 某药品可用库存。
type Availability struct {
	LocationID   int64  `json:"location_id"`
	LocationName string `json:"location_name"`
	Available    int64  `json:"available"`
	ExpirySoon   bool   `json:"expiry_soon"`
}

// IStockService 库存服务接口，一期由 InventoryService 实现。
type IStockService interface {
	ReserveStock(ctx context.Context, refType string, refID int64, items []ReserveItem) ([]ReservationResult, error)
	DispenseAndReduceStock(ctx context.Context, refType string, refID int64, items []DispenseItem) error
	CancelReservation(ctx context.Context, refType string, refID int64, items []ReserveItem) error
	GetDrugAvailability(ctx context.Context, drugID int64) ([]Availability, error)
}

// PriceLine 计价行。
type PriceLine struct {
	LineNo    int    `json:"line_no"`
	ItemType  string `json:"item_type"` // drug / registration / diagnosis / other
	RefID     int64  `json:"ref_id"`
	Quantity  int64  `json:"quantity"`
	UnitPrice int64  `json:"unit_price"` // 分
	Amount    int64  `json:"amount"`     // 分
}

// IPricingService 计价服务接口。一期仅计算药费，二期扩展多费用项。
type IPricingService interface {
	CalculatePrescriptionAmount(ctx context.Context, itemIDs []int64) ([]PriceLine, error)
}
