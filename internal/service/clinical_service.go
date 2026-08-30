package service

import (
	"context"
	"errors"
	"strconv"

	"gorm.io/gorm"

	"yaofang/internal/domain/enum"
	"yaofang/internal/model"
	"yaofang/internal/pkg/errs"
	"yaofang/internal/pkg/money"
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
	// 重复编码友好提示（docs/15 M1）
	if _, err := repository.NewClinicalServiceRepo(s.db).GetByCode(ctx, svc.Code); err == nil {
		return errs.ErrServiceCodeExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
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

// SetServiceStatus 启停用诊疗项目（docs/15 G7）。
func (s *ClinicalService) SetServiceStatus(ctx context.Context, id int64, status int) error {
	if _, err := repository.NewClinicalServiceRepo(s.db).GetByID(ctx, id); err != nil {
		return errs.ErrNotFound
	}
	return repository.NewClinicalServiceRepo(s.db).SetStatus(ctx, id, status)
}

func (s *ClinicalService) ListServices(ctx context.Context, keyword string, page, pageSize int) ([]model.ClinicalService, int64, error) {
	return repository.NewClinicalServiceRepo(s.db).List(ctx, keyword, (page-1)*pageSize, pageSize)
}

// ---- 计费记录 ----

// ChargeInput 计费输入。
type ChargeInput struct {
	PatientID     int64  `json:"patient_id"` // 关联患者档案（可选）
	PatientName   string `json:"patient_name" binding:"required"`
	PatientCardNo string `json:"patient_card_no"`
	ItemType      string `json:"item_type" binding:"required"` // drug / consumable / clinical_service
	ItemID        *int64 `json:"item_id"`
	ItemName      string `json:"item_name" binding:"required"`
	Quantity      int    `json:"quantity"`
	UnitPrice     int64  `json:"unit_price"` // 分；0 允许（免费项）
	Remarks       string `json:"remarks"`
}

// CreateCharge 创建计费记录（docs/15 M4：入参校验 + 项目存在性校验）。
func (s *ClinicalService) CreateCharge(ctx context.Context, input ChargeInput, operatorID int64, operatorName string) (*model.ChargeRecord, error) {
	if input.Quantity < 0 || input.UnitPrice < 0 {
		return nil, errs.ErrBadRequest
	}
	switch input.ItemType {
	case enum.ItemTypeDrug, enum.ItemTypeConsumable:
		// 药品/耗材：提供了 ItemID 时校验存在且启用
		if input.ItemID != nil && *input.ItemID > 0 {
			d, err := repository.NewDrugRepo(s.db).GetByID(ctx, *input.ItemID)
			if err != nil {
				return nil, errs.ErrDrugNotFound
			}
			if d.Status != 1 {
				return nil, errs.ErrDrugInactive
			}
		}
	case "clinical_service":
		if input.ItemID != nil && *input.ItemID > 0 {
			cs, err := repository.NewClinicalServiceRepo(s.db).GetByID(ctx, *input.ItemID)
			if err != nil {
				return nil, errs.ErrNotFound
			}
			if cs.Status != 1 {
				return nil, errs.ErrBadRequest
			}
		}
	default:
		return nil, errs.ErrBadRequest
	}
	if input.Quantity == 0 {
		input.Quantity = 1
	}
	cr := &model.ChargeRecord{
		PatientID:     input.PatientID,
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

// ListCharges 查询计费记录（docs/15 G4：支持来源/类型/患者/日期筛选）。
func (s *ClinicalService) ListCharges(ctx context.Context, f repository.ChargeListFilter, page, pageSize int) ([]model.ChargeRecord, int64, error) {
	return repository.NewChargeRecordRepo(s.db).List(ctx, f, (page-1)*pageSize, pageSize)
}

// VoidCharge 红冲计费记录（docs/15 G5）：标记原单红冲并写负金额冲正单，幂等。
func (s *ClinicalService) VoidCharge(ctx context.Context, id int64, operatorID int64, operatorName string) error {
	chargeRepo := repository.NewChargeRecordRepo(s.db)
	cr, err := chargeRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.ErrNotFound
		}
		return err
	}
	if cr.Voided || cr.Amount <= 0 {
		return errs.ErrChargeVoided
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 条件更新（voided=FALSE）：并发双击时仅一个请求成功，其余整体回滚
		ok, err := repository.NewChargeRecordRepo(tx).MarkVoided(ctx, id)
		if err != nil {
			return err
		}
		if !ok {
			return errs.ErrChargeVoided
		}
		itemID := cr.ItemID
		return repository.NewChargeRecordRepo(tx).Create(ctx, &model.ChargeRecord{
			PatientID: cr.PatientID, PatientName: cr.PatientName, PatientCardNo: cr.PatientCardNo,
			ItemType: cr.ItemType, ItemID: itemID, ItemName: cr.ItemName,
			Quantity: cr.Quantity, UnitPrice: cr.UnitPrice, Amount: -cr.Amount,
			Voided: false, RefType: "charge_void", RefID: id,
			OperatorID: &operatorID, OperatorName: operatorName, Remarks: "红冲 原单#" + strconv.FormatInt(id, 10),
		})
	})
}

// ChargePrescription 从已发药处方生成计费记录（幂等：同处方仅计费一次）。
// 仅已发药（dispensed）处方可计费（docs/15 L3）。
// 整体在事务内完成：锁处方行串行化同一处方的并发计费，幂等检查与写入原子。
func (s *ClinicalService) ChargePrescription(ctx context.Context, prescriptionID int64, operatorID int64, operatorName string) ([]model.ChargeRecord, error) {
	var charges []model.ChargeRecord
	err := s.db.Transaction(func(tx *gorm.DB) error {
		p, err := repository.NewPrescriptionRepo(tx).LockForUpdate(ctx, prescriptionID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errs.ErrNotFound
			}
			return err
		}
		// 幂等：已有正计费则直接返回空（不重复计费）；锁内复查，防并发重复
		n, err := repository.NewChargeRecordRepo(tx).CountByRef(ctx, "prescription", prescriptionID)
		if err != nil {
			return err
		}
		if n > 0 {
			return nil
		}
		if p.Status != "dispensed" {
			return errs.ErrNotDispensed
		}
		records, err := repository.NewDispenseRecordRepo(tx).ListByPrescription(ctx, prescriptionID)
		if err != nil {
			return err
		}
		charges, err = s.buildChargesFromRecordsTx(ctx, tx, p, records, operatorID, operatorName)
		return err
	})
	if err != nil {
		return nil, err
	}
	return charges, nil
}

