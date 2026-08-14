package service

import (
	"context"

	"gorm.io/gorm"

	"yaofang/internal/domain/enum"
	"yaofang/internal/model"
	"yaofang/internal/pkg/errs"
	"yaofang/internal/repository"
)

// ClinicalService 诊疗项目与计费服务。
type ClinicalService struct {
	db *gorm.DB
}

func NewClinicalService(db *gorm.DB) *ClinicalService {
	return &ClinicalService{db: db}
}

// ---- 诊疗项目 CRUD ----

func (s *ClinicalService) CreateService(ctx context.Context, svc *model.ClinicalService) error {
	if svc.Code == "" || svc.Name == "" {
		return errs.ErrBadRequest
	}
	return repository.NewClinicalServiceRepo(s.db).Create(ctx, svc)
}

func (s *ClinicalService) UpdateService(ctx context.Context, id int64, svc *model.ClinicalService) error {
	existing, err := repository.NewClinicalServiceRepo(s.db).GetByID(ctx, id)
	if err != nil {
		return errs.ErrNotFound
	}
	svc.ID = existing.ID
	svc.CreatedAt = existing.CreatedAt
	return repository.NewClinicalServiceRepo(s.db).Update(ctx, svc)
}

func (s *ClinicalService) DeleteService(ctx context.Context, id int64) error {
	return repository.NewClinicalServiceRepo(s.db).Delete(ctx, id)
}

func (s *ClinicalService) ListServices(ctx context.Context, keyword string, page, pageSize int) ([]model.ClinicalService, int64, error) {
	return repository.NewClinicalServiceRepo(s.db).List(ctx, keyword, (page-1)*pageSize, pageSize)
}

// ---- 计费记录 ----

// ChargeInput 计费输入。
type ChargeInput struct {
	PatientName   string `json:"patient_name" binding:"required"`
	PatientCardNo string `json:"patient_card_no"`
	ItemType      string `json:"item_type" binding:"required"` // drug / consumable / clinical_service
	ItemID        *int64 `json:"item_id"`
	ItemName      string `json:"item_name" binding:"required"`
	Quantity      int    `json:"quantity"`
	UnitPrice     int64  `json:"unit_price" binding:"required"`
	Remarks       string `json:"remarks"`
}

// CreateCharge 创建计费记录。
func (s *ClinicalService) CreateCharge(ctx context.Context, input ChargeInput, operatorID int64, operatorName string) (*model.ChargeRecord, error) {
	if input.Quantity <= 0 {
		input.Quantity = 1
	}
	cr := &model.ChargeRecord{
		PatientName:   input.PatientName,
		PatientCardNo: input.PatientCardNo,
		ItemType:      input.ItemType,
		ItemID:        input.ItemID,
		ItemName:      input.ItemName,
		Quantity:      input.Quantity,
		UnitPrice:     input.UnitPrice,
		Amount:        int64(input.Quantity) * input.UnitPrice,
		OperatorID:    &operatorID,
		OperatorName:  operatorName,
		Remarks:       input.Remarks,
	}
	if err := repository.NewChargeRecordRepo(s.db).Create(ctx, cr); err != nil {
		return nil, err
	}
	return cr, nil
}

// ListCharges 查询计费记录。
func (s *ClinicalService) ListCharges(ctx context.Context, keyword string, page, pageSize int) ([]model.ChargeRecord, int64, error) {
	return repository.NewChargeRecordRepo(s.db).List(ctx, keyword, (page-1)*pageSize, pageSize)
}

// ChargePrescription 从已发药处方生成计费记录（幂等：同处方仅计费一次）。
// 逐发药记录生成，取发药快照价（整盒按盒价、拆零按拆零价），金额与发药记录一致。
func (s *ClinicalService) ChargePrescription(ctx context.Context, prescriptionID int64, operatorID int64, operatorName string) ([]model.ChargeRecord, error) {
	chargeRepo := repository.NewChargeRecordRepo(s.db)
	// 幂等：已有正计费则直接返回空（不重复计费）
	if n, err := chargeRepo.CountByRef(ctx, "prescription", prescriptionID); err != nil {
		return nil, err
	} else if n > 0 {
		return nil, nil
	}
	p, err := repository.NewPrescriptionRepo(s.db).GetByID(ctx, prescriptionID)
	if err != nil {
		return nil, err
	}
	records, err := repository.NewDispenseRecordRepo(s.db).ListByPrescription(ctx, prescriptionID)
	if err != nil {
		return nil, err
	}
	return s.buildChargesFromRecords(ctx, p, records, operatorID, operatorName)
}

