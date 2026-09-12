package service

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"yaofang/internal/domain/enum"
	"yaofang/internal/model"
	"yaofang/internal/pkg/errs"
	"yaofang/internal/repository"
)

// 麻精药品单张处方限量（天数），常量可配置。
const (
	specialLimitNarcoticDays  = 3 // 麻醉 ≤3 日量
	specialLimitPsychoOneDays = 3 // 一类精神 ≤3 日量
	specialLimitPsychoTwoDays = 7 // 二类精神 ≤7 日量
)

// SpecialDrugService 特殊药品服务（麻精五专、空安瓿回收、专账）。
type SpecialDrugService struct {
	db *gorm.DB
}

// NewSpecialDrugService 构建特殊药品服务。
func NewSpecialDrugService(db *gorm.DB) *SpecialDrugService { return &SpecialDrugService{db: db} }

// CheckPrescriptionLimit 麻精处方单张剂量限量。
func (s *SpecialDrugService) CheckPrescriptionLimit(_ context.Context, prescType int, items []PrescriptionItemInput) error {
	limit := 0
	switch prescType {
	case enum.PrescriptionTypeNarcotic:
		limit = specialLimitNarcoticDays
	case enum.PrescriptionTypePsychoOne:
		limit = specialLimitPsychoOneDays
	case enum.PrescriptionTypePsychoTwo:
		limit = specialLimitPsychoTwoDays
	default:
		return nil // 毒性/放射性/普通不限
	}
	for _, it := range items {
		if it.Days > limit {
			return errs.ErrSpecialDrugLimit
		}
	}
	return nil
}

// WriteDispenseLedgerTx 发药专账登记（在处方发药事务内调用）。
func (s *SpecialDrugService) WriteDispenseLedgerTx(ctx context.Context, tx *gorm.DB, p *model.Prescription) error {
	items, err := repository.NewPrescriptionItemRepo(tx).ListByPrescription(ctx, p.ID)
	if err != nil {
		return err
	}
	// 取发药记录以获得批次
	records, err := repository.NewDispenseRecordRepo(tx).ListByPrescription(ctx, p.ID)
	if err != nil {
		return err
	}
	recByItem := make(map[int64][]model.PrescriptionDispenseRecord)
	for _, r := range records {
		recByItem[r.ItemID] = append(recByItem[r.ItemID], r)
	}
	ledgerRepo := repository.NewSpecialDrugLedgerRepo(tx)
	for _, it := range items {
		for _, rec := range recByItem[it.ID] {
			if err := ledgerRepo.Create(ctx, &model.SpecialDrugLedger{
				DrugID: it.DrugID, BatchNo: rec.BatchNo, PrescriptionID: p.ID,
				LogType: "dispense", Quantity: rec.Quantity,
				PatientName: p.PatientName, PatientCardNo: p.PatientCardNo,
				OperatorName: p.CheckerName, Notes: "处方发药",
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

// CreateAmpouleReturn 空安瓿回收登记。
func (s *SpecialDrugService) CreateAmpouleReturn(ctx context.Context, a *model.AmpouleReturn) error {
	if a.DrugID <= 0 || a.Quantity <= 0 {
		return errs.ErrBadRequest
	}
	if a.ReturnDate.IsZero() {
		a.ReturnDate = time.Now()
	}
	a.Status = "pending"
	return repository.NewAmpouleReturnRepo(s.db).Create(ctx, a)
}

// VerifyAmpouleReturn 核对空安瓿回收（与发药数联动由药房线下执行，系统登记核对人）。
// 核对人落库（verified_by），并保证幂等：仅 pending → verified 一次，重复核对返回状态冲突。
func (s *SpecialDrugService) VerifyAmpouleReturn(ctx context.Context, id int64, verifiedBy string) error {
	repo := repository.NewAmpouleReturnRepo(s.db)
	if _, err := repo.GetByID(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.ErrNotFound
		}
		return err
	}
	ok, err := repo.Verify(ctx, id, verifiedBy)
	if err != nil {
		return err
	}
	if !ok {
		return errs.ErrStateConflict
	}
	return nil
}

// ListAmpouleReturns 分页查询空安瓿回收。
func (s *SpecialDrugService) ListAmpouleReturns(ctx context.Context, drugID int64, status string, page, pageSize int) ([]model.AmpouleReturn, int64, error) {
	return repository.NewAmpouleReturnRepo(s.db).List(ctx, drugID, status, (page-1)*pageSize, pageSize)
}

// ListLedgers 分页查询专账。
func (s *SpecialDrugService) ListLedgers(ctx context.Context, f repository.LedgerFilter, page, pageSize int) ([]model.SpecialDrugLedger, int64, error) {
	return repository.NewSpecialDrugLedgerRepo(s.db).List(ctx, f, (page-1)*pageSize, pageSize)
}

// RegisterDispense 发药专册登记（独立入口，供手工补录）。
func (s *SpecialDrugService) RegisterDispense(ctx context.Context, l *model.SpecialDrugLedger) error {
	if l.DrugID <= 0 || l.BatchNo == "" || l.Quantity == 0 {
		return errs.ErrBadRequest
	}
	if l.LogType == "" {
		l.LogType = "dispense"
	}
	l.ID = 0
	return repository.NewSpecialDrugLedgerRepo(s.db).Create(ctx, l)
}

// ListSpecialPrescriptions 查询特殊药品处方列表（麻精/毒性/放射性）。
// 特殊处方类型：麻醉(1)/精神一类(2)/精神二类(3)/毒性(4)/放射性(5) → prescription_type > 0。
func (s *SpecialDrugService) ListSpecialPrescriptions(ctx context.Context, status string, keyword string, page, pageSize int) ([]model.Prescription, int64, error) {
	repo := repository.NewPrescriptionRepo(s.db)
	f := repository.PrescriptionFilter{
		Status:      status,
		SpecialOnly: true, // 仅特殊类型（prescription_type > 0）
	}
	if keyword != "" {
		f.PatientName = keyword
	}
	return repo.List(ctx, f, (page-1)*pageSize, pageSize)
}
