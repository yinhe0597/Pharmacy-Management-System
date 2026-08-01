package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"yaofang/internal/middleware"
	"yaofang/internal/pkg/errs"
	"yaofang/internal/pkg/pagination"
	"yaofang/internal/repository"
	"yaofang/internal/service"
)

// PrescriptionHandler 处方接口。
type PrescriptionHandler struct {
	svc *service.PrescriptionService
}

// NewPrescriptionHandler 构建处方 Handler。
func NewPrescriptionHandler(svc *service.PrescriptionService) *PrescriptionHandler {
	return &PrescriptionHandler{svc: svc}
}

// Register 注册路由。
func (h *PrescriptionHandler) Register(r *gin.RouterGroup, _ *gin.RouterGroup, _ *gin.RouterGroup) {
	r.POST("/prescriptions", h.Create)
	r.PUT("/prescriptions/:id", h.Update)
	r.POST("/prescriptions/:id/submit", h.Submit)
	r.POST("/prescriptions/:id/review", h.Review)
	r.POST("/prescriptions/:id/dispense", h.Dispense)
	r.POST("/prescriptions/:id/confirm-dispense", h.ConfirmDispense)
	r.POST("/prescriptions/:id/return", h.Return)
	r.POST("/prescriptions/:id/cancel", h.Cancel)
	r.GET("/prescriptions", h.List)
	r.GET("/prescriptions/:id", h.Get)
	r.GET("/prescriptions/:id/audit-log", h.AuditLog)
}

// Create godoc
// @Summary 处方录入
// @Tags prescriptions
// @Accept json
// @Security BearerAuth
// @Param body body service.PrescriptionInput true "处方"
// @Success 200 {object} Body
// @Router /prescriptions [post]
func (h *PrescriptionHandler) Create(c *gin.Context) {
	var req service.PrescriptionInput
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	p, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, p)
}

// Update godoc
// @Summary 修改处方
// @Tags prescriptions
// @Accept json
// @Security BearerAuth
// @Param id path int true "处方ID"
// @Param body body service.PrescriptionInput true "处方"
// @Success 200 {object} Body
// @Router /prescriptions/{id} [put]
func (h *PrescriptionHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	var req service.PrescriptionInput
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.Update(c.Request.Context(), id, req); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

// Submit godoc
// @Summary 提交审核（预占库存）
// @Tags prescriptions
// @Security BearerAuth
// @Param id path int true "处方ID"
// @Success 200 {object} Body
// @Router /prescriptions/{id}/submit [post]
func (h *PrescriptionHandler) Submit(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.Submit(c.Request.Context(), id, middleware.UserIDFromCtx(c), middleware.UserNameFromCtx(c)); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

// Review godoc
// @Summary 处方审核（仅药师/药房主任可执行）
// @Description 审核动作：pass(通过)→调配中；reject(驳回)→释放预占；return(退回医生)→保持待审核+保留预占，医生可修改后重提交。
// @Tags prescriptions
// @Accept json
// @Security BearerAuth
// @Param id path int true "处方ID"
// @Param body body service.AuditInput true "审核结果（action: pass/reject/return）"
// @Success 200 {object} Body
// @Router /prescriptions/{id}/review [post]
func (h *PrescriptionHandler) Review(c *gin.Context) {
	// 仅药师和药房主任可审核处方
	role := middleware.UserRoleFromCtx(c)
	if role != "pharmacist" && role != "pharmacy_director" && role != "admin" {
		Error(c, errs.ErrForbidden)
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	var req service.AuditInput
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	result, err := h.svc.Review(c.Request.Context(), id, req, middleware.UserIDFromCtx(c), middleware.UserNameFromCtx(c))
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, result)
}

// Dispense godoc
// @Summary 调配（按批号分配）
// @Tags prescriptions
// @Security BearerAuth
// @Param id path int true "处方ID"
// @Success 200 {object} Body
// @Router /prescriptions/{id}/dispense [post]
func (h *PrescriptionHandler) Dispense(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.Dispense(c.Request.Context(), id, middleware.UserIDFromCtx(c), middleware.UserNameFromCtx(c)); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

type confirmDispenseRequest struct {
	CheckerID int64 `json:"checker_id"`
}

// ConfirmDispense godoc
// @Summary 发药确认（实扣库存）
// @Tags prescriptions
// @Accept json
// @Security BearerAuth
// @Param id path int true "处方ID"
// @Param body body confirmDispenseRequest true "核对药师"
// @Success 200 {object} Body
// @Router /prescriptions/{id}/confirm-dispense [post]
func (h *PrescriptionHandler) ConfirmDispense(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	var req confirmDispenseRequest
	_ = c.ShouldBindJSON(&req)
	checkerID := req.CheckerID
	if checkerID == 0 {
		checkerID = middleware.UserIDFromCtx(c)
	}
	if err := h.svc.ConfirmDispense(c.Request.Context(), id, checkerID, middleware.UserNameFromCtx(c)); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

// Return godoc
// @Summary 退药（整方/部分）
// @Tags prescriptions
// @Accept json
// @Security BearerAuth
// @Param id path int true "处方ID"
// @Param body body returnRequest true "退药明细"
// @Success 200 {object} Body
// @Router /prescriptions/{id}/return [post]
func (h *PrescriptionHandler) Return(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	var req returnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.Return(c.Request.Context(), id, req.Items, middleware.UserIDFromCtx(c), middleware.UserNameFromCtx(c)); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

type returnRequest struct {
	Items []service.ReturnItemInput `json:"items" binding:"required"`
}

// Cancel godoc
// @Summary 作废处方（释放预占）
// @Tags prescriptions
// @Security BearerAuth
// @Param id path int true "处方ID"
// @Success 200 {object} Body
// @Router /prescriptions/{id}/cancel [post]
func (h *PrescriptionHandler) Cancel(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.Cancel(c.Request.Context(), id, middleware.UserNameFromCtx(c)); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

// List godoc
// @Summary 处方列表
// @Tags prescriptions
// @Security BearerAuth
// @Param status query string false "状态"
// @Param patient_name query string false "患者姓名"
// @Param start query string false "开始时间"
// @Param end query string false "结束时间"
// @Param page query int false "页码"
// @Param page_size query int false "每页条数"
// @Success 200 {object} Body
// @Router /prescriptions [get]
func (h *PrescriptionHandler) List(c *gin.Context) {
	var q pagination.Query
	if err := c.ShouldBindQuery(&q); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	q.Normalize()
	f := repository.PrescriptionFilter{
		Status:      c.Query("status"),
		PatientName: c.Query("patient_name"),
		Start:       parseTime(c.Query("start")),
		End:         parseTime(c.Query("end")),
	}
	list, total, err := h.svc.List(c.Request.Context(), f, q.Page, q.PageSize)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, pagination.Of(list, total, &q))
}

// Get godoc
// @Summary 处方详情
// @Tags prescriptions
// @Security BearerAuth
// @Param id path int true "处方ID"
// @Success 200 {object} Body
// @Router /prescriptions/{id} [get]
func (h *PrescriptionHandler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	detail, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, detail)
}

// AuditLog godoc
// @Summary 处方状态流转日志
// @Tags prescriptions
// @Security BearerAuth
// @Param id path int true "处方ID"
// @Success 200 {object} Body
// @Router /prescriptions/{id}/audit-log [get]
func (h *PrescriptionHandler) AuditLog(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	detail, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, detail.AuditLogs)
}