// buildChargesFromRecords 按发药记录构建计费，补药品名。
func (s *ClinicalService) buildChargesFromRecords(ctx context.Context, p *model.Prescription, records []model.PrescriptionDispenseRecord, operatorID int64, operatorName string) ([]model.ChargeRecord, error) {
	items, err := repository.NewPrescriptionItemRepo(s.db).ListByPrescription(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	nameByDrug := make(map[int64]string, len(items))
	for _, it := range items {
		nameByDrug[it.DrugID] = it.DrugName
	}
	charges := make([]model.ChargeRecord, 0, len(records))
	for _, rec := range records {
		if rec.Quantity <= 0 {
			continue
		}
		itemID := rec.DrugID
		name := nameByDrug[rec.DrugID]
		if name == "" {
			name = rec.BatchNo
		}
		charges = append(charges, model.ChargeRecord{
			PatientName: p.PatientName, PatientCardNo: p.PatientCardNo,
			ItemType: enum.ItemTypeDrug, ItemID: &itemID, ItemName: name,
			Quantity: int(rec.Quantity), UnitPrice: rec.UnitPrice, Amount: rec.Amount,
			RefType: "prescription", RefID: p.ID,
			OperatorID: &operatorID, OperatorName: operatorName, Remarks: "处方发药计费 " + p.PrescriptionNo,
		})
	}
	if err := repository.NewChargeRecordRepo(s.db).CreateBatch(ctx, toChargePtrs(charges)); err != nil {
		return nil, err
	}
	return charges, nil
}

// RefundPrescription 退药冲正：按退药明细生成负金额计费记录（按处方快照价，混合口径）。
func (s *ClinicalService) RefundPrescription(ctx context.Context, prescriptionID int64, returns []ReturnItemInput, operatorID int64, operatorName string) error {
	if len(returns) == 0 {
		return nil
	}
	p, err := repository.NewPrescriptionRepo(s.db).GetByID(ctx, prescriptionID)
	if err != nil {
		return err
	}
	items, err := repository.NewPrescriptionItemRepo(s.db).ListByPrescription(ctx, prescriptionID)
	if err != nil {
		return err
	}
	itemByID := make(map[int64]model.PrescriptionItem, len(items))
	for _, it := range items {
		itemByID[it.ID] = it
	}
	charges := make([]model.ChargeRecord, 0, len(returns))
	for _, in := range returns {
		it, ok := itemByID[in.ItemID]
		if !ok {
			return errs.ErrItemNotFound
		}
		amt := itemAmount(&it, in.ReturnQuantity)
		itemID := it.DrugID
		charges = append(charges, model.ChargeRecord{
			PatientName: p.PatientName, PatientCardNo: p.PatientCardNo,
			ItemType: enum.ItemTypeDrug, ItemID: &itemID, ItemName: it.DrugName,
			Quantity: int(in.ReturnQuantity), UnitPrice: it.UnitPrice, Amount: -amt,
			RefType: "prescription", RefID: prescriptionID,
			OperatorID: &operatorID, OperatorName: operatorName, Remarks: "退药冲正 " + p.PrescriptionNo,
		})
	}
	return repository.NewChargeRecordRepo(s.db).CreateBatch(ctx, toChargePtrs(charges))
}

// itemAmount 按处方明细快照价计算 qty（LDU）金额（与处方计价同口径，docs/13 §4.3）。
func itemAmount(it *model.PrescriptionItem, qty int64) int64 {
	switch {
	case it.IsSplit:
		return qty * it.UnitPrice
	case it.IsSplitAllowed:
		boxes := qty / int64(it.PackSize)
		units := qty % int64(it.PackSize)
		return boxes*it.RetailPrice + units*it.UnitPrice
	default:
		return qty * it.RetailPrice
	}
}

func toChargePtrs(list []model.ChargeRecord) []*model.ChargeRecord {
	out := make([]*model.ChargeRecord, 0, len(list))
	for i := range list {
		out = append(out, &list[i])
	}
	return out
}
