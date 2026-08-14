package patient

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"yaofang/internal/model"
	"yaofang/internal/pkg/errs"
	"yaofang/internal/repository"
	"yaofang/internal/service/port"
)

// PatientService 患者服务完整实现（二期，替换 SimplePatientService）。
// 实现 port.IPatientService：患者档案 + 过敏史，供处方审核时带入个体化禁忌检查。
type PatientService struct {
	db *gorm.DB
}

// NewPatientService 构建患者服务。
func NewPatientService(db *gorm.DB) *PatientService { return &PatientService{db: db} }

// 编译期断言：完整实现必须满足 port.IPatientService 契约。
var _ port.IPatientService = (*PatientService)(nil)

// Register 注册患者（卡号唯一，已存在则返回现有 ID）。
func (s *PatientService) Register(ctx context.Context, p *port.Patient) (int64, error) {
	repo := repository.NewPatientRepo(s.db)
	if p.CardNo != "" {
		if existing, err := repo.GetByCardNo(ctx, p.CardNo); err == nil {
			return existing.ID, nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, err
		}
	}
	m := &model.Patient{
		CardNo: p.CardNo, Name: p.Name, Gender: p.Gender, Age: p.Age, Phone: p.Phone,
	}
	if err := repo.Create(ctx, m); err != nil {
		return 0, err
	}
	return m.ID, nil
}

// GetPatient 查询患者档案。
func (s *PatientService) GetPatient(ctx context.Context, patientID int64) (*port.Patient, error) {
	m, err := repository.NewPatientRepo(s.db).GetByID(ctx, patientID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return toPortPatient(m), nil
}

// GetAllergies 查询患者过敏史（审核时驱动过敏史禁忌 3010）。
func (s *PatientService) GetAllergies(ctx context.Context, patientID int64) ([]port.Allergy, error) {
	list, err := repository.NewPatientRepo(s.db).ListAllergies(ctx, patientID)
	if err != nil {
		return nil, err
	}
	out := make([]port.Allergy, 0, len(list))
	for _, a := range list {
		out = append(out, port.Allergy{DrugName: a.DrugName, Reaction: a.Reaction, Severity: a.Severity})
	}
	return out, nil
}

// GetMedicationHistory 查询患者用药史（docs/15 L6：已发药/已退药处方明细）。
func (s *PatientService) GetMedicationHistory(ctx context.Context, patientID int64) ([]port.MedicationRecord, error) {
	rows, err := repository.NewPatientRepo(s.db).MedicationHistory(ctx, patientID)
	if err != nil {
		return nil, err
	}
	out := make([]port.MedicationRecord, 0, len(rows))
	for _, r := range rows {
		out = append(out, port.MedicationRecord{
			DrugName:  r.DrugName,
			Dosage:    r.Dosage,
			BeginDate: r.BeginDate.Format("2006-01-02"),
		})
	}
	return out, nil
}

// ---- 档案管理（HTTP 层 CRUD，返回 model 类型） ----

// List 分页查询患者档案（姓名/卡号/电话模糊搜索）。
func (s *PatientService) List(ctx context.Context, keyword string, page, pageSize int) ([]model.Patient, int64, error) {
	return repository.NewPatientRepo(s.db).List(ctx, keyword, (page-1)*pageSize, pageSize)
}

// GetDetail 查询患者详情（未找到返回 404，docs/15 M1）。
func (s *PatientService) GetDetail(ctx context.Context, id int64) (*model.Patient, error) {
	p, err := repository.NewPatientRepo(s.db).GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrPatientNotFound
		}
		return nil, err
	}
	return p, nil
}

// Create 新建患者档案（重复卡号返回 409，docs/15 M1）。
func (s *PatientService) Create(ctx context.Context, p *model.Patient) error {
	if p.CardNo != "" {
		if existing, err := repository.NewPatientRepo(s.db).GetByCardNo(ctx, p.CardNo); err == nil {
			_ = existing
			return errs.ErrPatientCardExists
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}
	return repository.NewPatientRepo(s.db).Create(ctx, p)
}

// Update 更新患者档案（未找到返回 404，docs/15 M1）。
func (s *PatientService) Update(ctx context.Context, id int64, p *model.Patient) error {
	repo := repository.NewPatientRepo(s.db)
	existing, err := repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.ErrPatientNotFound
		}
		return err
	}
	p.ID = existing.ID
	p.CreatedAt = existing.CreatedAt
	return repo.Update(ctx, p)
}

// ListAllergiesDetail 查询患者过敏史（model 类型）。
func (s *PatientService) ListAllergiesDetail(ctx context.Context, patientID int64) ([]model.PatientAllergy, error) {
	return repository.NewPatientRepo(s.db).ListAllergies(ctx, patientID)
}

// AddAllergy 新增过敏记录（severity 限 1-3，docs/15 M5）。
func (s *PatientService) AddAllergy(ctx context.Context, a *model.PatientAllergy) error {
	if a.PatientID <= 0 || a.DrugName == "" {
		return errs.ErrBadRequest
	}
	if a.Severity < 1 || a.Severity > 3 {
		return errs.ErrBadRequest
	}
	return repository.NewPatientRepo(s.db).CreateAllergy(ctx, a)
}

// DeleteAllergy 删除过敏记录。
func (s *PatientService) DeleteAllergy(ctx context.Context, id int64) error {
	return repository.NewPatientRepo(s.db).DeleteAllergy(ctx, id)
}

func toPortPatient(m *model.Patient) *port.Patient {
	if m == nil {
		return nil
	}
	return &port.Patient{
		ID: m.ID, Name: m.Name, Gender: m.Gender, Age: m.Age, CardNo: m.CardNo,
		Phone: m.Phone, IsLactating: m.IsLactating, CreatedAt: m.CreatedAt,
	}
}
