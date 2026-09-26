package service

import (
	"context"
	"time"

	"gorm.io/gorm"

	"yaofang/internal/pkg/errs"
	"yaofang/internal/repository"
)

// ReportService 报表服务。
type ReportService struct {
	db *gorm.DB
}

// NewReportService 构建报表服务。
func NewReportService(db *gorm.DB) *ReportService { return &ReportService{db: db} }

// parsePeriod 校验并构造期间时间（start<=end）。
func parsePeriod(start, end *time.Time) (*time.Time, *time.Time, error) {
	if start == nil || end == nil {
		return nil, nil, errs.ErrReportPeriod
	}
	if start.After(*end) {
		return nil, nil, errs.ErrReportPeriod
	}
	return start, end, nil
}

// InventorySummary 进销存汇总。
func (s *ReportService) InventorySummary(ctx context.Context, start, end *time.Time) ([]repository.InventorySummaryRow, error) {
	st, ed, err := parsePeriod(start, end)
	if err != nil {
		return nil, err
	}
	return repository.NewReportRepo(s.db).InventorySummary(ctx, st, ed)
}

// ExpiryAnalysis 效期分析。
func (s *ReportService) ExpiryAnalysis(ctx context.Context, today time.Time) ([]repository.ExpiryBucketRow, error) {
	return repository.NewReportRepo(s.db).ExpiryAnalysis(ctx, today)
}

// SpecialDrugUsage 特殊药品使用统计。
func (s *ReportService) SpecialDrugUsage(ctx context.Context, start, end *time.Time) ([]repository.SpecialUsageRow, error) {
	st, ed, err := parsePeriod(start, end)
	if err != nil {
		return nil, err
	}
	return repository.NewReportRepo(s.db).SpecialDrugUsage(ctx, st, ed)
}

// DispensingWorkload 调配工作量统计。
func (s *ReportService) DispensingWorkload(ctx context.Context, start, end *time.Time) ([]repository.WorkloadRow, error) {
	st, ed, err := parsePeriod(start, end)
	if err != nil {
		return nil, err
	}
	return repository.NewReportRepo(s.db).DispensingWorkload(ctx, st, ed)
}

// SplitStatistics 拆零统计（拆零量/损耗）。
func (s *ReportService) SplitStatistics(ctx context.Context, start, end *time.Time) ([]repository.SplitStatRow, error) {
	st, ed, err := parsePeriod(start, end)
	if err != nil {
		return nil, err
	}
	return repository.NewReportRepo(s.db).SplitStatistics(ctx, st, ed)
}

// VisitVolume 期间按日就诊量（docs/20 S7）。
func (s *ReportService) VisitVolume(ctx context.Context, start, end *time.Time) ([]repository.VisitVolumeRow, error) {
	st, ed, err := parsePeriod(start, end)
	if err != nil {
		return nil, err
	}
	return repository.NewReportRepo(s.db).VisitVolume(ctx, st, ed)
}

// RevenueBreakdown 期间收入构成（docs/20 S7）。
func (s *ReportService) RevenueBreakdown(ctx context.Context, start, end *time.Time) ([]repository.RevenueBreakdownRow, error) {
	st, ed, err := parsePeriod(start, end)
	if err != nil {
		return nil, err
	}
	return repository.NewReportRepo(s.db).RevenueBreakdown(ctx, st, ed)
}

// DiagnosisDistribution 期间诊断分布 Top20（docs/20 S7）。
func (s *ReportService) DiagnosisDistribution(ctx context.Context, start, end *time.Time) ([]repository.DiagnosisDistributionRow, error) {
	st, ed, err := parsePeriod(start, end)
	if err != nil {
		return nil, err
	}
	return repository.NewReportRepo(s.db).DiagnosisDistribution(ctx, st, ed)
}

// PatientCharges 按患者聚合计费（docs/15 G6）。
func (s *ReportService) PatientCharges(ctx context.Context, patientID int64, start, end *time.Time) ([]repository.PatientChargeRow, error) {
	var st, ed *time.Time
	var err error
	if start != nil || end != nil {
		st, ed, err = parsePeriod(start, end)
		if err != nil {
			return nil, err
		}
	}
	return repository.NewReportRepo(s.db).PatientCharges(ctx, patientID, st, ed)
}
