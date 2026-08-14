package handler

import (
	"github.com/gin-gonic/gin"

	"yaofang/internal/service"
)

// ReportHandler 报表接口。
type ReportHandler struct {
	svc *service.ReportService
}

// NewReportHandler 构建报表 Handler。
func NewReportHandler(svc *service.ReportService) *ReportHandler { return &ReportHandler{svc: svc} }

// Register 注册路由。
func (h *ReportHandler) Register(g Groups) {
	g.Report.GET("/reports/inventory-summary", h.InventorySummary)
	g.Report.GET("/reports/expiry-analysis", h.ExpiryAnalysis)
	g.Report.GET("/reports/special-drug-usage", h.SpecialDrugUsage)
	g.Report.GET("/reports/dispensing-workload", h.DispensingWorkload)
	g.Report.GET("/reports/split-statistics", h.SplitStatistics)
	g.Report.GET("/reports/patient-charges", h.PatientCharges)
}

// InventorySummary godoc
// @Summary 进销存汇总
// @Tags reports
// @Security BearerAuth
// @Param start query string true "开始时间"
// @Param end query string true "结束时间"
// @Success 200 {object} Body
// @Router /reports/inventory-summary [get]
func (h *ReportHandler) InventorySummary(c *gin.Context) {
	start, end, err := requiredPeriod(c)
	if err != nil {
		Error(c, err)
		return
	}
	rows, err := h.svc.InventorySummary(c.Request.Context(), start, end)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, rows)
}

// ExpiryAnalysis godoc
// @Summary 效期分析
// @Tags reports
// @Security BearerAuth
// @Success 200 {object} Body
// @Router /reports/expiry-analysis [get]
func (h *ReportHandler) ExpiryAnalysis(c *gin.Context) {
	rows, err := h.svc.ExpiryAnalysis(c.Request.Context(), nowToday())
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, rows)
}

// SpecialDrugUsage godoc
// @Summary 特殊药品使用统计
// @Tags reports
// @Security BearerAuth
// @Param start query string true "开始时间"
// @Param end query string true "结束时间"
// @Success 200 {object} Body
// @Router /reports/special-drug-usage [get]
func (h *ReportHandler) SpecialDrugUsage(c *gin.Context) {
	start, end, err := requiredPeriod(c)
	if err != nil {
		Error(c, err)
		return
	}
	rows, err := h.svc.SpecialDrugUsage(c.Request.Context(), start, end)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, rows)
}

// SpecialDrugUsageAlias 特殊药品使用统计别名（兼容设计文档路径 /special-drugs/reports/usage）。
func (h *ReportHandler) SpecialDrugUsageAlias(c *gin.Context) {
	h.SpecialDrugUsage(c)
}

// DispensingWorkload godoc
// @Summary 调配工作量统计
// @Tags reports
// @Security BearerAuth
// @Param start query string true "开始时间"
// @Param end query string true "结束时间"
// @Success 200 {object} Body
// @Router /reports/dispensing-workload [get]
func (h *ReportHandler) DispensingWorkload(c *gin.Context) {
	start, end, err := requiredPeriod(c)
	if err != nil {
		Error(c, err)
		return
	}
	rows, err := h.svc.DispensingWorkload(c.Request.Context(), start, end)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, rows)
}

// SplitStatistics godoc
// @Summary 拆零统计（拆零量/损耗）
// @Tags reports
// @Security BearerAuth
// @Param start query string true "开始时间"
// @Param end query string true "结束时间"
// @Success 200 {object} Body
// @Router /reports/split-statistics [get]
func (h *ReportHandler) SplitStatistics(c *gin.Context) {
	start, end, err := requiredPeriod(c)
	if err != nil {
		Error(c, err)
		return
	}
	rows, err := h.svc.SplitStatistics(c.Request.Context(), start, end)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, rows)
}

// PatientCharges godoc
// @Summary 按患者聚合计费（docs/15 G6）
// @Tags reports
// @Security BearerAuth
// @Param patient_id query int false "患者ID（不填则全部）"
// @Param start query string false "开始时间（可选）"
// @Param end query string false "结束时间（可选）"
// @Success 200 {object} Body
// @Router /reports/patient-charges [get]
func (h *ReportHandler) PatientCharges(c *gin.Context) {
	start, end := parseTime(c.Query("start")), parseTime(c.Query("end"))
	rows, err := h.svc.PatientCharges(c.Request.Context(), int64(atoi(c.Query("patient_id"))), start, end)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, rows)
}
