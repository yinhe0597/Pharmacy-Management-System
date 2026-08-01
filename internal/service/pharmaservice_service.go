package service

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"yaofang/internal/model"
	"yaofang/internal/pkg/errs"
	"yaofang/internal/repository"
)

// PharmaService 药学服务：用药咨询、不良反应、用药指导。
type PharmaService struct {
	db *gorm.DB
}

// NewPharmaService 构建药学服务。
func NewPharmaService(db *gorm.DB) *PharmaService { return &PharmaService{db: db} }

// ---- 用药咨询 ----

// CreateConsultation 新建咨询。
func (s *PharmaService) CreateConsultation(ctx context.Context, c *model.Consultation) error {
	if c.Question == "" {
		return errs.ErrBadRequest
	}
	if c.ConsultedAt.IsZero() {
		c.ConsultedAt = time.Now()
	}
	return repository.NewConsultationRepo(s.db).Create(ctx, c)
}

// UpdateConsultation 更新咨询。
func (s *PharmaService) UpdateConsultation(ctx context.Context, id int64, c *model.Consultation) error {
	existing, err := repository.NewConsultationRepo(s.db).GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.ErrNotFound
		}
		return err
	}
	c.ID = existing.ID
	c.CreatedAt = existing.CreatedAt
	return repository.NewConsultationRepo(s.db).Update(ctx, c)
}

// DeleteConsultation 删除咨询。
func (s *PharmaService) DeleteConsultation(ctx context.Context, id int64) error {
	return repository.NewConsultationRepo(s.db).Delete(ctx, id)
}

// ListConsultations 分页查询咨询。
func (s *PharmaService) ListConsultations(ctx context.Context, keyword string, page, pageSize int) ([]model.Consultation, int64, error) {
	return repository.NewConsultationRepo(s.db).List(ctx, keyword, (page-1)*pageSize, pageSize)
}

// ---- 不良反应 ----

// CreateAdverseReaction 新建登记。
func (s *PharmaService) CreateAdverseReaction(ctx context.Context, a *model.AdverseReaction) error {
	if a.ReactionDesc == "" {
		return errs.ErrBadRequest
	}
	if a.ReportDate.IsZero() {
		a.ReportDate = time.Now()
	}
	return repository.NewAdverseReactionRepo(s.db).Create(ctx, a)
}

// UpdateAdverseReaction 更新登记。
func (s *PharmaService) UpdateAdverseReaction(ctx context.Context, id int64, a *model.AdverseReaction) error {
	existing, err := repository.NewAdverseReactionRepo(s.db).GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.ErrNotFound
		}
		return err
	}
	a.ID = existing.ID
	a.CreatedAt = existing.CreatedAt
	return repository.NewAdverseReactionRepo(s.db).Update(ctx, a)
}

// DeleteAdverseReaction 删除登记。
func (s *PharmaService) DeleteAdverseReaction(ctx context.Context, id int64) error {
	return repository.NewAdverseReactionRepo(s.db).Delete(ctx, id)
}

// ListAdverseReactions 分页查询登记。
func (s *PharmaService) ListAdverseReactions(ctx context.Context, drugID int64, keyword string, page, pageSize int) ([]model.AdverseReaction, int64, error) {
	return repository.NewAdverseReactionRepo(s.db).List(ctx, drugID, keyword, (page-1)*pageSize, pageSize)
}

// ---- 用药指导 ----

// CreateGuidance 新建指导。
func (s *PharmaService) CreateGuidance(ctx context.Context, m *model.MedicationGuidance) error {
	if m.Content == "" {
		return errs.ErrBadRequest
	}
	if m.GuidedAt.IsZero() {
		m.GuidedAt = time.Now()
	}
	return repository.NewMedicationGuidanceRepo(s.db).Create(ctx, m)
}

// ListGuidances 分页查询指导。
func (s *PharmaService) ListGuidances(ctx context.Context, keyword string, page, pageSize int) ([]model.MedicationGuidance, int64, error) {
	return repository.NewMedicationGuidanceRepo(s.db).List(ctx, keyword, (page-1)*pageSize, pageSize)
}

// UpdateGuidance 更新指导。
func (s *PharmaService) UpdateGuidance(ctx context.Context, id int64, m *model.MedicationGuidance) error {
	_, err := repository.NewMedicationGuidanceRepo(s.db).GetByID(ctx, id)
	if err != nil {
		return errs.ErrNotFound
	}
	m.ID = id
	return repository.NewMedicationGuidanceRepo(s.db).Update(ctx, m)
}

// DeleteGuidance 删除指导。
func (s *PharmaService) DeleteGuidance(ctx context.Context, id int64) error {
	return repository.NewMedicationGuidanceRepo(s.db).Delete(ctx, id)
}
