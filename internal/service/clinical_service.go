package service

import (
	"context"

	"gorm.io/gorm"

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
