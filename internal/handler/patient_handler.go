package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"yaofang/internal/model"
	"yaofang/internal/pkg/errs"
	"yaofang/internal/pkg/pagination"
	"yaofang/internal/service/patient"
)

// PatientHandler 患者档案与过敏史接口。
type PatientHandler struct {
	svc *patient.PatientService
}

// NewPatientHandler 构建患者 Handler。
func NewPatientHandler(svc *patient.PatientService) *PatientHandler {
	return &PatientHandler{svc: svc}
}

// Register 注册路由（患者信息管理：写=跟诊护士/医生/主任/管理员；读=患者管理∪药房人员，docs/18）。
func (h *PatientHandler) Register(g Groups) {
	g.PatientRead.GET("/patients", h.List)
	g.Patient.POST("/patients", h.Create)
	g.PatientRead.GET("/patients/:id", h.Get)
	g.Patient.PUT("/patients/:id", h.Update)
	g.PatientRead.GET("/patients/:id/allergies", h.ListAllergies)
	g.Patient.POST("/patients/:id/allergies", h.AddAllergy)
	g.Patient.DELETE("/patient-allergies/:id", h.DeleteAllergy)
	g.PatientRead.GET("/patients/:id/medication-history", h.MedicationHistory)
}

// List godoc
// @Summary 患者档案列表
// @Tags patients
// @Security BearerAuth
// @Param keyword query string false "姓名/卡号/电话"
// @Param page query int false "页码"
// @Param page_size query int false "每页条数"
// @Success 200 {object} Body
// @Router /patients [get]
func (h *PatientHandler) List(c *gin.Context) {
	var q pagination.Query
	if err := c.ShouldBindQuery(&q); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	q.Normalize()
	list, total, err := h.svc.List(c.Request.Context(), c.Query("keyword"), q.Page, q.PageSize)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, pagination.Of(list, total, &q))
}

// Create godoc
// @Summary 新建患者档案
// @Tags patients
// @Accept json
// @Security BearerAuth
// @Param body body model.Patient true "患者档案"
// @Success 200 {object} Body
// @Router /patients [post]
func (h *PatientHandler) Create(c *gin.Context) {
	var p model.Patient
	if err := c.ShouldBindJSON(&p); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if p.CardNo == "" || p.Name == "" {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.Create(c.Request.Context(), &p); err != nil {
		Error(c, err)
		return
	}
	OK(c, p)
}

// Get godoc
// @Summary 患者档案详情
// @Tags patients
// @Security BearerAuth
// @Param id path int true "患者ID"
// @Success 200 {object} Body
// @Router /patients/{id} [get]
func (h *PatientHandler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	p, err := h.svc.GetDetail(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, p)
}

// Update godoc
// @Summary 更新患者档案
// @Tags patients
// @Accept json
// @Security BearerAuth
// @Param id path int true "患者ID"
// @Param body body model.Patient true "患者档案"
// @Success 200 {object} Body
// @Router /patients/{id} [put]
func (h *PatientHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	var p model.Patient
	if err := c.ShouldBindJSON(&p); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.Update(c.Request.Context(), id, &p); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

// ListAllergies godoc
// @Summary 患者过敏史
// @Tags patients
// @Security BearerAuth
// @Param id path int true "患者ID"
// @Success 200 {object} Body
// @Router /patients/{id}/allergies [get]
func (h *PatientHandler) ListAllergies(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	list, err := h.svc.ListAllergiesDetail(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, list)
}

// AddAllergy godoc
// @Summary 新增过敏记录
// @Tags patients
// @Accept json
// @Security BearerAuth
// @Param id path int true "患者ID"
// @Param body body model.PatientAllergy true "过敏记录"
// @Success 200 {object} Body
// @Router /patients/{id}/allergies [post]
func (h *PatientHandler) AddAllergy(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	var a model.PatientAllergy
	if err := c.ShouldBindJSON(&a); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	a.PatientID = id
	if err := h.svc.AddAllergy(c.Request.Context(), &a); err != nil {
		Error(c, err)
		return
	}
	OK(c, a)
}

// DeleteAllergy godoc
// @Summary 删除过敏记录
// @Tags patients
// @Security BearerAuth
// @Param id path int true "过敏记录ID"
// @Success 200 {object} Body
// @Router /patient-allergies/{id} [delete]
func (h *PatientHandler) DeleteAllergy(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.DeleteAllergy(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

// MedicationHistory godoc
// @Summary 患者用药史（已发药/已退药明细）
// @Tags patients
// @Security BearerAuth
// @Param id path int true "患者ID"
// @Success 200 {object} Body
// @Router /patients/{id}/medication-history [get]
func (h *PatientHandler) MedicationHistory(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	list, err := h.svc.GetMedicationHistory(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, list)
}
