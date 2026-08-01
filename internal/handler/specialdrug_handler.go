package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"yaofang/internal/model"
	"yaofang/internal/pkg/errs"
	"yaofang/internal/pkg/pagination"
	"yaofang/internal/repository"
	"yaofang/internal/service"
)

// SpecialDrugHandler 特殊药品接口。
type SpecialDrugHandler struct {
	svc *service.SpecialDrugService
}

// NewSpecialDrugHandler 构建特殊药品 Handler。
func NewSpecialDrugHandler(svc *service.SpecialDrugService) *SpecialDrugHandler {
	return &SpecialDrugHandler{svc: svc}
}

// Register 注册路由。
func (h *SpecialDrugHandler) Register(r *gin.RouterGroup, _ *gin.RouterGroup, _ *gin.RouterGroup) {
	r.POST("/special-drugs/dispense-register", h.RegisterDispense)
	r.POST("/special-drugs/ampoule-returns", h.CreateAmpouleReturn)
	r.POST("/special-drugs/ampoule-returns/:id/verify", h.VerifyAmpouleReturn)
	r.GET("/special-drugs/ampoule-returns", h.ListAmpouleReturns)
	r.GET("/special-drugs/ledgers", h.ListLedgers)
}

// RegisterDispense godoc
// @Summary 发药专册登记（补录）
// @Tags special-drugs
// @Accept json
// @Security BearerAuth
// @Param body body model.SpecialDrugLedger true "专账记录"
// @Success 200 {object} Body
// @Router /special-drugs/dispense-register [post]
func (h *SpecialDrugHandler) RegisterDispense(c *gin.Context) {
	var l model.SpecialDrugLedger
	if err := c.ShouldBindJSON(&l); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.RegisterDispense(c.Request.Context(), &l); err != nil {
		Error(c, err)
		return
	}
	OK(c, l)
}

// CreateAmpouleReturn godoc
// @Summary 空安瓿回收登记
// @Tags special-drugs
// @Accept json
// @Security BearerAuth
// @Param body body model.AmpouleReturn true "回收记录"
// @Success 200 {object} Body
// @Router /special-drugs/ampoule-returns [post]
func (h *SpecialDrugHandler) CreateAmpouleReturn(c *gin.Context) {
	var a model.AmpouleReturn
	if err := c.ShouldBindJSON(&a); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.CreateAmpouleReturn(c.Request.Context(), &a); err != nil {
		Error(c, err)
		return
	}
	OK(c, a)
}

// VerifyAmpouleReturn godoc
// @Summary 核对空安瓿回收
// @Tags special-drugs
// @Security BearerAuth
// @Param id path int true "回收记录ID"
// @Success 200 {object} Body
// @Router /special-drugs/ampoule-returns/{id}/verify [post]
func (h *SpecialDrugHandler) VerifyAmpouleReturn(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.VerifyAmpouleReturn(c.Request.Context(), id, c.Query("verified_by")); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

// ListAmpouleReturns godoc
// @Summary 空安瓿回收记录
// @Tags special-drugs
// @Security BearerAuth
// @Param drug_id query int false "药品ID"
// @Param status query string false "状态"
// @Param page query int false "页码"
// @Param page_size query int false "每页条数"
// @Success 200 {object} Body
// @Router /special-drugs/ampoule-returns [get]
func (h *SpecialDrugHandler) ListAmpouleReturns(c *gin.Context) {
	var q pagination.Query
	if err := c.ShouldBindQuery(&q); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	q.Normalize()
	list, total, err := h.svc.ListAmpouleReturns(c.Request.Context(), int64(atoi(c.Query("drug_id"))), c.Query("status"), q.Page, q.PageSize)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, pagination.Of(list, total, &q))
}

// ListLedgers godoc
// @Summary 专账查询
// @Tags special-drugs
// @Security BearerAuth
// @Param drug_id query int false "药品ID"
// @Param log_type query string false "记录类型"
// @Param start query string false "开始时间"
// @Param end query string false "结束时间"
// @Param page query int false "页码"
// @Param page_size query int false "每页条数"
// @Success 200 {object} Body
// @Router /special-drugs/ledgers [get]
func (h *SpecialDrugHandler) ListLedgers(c *gin.Context) {
	var q pagination.Query
	if err := c.ShouldBindQuery(&q); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	q.Normalize()
	f := repository.LedgerFilter{
		DrugID:  int64(atoi(c.Query("drug_id"))),
		LogType: c.Query("log_type"),
		Start:   parseTime(c.Query("start")),
		End:     parseTime(c.Query("end")),
	}
	list, total, err := h.svc.ListLedgers(c.Request.Context(), f, q.Page, q.PageSize)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, pagination.Of(list, total, &q))
}
