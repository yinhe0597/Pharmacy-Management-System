package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"yaofang/internal/middleware"
	"yaofang/internal/model"
	"yaofang/internal/pkg/errs"
	"yaofang/internal/pkg/pagination"
	"yaofang/internal/service"
)

// ClinicalHandler 诊疗项目与计费接口（护士/医生/药师/管理员可操作）。
type ClinicalHandler struct {
	svc *service.ClinicalService
}

func NewClinicalHandler(svc *service.ClinicalService) *ClinicalHandler {
	return &ClinicalHandler{svc: svc}
}

func (h *ClinicalHandler) Register(g Groups) {
	// 诊疗项目目录（目录维护为药房专业角色；录入计费为药房工作人员）
	g.Authed.GET("/clinical-services", h.ListServices)
	g.DrugAdmin.POST("/clinical-services", h.CreateService)
	g.DrugAdmin.PUT("/clinical-services/:id", h.UpdateService)
	g.DrugAdmin.DELETE("/clinical-services/:id", h.DeleteService)
	// 计费记录（护士/医生/药师/管理员可录入）
	g.Authed.GET("/charge-records", h.ListCharges)
	g.Pharmacy.POST("/charge-records", h.CreateCharge)
	g.Pharmacy.POST("/charge-records/from-prescription/:id", h.ChargePrescription)
}

// ---- 诊疗项目 ----

// ListServices godoc
// @Summary 诊疗项目列表
// @Tags clinical-services
// @Security BearerAuth
// @Param keyword query string false "关键字"
// @Param page query int false "页码"
// @Param page_size query int false "每页条数"
// @Success 200 {object} Body
// @Router /clinical-services [get]
func (h *ClinicalHandler) ListServices(c *gin.Context) {
	var q pagination.Query
	if err := c.ShouldBindQuery(&q); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	q.Normalize()
	list, total, err := h.svc.ListServices(c.Request.Context(), c.Query("keyword"), q.Page, q.PageSize)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, pagination.Of(list, total, &q))
}

// CreateService godoc
// @Summary 新建诊疗项目
// @Tags clinical-services
// @Accept json
// @Security BearerAuth
// @Param body body model.ClinicalService true "诊疗项目"
// @Success 200 {object} Body
// @Router /clinical-services [post]
func (h *ClinicalHandler) CreateService(c *gin.Context) {
	var v model.ClinicalService
	if err := c.ShouldBindJSON(&v); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.CreateService(c.Request.Context(), &v); err != nil {
		Error(c, err)
		return
	}
	OK(c, v)
}

// UpdateService godoc
// @Summary 更新诊疗项目
// @Tags clinical-services
// @Accept json
// @Security BearerAuth
// @Param id path int true "项目ID"
// @Param body body model.ClinicalService true "诊疗项目"
// @Success 200 {object} Body
// @Router /clinical-services/{id} [put]
func (h *ClinicalHandler) UpdateService(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	var v model.ClinicalService
	if err := c.ShouldBindJSON(&v); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.UpdateService(c.Request.Context(), id, &v); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

// DeleteService godoc
// @Summary 删除诊疗项目
// @Tags clinical-services
// @Security BearerAuth
// @Param id path int true "项目ID"
// @Success 200 {object} Body
// @Router /clinical-services/{id} [delete]
func (h *ClinicalHandler) DeleteService(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.DeleteService(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

// ---- 计费记录 ----

// ListCharges godoc
// @Summary 计费记录列表
// @Tags charge-records
// @Security BearerAuth
// @Param keyword query string false "患者姓名/项目名"
// @Param page query int false "页码"
// @Param page_size query int false "每页条数"
// @Success 200 {object} Body
// @Router /charge-records [get]
func (h *ClinicalHandler) ListCharges(c *gin.Context) {
	var q pagination.Query
	if err := c.ShouldBindQuery(&q); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	q.Normalize()
	list, total, err := h.svc.ListCharges(c.Request.Context(), c.Query("keyword"), q.Page, q.PageSize)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, pagination.Of(list, total, &q))
}

// CreateCharge godoc
// @Summary 新建计费记录（药品/耗材/诊疗项目）
// @Tags charge-records
// @Accept json
// @Security BearerAuth
// @Param body body service.ChargeInput true "计费信息"
// @Success 200 {object} Body
// @Router /charge-records [post]
func (h *ClinicalHandler) CreateCharge(c *gin.Context) {
	var req service.ChargeInput
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	cr, err := h.svc.CreateCharge(c.Request.Context(), req, middleware.UserIDFromCtx(c), middleware.UserNameFromCtx(c))
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, cr)
}

// ChargePrescription godoc
// @Summary 从已发药处方生成计费记录（幂等）
// @Tags charge-records
// @Security BearerAuth
// @Param id path int true "处方ID"
// @Success 200 {object} Body
// @Router /charge-records/from-prescription/{id} [post]
func (h *ClinicalHandler) ChargePrescription(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	list, err := h.svc.ChargePrescription(c.Request.Context(), id, middleware.UserIDFromCtx(c), middleware.UserNameFromCtx(c))
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, list)
}
