package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strconv"
	"time"

	"yaofang/internal/pkg/errs"
	"yaofang/internal/repository"
)

// 报表导出名常量（与 GET /reports/export?name= 取值一致）。
const (
	ExportInventorySummary      = "inventory-summary"
	ExportExpiryAnalysis        = "expiry-analysis"
	ExportSpecialDrugUsage      = "special-drug-usage"
	ExportDispensingWorkload    = "dispensing-workload"
	ExportSplitStatistics       = "split-statistics"
	ExportPatientCharges        = "patient-charges"
	ExportVisitVolume           = "visit-volume"
	ExportRevenueBreakdown      = "revenue-breakdown"
	ExportDiagnosisDistribution = "diagnosis-distribution"
)

// yuan 输出用：分 → 元（保留两位，展示口径）.
func yuan(v int64) string { return fmt.Sprintf("%.2f", float64(v)/100) }

// encodeCSV 纯函数：表头 + 行 → 带 BOM 的 CSV 字节（Excel 直接打开不乱码）。
func encodeCSV(headers []string, records [][]string) []byte {
	var buf bytes.Buffer
	buf.Write([]byte{0xEF, 0xBB, 0xBF}) // UTF-8 BOM
	w := csv.NewWriter(&buf)
	_ = w.Write(headers)
	_ = w.WriteAll(records)
	w.Flush()
	return buf.Bytes()
}

// csvBuilder 单表构建器：返回文件名、表头、数据行。
type csvBuilder func(ctx context.Context) (filename string, headers []string, records [][]string, err error)

// ExportCSV 导出指定报表为 CSV。period 校验复用各报表服务方法（口径一致）。
func (s *ReportService) ExportCSV(ctx context.Context, name string, patientID int64, start, end *time.Time, today time.Time) (string, []byte, error) {
	builders := map[string]csvBuilder{
		ExportInventorySummary: func(ctx context.Context) (string, []string, [][]string, error) {
			return s.exportInventoryRows(ctx, start, end)
		},
		ExportExpiryAnalysis: func(ctx context.Context) (string, []string, [][]string, error) { return s.exportExpiryRows(ctx, today) },
		ExportSpecialDrugUsage: func(ctx context.Context) (string, []string, [][]string, error) {
			return s.exportSpecialRows(ctx, start, end)
		},
		ExportDispensingWorkload: func(ctx context.Context) (string, []string, [][]string, error) {
			return s.exportWorkloadRows(ctx, start, end)
		},
		ExportSplitStatistics: func(ctx context.Context) (string, []string, [][]string, error) {
			return s.exportSplitRows(ctx, start, end)
		},
		ExportPatientCharges: func(ctx context.Context) (string, []string, [][]string, error) {
			return s.exportPatientRows(ctx, patientID, start, end)
		},
		ExportVisitVolume: func(ctx context.Context) (string, []string, [][]string, error) {
			return s.exportVisitRows(ctx, start, end)
		},
		ExportRevenueBreakdown: func(ctx context.Context) (string, []string, [][]string, error) {
			return s.exportRevenueRows(ctx, start, end)
		},
		ExportDiagnosisDistribution: func(ctx context.Context) (string, []string, [][]string, error) {
			return s.exportDiagnosisRows(ctx, start, end)
		},
	}
	build, ok := builders[name]
	if !ok {
		return "", nil, errs.ErrBadRequest
	}
	filename, headers, records, err := build(ctx)
	if err != nil {
		return "", nil, err
	}
	return filename, encodeCSV(headers, records), nil
}

func (s *ReportService) exportInventoryRows(ctx context.Context, start, end *time.Time) (string, []string, [][]string, error) {
	st, ed, err := parsePeriod(start, end)
	if err != nil {
		return "", nil, nil, err
	}
	rows, err := repository.NewReportRepo(s.db).InventorySummary(ctx, st, ed)
	if err != nil {
		return "", nil, nil, err
	}
	recs := make([][]string, 0, len(rows))
	for _, r := range rows {
		recs = append(recs, []string{r.DrugName, strconv.FormatInt(r.DrugID, 10),
			strconv.FormatInt(r.Opening, 10), strconv.FormatInt(r.Inbound, 10),
			strconv.FormatInt(r.Outbound, 10), strconv.FormatInt(r.Closing, 10)})
	}
	return "inventory-summary.csv", []string{"药品", "药品ID", "期初", "入库", "出库", "期末"}, recs, nil
}

func (s *ReportService) exportExpiryRows(ctx context.Context, today time.Time) (string, []string, [][]string, error) {
	rows, err := repository.NewReportRepo(s.db).ExpiryAnalysis(ctx, today)
	if err != nil {
		return "", nil, nil, err
	}
	recs := make([][]string, 0, len(rows))
	for _, r := range rows {
		recs = append(recs, []string{r.DrugName, r.BatchNo,
			strconv.FormatInt(r.Expired, 10), strconv.FormatInt(r.In3M, 10),
			strconv.FormatInt(r.In6M, 10), strconv.FormatInt(r.In12M, 10),
			strconv.FormatInt(r.After12M, 10)})
	}
	return "expiry-analysis.csv", []string{"药品", "批号", "已过期", "3月内", "6月内", "12月内", "12月以上"}, recs, nil
}

