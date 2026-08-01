package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"yaofang/internal/model"
	"yaofang/internal/pkg/errs"
	"yaofang/internal/pkg/pagination"
	"yaofang/internal/service"
)

// PharmaServiceHandler 药学服务接口。
type PharmaServiceHandler struct {
	svc *service.PharmaService
}

// NewPharmaServiceHandler 构建药学服务 Handler。
func NewPharmaServiceHandler(svc *service.PharmaService) *PharmaServiceHandler {
	return &PharmaServiceHandler{svc: svc}
}

// Register 注册路由。
func (h *PharmaServiceHandler) Register(r *gin.RouterGroup, _ *gin.RouterGroup, _ *gin.RouterGroup) {
	r.GET("/consultations", h.ListConsultations)
	r.POST("/consultations", h.CreateConsultation)
	r.PUT("/consultations/:id", h.UpdateConsultation)
	r.DELETE("/consultations/:id", h.DeleteConsultation)
	r.GET("/adverse-reactions", h.ListAdverseReactions)
	r.POST("/adverse-reactions", h.CreateAdverseReaction)
	r.PUT("/adverse-reactions/:id", h.UpdateAdverseReaction)
	r.DELETE("/adverse-reactions/:id", h.DeleteAdverseReaction)
	r.GET("/medication-guidances", h.ListGuidances)
	r.POST("/medication-guidances", h.CreateGuidance)
	r.PUT("/medication-guidances/:id", h.UpdateGuidance)
	r.DELETE("/medication-guidances/:id", h.DeleteGuidance)
}

// ListConsultations godoc
// @Summary 用药咨询列表
// @Tags consultations
// @Security BearerAuth
// @Param keyword query string false "关键字"
// @Param page query int false "页码"
// @Param page_size query int false "每页条数"
// @Success 200 {object} Body
// @Router /consultations [get]
func (h *PharmaServiceHandler) ListConsultations(c *gin.Context) {
	var q pagination.Query
	if err := c.ShouldBindQuery(&q); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	q.Normalize()
	list, total, err := h.svc.ListConsultations(c.Request.Context(), c.Query("keyword"), q.Page, q.PageSize)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, pagination.Of(list, total, &q))
}

// CreateConsultation godoc
// @Summary 新建用药咨询
// @Tags consultations
// @Accept json
// @Security BearerAuth
// @Param body body model.Consultation true "咨询记录"
// @Success 200 {object} Body
// @Router /consultations [post]
func (h *PharmaServiceHandler) CreateConsultation(c *gin.Context) {
	var v model.Consultation
	if err := c.ShouldBindJSON(&v); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.CreateConsultation(c.Request.Context(), &v); err != nil {
		Error(c, err)
		return
	}
	OK(c, v)
}