// buildChargesFromRecordsTx 按发药记录构建计费（在给定事务/连接上写入），补药品名。
func (s *ClinicalService) buildChargesFromRecordsTx(ctx context.Context, tx *gorm.DB, p *model.Prescription, records []model.PrescriptionDispenseRecord, operatorID int64, operatorName string) ([]model.ChargeRecord, error) {
	items, err := repository.NewPrescriptionItemRepo(tx).ListByPrescription(ctx, p.ID)
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
			PatientID: p.PatientID, PatientName: p.PatientName, PatientCardNo: p.PatientCardNo,
			ItemType: enum.ItemTypeDrug, ItemID: &itemID, ItemName: name,
			Quantity: int(rec.Quantity), UnitPrice: rec.UnitPrice, Amount: rec.Amount,
			RefType: "prescription", RefID: p.ID,
			OperatorID: &operatorID, OperatorName: operatorName, Remarks: "处方发药计费 " + p.PrescriptionNo,
		})
	}
	if err := repository.NewChargeRecordRepo(tx).CreateBatch(ctx, toChargePtrs(charges)); err != nil {
		return nil, err
	}
	return charges, nil
}

// RefundPrescription 退药冲正：按退药明细生成负金额计费记录（按处方快照价，混合口径）。
func (s *ClinicalService) RefundPrescription(ctx context.Context, prescriptionID int64, returns []ReturnItemInput, operatorID int64, operatorName string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		return s.refundPrescriptionTx(ctx, tx, prescriptionID, returns, operatorID, operatorName)
	})
}

// RefundPrescriptionTx 退药冲正（事务内版本，供退药流程原子调用，docs/15 H3）。
func (s *ClinicalService) RefundPrescriptionTx(ctx context.Context, tx *gorm.DB, prescriptionID int64, returns []ReturnItemInput, operatorID int64, operatorName string) error {
	return s.refundPrescriptionTx(ctx, tx, prescriptionID, returns, operatorID, operatorName)
}

func (s *ClinicalService) refundPrescriptionTx(ctx context.Context, tx *gorm.DB, prescriptionID int64, returns []ReturnItemInput, operatorID int64, operatorName string) error {
	if len(returns) == 0 {
		return nil
	}
	p, err := repository.NewPrescriptionRepo(tx).GetByID(ctx, prescriptionID)
	if err != nil {
		return err
	}
	items, err := repository.NewPrescriptionItemRepo(tx).ListByPrescription(ctx, prescriptionID)
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
			PatientID: p.PatientID, PatientName: p.PatientName, PatientCardNo: p.PatientCardNo,
			ItemType: enum.ItemTypeDrug, ItemID: &itemID, ItemName: it.DrugName,
			Quantity: int(in.ReturnQuantity), UnitPrice: 0, Amount: -amt,
			RefType: "prescription", RefID: prescriptionID,
			OperatorID: &operatorID, OperatorName: operatorName, Remarks: "退药冲正 " + p.PrescriptionNo,
		})
	}
	return repository.NewChargeRecordRepo(tx).CreateBatch(ctx, toChargePtrs(charges))
}

// itemAmount 按处方明细快照价计算 qty（LDU）金额（统一走 money.ItemAmount，docs/13 §4.3）。
func itemAmount(it *model.PrescriptionItem, qty int64) int64 {
	return money.ItemAmount(it.IsSplit, it.IsSplitAllowed, it.PackSize, it.UnitPrice, it.RetailPrice, qty)
}

func toChargePtrs(list []model.ChargeRecord) []*model.ChargeRecord {
	out := make([]*model.ChargeRecord, 0, len(list))
	for i := range list {
		out = append(out, &list[i])
	}
	return out
}