func (s *ReportService) exportSpecialRows(ctx context.Context, start, end *time.Time) (string, []string, [][]string, error) {
	rows, err := s.SpecialDrugUsage(ctx, start, end)
	if err != nil {
		return "", nil, nil, err
	}
	recs := make([][]string, 0, len(rows))
	for _, r := range rows {
		recs = append(recs, []string{r.DrugName, r.BatchNo, r.Month,
			strconv.FormatInt(r.DispenseQty, 10), strconv.FormatInt(r.ReturnQty, 10)})
	}
	return "special-drug-usage.csv", []string{"药品", "批号", "月份", "发药", "退回"}, recs, nil
}

func (s *ReportService) exportWorkloadRows(ctx context.Context, start, end *time.Time) (string, []string, [][]string, error) {
	rows, err := s.DispensingWorkload(ctx, start, end)
	if err != nil {
		return "", nil, nil, err
	}
	recs := make([][]string, 0, len(rows))
	for _, r := range rows {
		recs = append(recs, []string{r.DispensedName, r.Date,
			strconv.FormatInt(r.PrescriptionCount, 10), strconv.FormatInt(r.ItemCount, 10)})
	}
	return "dispensing-workload.csv", []string{"发药人", "日期", "处方数", "件数"}, recs, nil
}

func (s *ReportService) exportSplitRows(ctx context.Context, start, end *time.Time) (string, []string, [][]string, error) {
	rows, err := s.SplitStatistics(ctx, start, end)
	if err != nil {
		return "", nil, nil, err
	}
	recs := make([][]string, 0, len(rows))
	for _, r := range rows {
		recs = append(recs, []string{r.DrugName,
			strconv.FormatInt(r.SplitBoxes, 10), strconv.FormatInt(r.SplitUnits, 10),
			strconv.FormatInt(r.LossUnits, 10), yuan(r.SplitCost), yuan(r.SplitRevenue), yuan(r.Margin)})
	}
	return "split-statistics.csv", []string{"药品", "拆零盒数", "入片数", "损耗", "成本(元)", "收入(元)", "毛利(元)"}, recs, nil
}

func (s *ReportService) exportPatientRows(ctx context.Context, patientID int64, start, end *time.Time) (string, []string, [][]string, error) {
	rows, err := s.PatientCharges(ctx, patientID, start, end)
	if err != nil {
		return "", nil, nil, err
	}
	recs := make([][]string, 0, len(rows))
	for _, r := range rows {
		recs = append(recs, []string{r.PatientName, r.ItemType,
			strconv.FormatInt(r.ChargeCount, 10), strconv.FormatInt(r.RefundCount, 10), yuan(r.Amount)})
	}
	return "patient-charges.csv", []string{"患者", "类型", "收费笔数", "冲正笔数", "净额(元)"}, recs, nil
}

func (s *ReportService) exportVisitRows(ctx context.Context, start, end *time.Time) (string, []string, [][]string, error) {
	rows, err := s.VisitVolume(ctx, start, end)
	if err != nil {
		return "", nil, nil, err
	}
	recs := make([][]string, 0, len(rows))
	for _, r := range rows {
		recs = append(recs, []string{r.Date,
			strconv.FormatInt(r.VisitCount, 10), strconv.FormatInt(r.FinishedCount, 10),
			strconv.FormatInt(r.CancelledCount, 10)})
	}
	return "visit-volume.csv", []string{"日期", "挂号数", "已结束", "退号"}, recs, nil
}

func (s *ReportService) exportRevenueRows(ctx context.Context, start, end *time.Time) (string, []string, [][]string, error) {
	rows, err := s.RevenueBreakdown(ctx, start, end)
	if err != nil {
		return "", nil, nil, err
	}
	recs := make([][]string, 0, len(rows))
	for _, r := range rows {
		recs = append(recs, []string{r.ItemType,
			strconv.FormatInt(r.ChargeCount, 10), strconv.FormatInt(r.Quantity, 10), yuan(r.Amount)})
	}
	return "revenue-breakdown.csv", []string{"费用类型", "结算单数", "数量", "金额(元)"}, recs, nil
}

func (s *ReportService) exportDiagnosisRows(ctx context.Context, start, end *time.Time) (string, []string, [][]string, error) {
	rows, err := s.DiagnosisDistribution(ctx, start, end)
	if err != nil {
		return "", nil, nil, err
	}
	recs := make([][]string, 0, len(rows))
	for _, r := range rows {
		recs = append(recs, []string{r.DiagnosisCode, r.DiagnosisName, strconv.FormatInt(r.UseCount, 10)})
	}
	return "diagnosis-distribution.csv", []string{"诊断编码", "诊断名称", "使用次数"}, recs, nil
}
