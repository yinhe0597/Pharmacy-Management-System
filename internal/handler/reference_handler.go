package handler

import (
	"github.com/gin-gonic/gin"

	"yaofang/internal/pkg/errs"
	"yaofang/internal/pkg/pagination"
	"yaofang/internal/service"
)

// ReferenceHandler 参考数据查询接口（ICD-10/集采/医保/耗材/非医保，只读）。
type ReferenceHandler struct {
	svc *service.ReferenceService
}

// NewReferenceHandler 构建参考数据 Handler。
func NewReferenceHandler(svc *service.ReferenceService) *ReferenceHandler {
	return &ReferenceHandler{svc: svc}
}

// Register 注册路由（只读，任意登录用户可用）。
func (h *ReferenceHandler) Register(g Groups) {
	g.Authed.GET("/reference/diagnosis-codes", h.ListDiagnosisCodes)
	g.Authed.GET("/reference/vbp-drugs", h.ListVBPDrugs)
	g.Authed.GET("/reference/nhsa-drugs", h.ListNHSADrugs)
	g.Authed.GET("/reference/consumables", h.ListMedicalConsumables)
	g.Authed.GET("/reference/non-insurance-drugs", h.ListNonInsuranceDrugs)
	g.Authed.GET("/reference/drug-match", h.MatchDrug)
}

// ListDiagnosisCodes godoc
// @Summary ICD-10 诊断编码搜索
// @Tags reference
// @Security BearerAuth
// @Param keyword query string false "关键字（编码/名称/拼音码）"
// @Param page query int false "页码"
// @Param page_size query int false "每页条数"
// @Success 200 {object} Body
// @Router /reference/diagnosis-codes [get]
func (h *ReferenceHandler) ListDiagnosisCodes(c *gin.Context) {
	var q pagination.Query
	if err := c.ShouldBindQuery(&q); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	q.Normalize()
	list, total, err := h.svc.ListDiagnosisCodes(c.Request.Context(), c.Query("keyword"), q.Page, q.PageSize)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, pagination.Of(list, total, &q))
}

// ListVBPDrugs godoc
// @Summary 国家集采药品目录搜索
// @Tags reference
// @Security BearerAuth
// @Param keyword query string false "关键字（通用名/拼音码）"
// @Param batch query int false "集采批次"
// @Param page query int false "页码"
// @Param page_size query int false "每页条数"
// @Success 200 {object} Body
// @Router /reference/vbp-drugs [get]
func (h *ReferenceHandler) ListVBPDrugs(c *gin.Context) {
	var q pagination.Query
	if err := c.ShouldBindQuery(&q); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	q.Normalize()
	list, total, err := h.svc.ListVBPDrugs(c.Request.Context(), c.Query("keyword"), atoi(c.Query("batch")), q.Page, q.PageSize)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, pagination.Of(list, total, &q))
}

// ListNHSADrugs godoc
// @Summary 国家医保药品目录搜索
// @Tags reference
// @Security BearerAuth
// @Param keyword query string false "关键字（药名/拼音码）"
// @Param insurance_class query string false "医保类别（甲类/乙类）"
// @Param page query int false "页码"
// @Param page_size query int false "每页条数"
// @Success 200 {object} Body
// @Router /reference/nhsa-drugs [get]
func (h *ReferenceHandler) ListNHSADrugs(c *gin.Context) {
	var q pagination.Query
	if err := c.ShouldBindQuery(&q); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	q.Normalize()
	list, total, err := h.svc.ListNHSADrugs(c.Request.Context(), c.Query("keyword"), c.Query("insurance_class"), q.Page, q.PageSize)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, pagination.Of(list, total, &q))
}

// ListMedicalConsumables godoc
// @Summary 医用耗材目录搜索
// @Tags reference
// @Security BearerAuth
// @Param keyword query string false "关键字（名称/拼音码/分类）"
// @Param category query string false "分类"
// @Param page query int false "页码"
// @Param page_size query int false "每页条数"
// @Success 200 {object} Body
// @Router /reference/consumables [get]
func (h *ReferenceHandler) ListMedicalConsumables(c *gin.Context) {
	var q pagination.Query
	if err := c.ShouldBindQuery(&q); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	q.Normalize()
	list, total, err := h.svc.ListMedicalConsumables(c.Request.Context(), c.Query("keyword"), c.Query("category"), q.Page, q.PageSize)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, pagination.Of(list, total, &q))
}

// ListNonInsuranceDrugs godoc
// @Summary 非医保常用药品目录搜索
// @Tags reference
// @Security BearerAuth
// @Param keyword query string false "关键字（药名/拼音码/分类）"
// @Param category query string false "分类"
// @Param page query int false "页码"
// @Param page_size query int false "每页条数"
// @Success 200 {object} Body
// @Router /reference/non-insurance-drugs [get]
func (h *ReferenceHandler) ListNonInsuranceDrugs(c *gin.Context) {
	var q pagination.Query
	if err := c.ShouldBindQuery(&q); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	q.Normalize()
	list, total, err := h.svc.ListNonInsuranceDrugs(c.Request.Context(), c.Query("keyword"), c.Query("category"), q.Page, q.PageSize)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, pagination.Of(list, total, &q))
}

// MatchDrug godoc
// @Summary 药品目录匹配（医保类别+集采批次）
// @Tags reference
// @Security BearerAuth
// @Param name query string true "药品名称"
// @Success 200 {object} Body
// @Router /reference/drug-match [get]
func (h *ReferenceHandler) MatchDrug(c *gin.Context) {
	name := c.Query("name")
	if name == "" {
		Error(c, errs.ErrBadRequest)
		return
	}
	m, err := h.svc.MatchDrug(c.Request.Context(), name)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, m)
}
