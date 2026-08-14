package service

import (
	"context"

	"gorm.io/gorm"

	"yaofang/internal/model"
	"yaofang/internal/repository"
)

// ReferenceService 参考数据查询服务（v1.3：ICD-10/集采/医保/耗材/非医保，只读）。
type ReferenceService struct {
	db *gorm.DB
}

// NewReferenceService 创建参考数据服务。
func NewReferenceService(db *gorm.DB) *ReferenceService { return &ReferenceService{db: db} }

func (s *ReferenceService) ListDiagnosisCodes(ctx context.Context, keyword string, page, pageSize int) ([]model.DiagnosisCode, int64, error) {
	return repository.NewReferenceRepo(s.db).ListDiagnosisCodes(ctx, keyword, (page-1)*pageSize, pageSize)
}

func (s *ReferenceService) ListVBPDrugs(ctx context.Context, keyword string, batch, page, pageSize int) ([]model.VBPDrug, int64, error) {
	return repository.NewReferenceRepo(s.db).ListVBPDrugs(ctx, keyword, batch, (page-1)*pageSize, pageSize)
}

func (s *ReferenceService) ListNHSADrugs(ctx context.Context, keyword, insuranceClass string, page, pageSize int) ([]model.NHSADrug, int64, error) {
	return repository.NewReferenceRepo(s.db).ListNHSADrugs(ctx, keyword, insuranceClass, (page-1)*pageSize, pageSize)
}

func (s *ReferenceService) ListMedicalConsumables(ctx context.Context, keyword, category string, page, pageSize int) ([]model.MedicalConsumable, int64, error) {
	return repository.NewReferenceRepo(s.db).ListMedicalConsumables(ctx, keyword, category, (page-1)*pageSize, pageSize)
}

func (s *ReferenceService) ListNonInsuranceDrugs(ctx context.Context, keyword, category string, page, pageSize int) ([]model.NonInsuranceDrug, int64, error) {
	return repository.NewReferenceRepo(s.db).ListNonInsuranceDrugs(ctx, keyword, category, (page-1)*pageSize, pageSize)
}

// DrugMatch 药品目录匹配结果（医保类别 + 集采批次）。
type DrugMatch struct {
	Name           string `json:"name"`
	InsuranceClass string `json:"insurance_class"` // 甲类/乙类/空
	VPBBatch       int    `json:"vbp_batch"`       // 集采批次/0
}

// MatchDrug 按药品名称精确匹配医保/集采目录，供药品录入时自动标注。
func (s *ReferenceService) MatchDrug(ctx context.Context, name string) (*DrugMatch, error) {
	repo := repository.NewReferenceRepo(s.db)
	out := &DrugMatch{Name: name}
	if nhsa, err := repo.FindNHSAByName(ctx, name); err == nil {
		out.InsuranceClass = nhsa.InsuranceClass
	}
	if vbp, err := repo.FindVBPByName(ctx, name); err == nil {
		out.VPBBatch = vbp.VPBBatch
	}
	return out, nil
}
