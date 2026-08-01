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
func (h *ReportHandler) Register(r *gin.RouterGroup, _ *gin.RouterGroup, _ *gin.RouterGroup) {
	r.GET("/reports/inventory-summary", h.InventorySummary)
	r.GET("/reports/expiry-analysis", h.ExpiryAnalysis)
	r.GET("/reports/special-drug-usage", h.SpecialDrugUsage)
	r.GET("/reports/dispensing-workload", h.DispensingWorkload)
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