// UpdateConsultation godoc
// @Summary 更新用药咨询
// @Tags consultations
// @Security BearerAuth
// @Param id path int true "咨询ID"
// @Param body body model.Consultation true "咨询记录"
// @Success 200 {object} Body
// @Router /consultations/{id} [put]
func (h *PharmaServiceHandler) UpdateConsultation(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	var v model.Consultation
	if err := c.ShouldBindJSON(&v); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.UpdateConsultation(c.Request.Context(), id, &v); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

// DeleteConsultation godoc
// @Summary 删除用药咨询
// @Tags consultations
// @Security BearerAuth
// @Param id path int true "咨询ID"
// @Success 200 {object} Body
// @Router /consultations/{id} [delete]
func (h *PharmaServiceHandler) DeleteConsultation(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.DeleteConsultation(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

// ListAdverseReactions godoc
// @Summary 不良反应列表
// @Tags adverse-reactions
// @Security BearerAuth
// @Param drug_id query int false "药品ID"
// @Param keyword query string false "关键字"
// @Param page query int false "页码"
// @Param page_size query int false "每页条数"
// @Success 200 {object} Body
// @Router /adverse-reactions [get]
func (h *PharmaServiceHandler) ListAdverseReactions(c *gin.Context) {
	var q pagination.Query
	if err := c.ShouldBindQuery(&q); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	q.Normalize()
	list, total, err := h.svc.ListAdverseReactions(c.Request.Context(), int64(atoi(c.Query("drug_id"))), c.Query("keyword"), q.Page, q.PageSize)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, pagination.Of(list, total, &q))
}

// CreateAdverseReaction godoc
// @Summary 新建不良反应登记
// @Tags adverse-reactions
// @Accept json
// @Security BearerAuth
// @Param body body model.AdverseReaction true "登记记录"
// @Success 200 {object} Body
// @Router /adverse-reactions [post]
func (h *PharmaServiceHandler) CreateAdverseReaction(c *gin.Context) {
	var v model.AdverseReaction
	if err := c.ShouldBindJSON(&v); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.CreateAdverseReaction(c.Request.Context(), &v); err != nil {
		Error(c, err)
		return
	}
	OK(c, v)
}

// UpdateAdverseReaction godoc
// @Summary 更新不良反应登记
// @Tags adverse-reactions
// @Security BearerAuth
// @Param id path int true "登记ID"
// @Param body body model.AdverseReaction true "登记记录"
// @Success 200 {object} Body
// @Router /adverse-reactions/{id} [put]
func (h *PharmaServiceHandler) UpdateAdverseReaction(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	var v model.AdverseReaction
	if err := c.ShouldBindJSON(&v); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.UpdateAdverseReaction(c.Request.Context(), id, &v); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

// DeleteAdverseReaction godoc
// @Summary 删除不良反应登记
// @Tags adverse-reactions
// @Security BearerAuth
// @Param id path int true "登记ID"
// @Success 200 {object} Body
// @Router /adverse-reactions/{id} [delete]
func (h *PharmaServiceHandler) DeleteAdverseReaction(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.DeleteAdverseReaction(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

// ListGuidances godoc
// @Summary 用药指导列表
// @Tags medication-guidances
// @Security BearerAuth
// @Param keyword query string false "关键字"
// @Param page query int false "页码"
// @Param page_size query int false "每页条数"
// @Success 200 {object} Body
// @Router /medication-guidances [get]
func (h *PharmaServiceHandler) ListGuidances(c *gin.Context) {
	var q pagination.Query
	if err := c.ShouldBindQuery(&q); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	q.Normalize()
	list, total, err := h.svc.ListGuidances(c.Request.Context(), c.Query("keyword"), q.Page, q.PageSize)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, pagination.Of(list, total, &q))
}

// CreateGuidance godoc
// @Summary 新建用药指导
// @Tags medication-guidances
// @Accept json
// @Security BearerAuth
// @Param body body model.MedicationGuidance true "指导记录"
// @Success 200 {object} Body
// @Router /medication-guidances [post]
func (h *PharmaServiceHandler) CreateGuidance(c *gin.Context) {
	var v model.MedicationGuidance
	if err := c.ShouldBindJSON(&v); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.CreateGuidance(c.Request.Context(), &v); err != nil {
		Error(c, err)
		return
	}
	OK(c, v)
}

// UpdateGuidance godoc
// @Summary 更新用药指导
// @Tags medication-guidances
// @Security BearerAuth
// @Param id path int true "指导ID"
// @Param body body model.MedicationGuidance true "指导记录"
// @Success 200 {object} Body
// @Router /medication-guidances/{id} [put]
func (h *PharmaServiceHandler) UpdateGuidance(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	var v model.MedicationGuidance
	if err := c.ShouldBindJSON(&v); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.UpdateGuidance(c.Request.Context(), id, &v); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

// DeleteGuidance godoc
// @Summary 删除用药指导
// @Tags medication-guidances
// @Security BearerAuth
// @Param id path int true "指导ID"
// @Success 200 {object} Body
// @Router /medication-guidances/{id} [delete]
func (h *PharmaServiceHandler) DeleteGuidance(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.DeleteGuidance(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}
