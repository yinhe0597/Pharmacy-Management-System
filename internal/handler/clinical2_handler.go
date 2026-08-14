package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"yaofang/internal/middleware"
	"yaofang/internal/pkg/errs"
	"yaofang/internal/pkg/pagination"
	"yaofang/internal/service"
)

// VisitHandler 二期就诊模块接口（docs/20 S2-S4）：就诊/病历/合并结算。
type VisitHandler struct {
	visits  *service.VisitService
	records *service.MedicalRecordService
	charges *service.ChargeService
}

func NewVisitHandler(v *service.VisitService, r *service.MedicalRecordService, c *service.ChargeService) *VisitHandler {
	return &VisitHandler{visits: v, records: r, charges: c}
}

func (h *VisitHandler) Register(g Groups) {
	// 就诊：查询任意登录；写操作临床人员（跟诊护士挂号/接诊，医生接诊结束）
	g.Authed.GET("/visits", h.ListVisits)
	g.Authed.GET("/visits/:id", h.GetVisit)
	g.Clinical.POST("/visits", h.RegisterVisit)
	g.Clinical.POST("/visits/:id/start", h.StartVisit)
	g.Clinical.POST("/visits/:id/finish", h.FinishVisit)
	g.Clinical.POST("/visits/:id/cancel", h.CancelVisit)
	// 病历：跟诊护士/医生读写
	g.Patient.GET("/visits/:id/medical-record", h.GetMedicalRecord)
	g.Patient.POST("/visits/:id/medical-record", h.UpsertMedicalRecord)
	// 合并结算：查看=Billing；收费/退费=ChargeStaff（含双护士、财务）
	g.Billing.GET("/charges", h.ListCharges)
	g.Billing.GET("/charges/:id", h.GetCharge)
	g.Charge.POST("/visits/:id/charge", h.CreateCharge)
	g.Charge.POST("/charges/:id/pay", h.PayCharge)
	g.Charge.POST("/charges/:id/refund", h.RefundCharge)
}

// ---- 就诊 ----

// RegisterVisit godoc
// @Summary 挂号/分诊
// @Tags visits
// @Security BearerAuth
// @Accept json
// @Param body body service.VisitInput true "挂号信息"
// @Success 200 {object} Body
// @Router /visits [post]
func (h *VisitHandler) RegisterVisit(c *gin.Context) {
	var req service.VisitInput
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	v, err := h.visits.Register(c.Request.Context(), req, middleware.UserNameFromCtx(c))
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, v)
}

// ListVisits godoc
// @Summary 就诊列表
// @Tags visits
// @Security BearerAuth
// @Param patient_id query int false "患者ID"
// @Param doctor_id query int false "医生ID"
// @Param status query string false "状态 waiting/visiting/finished/cancelled"
// @Param start query string false "开始时间"
// @Param end query string false "结束时间"
// @Param page query int false "页码"
// @Param page_size query int false "每页条数"
// @Success 200 {object} Body
// @Router /visits [get]
func (h *VisitHandler) ListVisits(c *gin.Context) {
	var q pagination.Query
	if err := c.ShouldBindQuery(&q); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	q.Normalize()
	f := service.VisitFilter{
		PatientID: int64(atoi(c.Query("patient_id"))),
		DoctorID:  int64(atoi(c.Query("doctor_id"))),
		Status:    c.Query("status"),
	}
	if t := parseTime(c.Query("start")); t != nil {
		f.Start = *t
	}
	if t := parseTime(c.Query("end")); t != nil {
		f.End = *t
	}
	list, total, err := h.visits.List(c.Request.Context(), f, q.Page, q.PageSize)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, pagination.Of(list, total, &q))
}

// GetVisit godoc
// @Summary 就诊详情
// @Tags visits
// @Security BearerAuth
// @Param id path int true "就诊ID"
// @Success 200 {object} Body
// @Router /visits/{id} [get]
func (h *VisitHandler) GetVisit(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	v, err := h.visits.Get(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, v)
}

// StartVisit godoc
// @Summary 接诊（waiting→visiting）
// @Tags visits
// @Security BearerAuth
// @Param id path int true "就诊ID"
// @Success 200 {object} Body
// @Router /visits/{id}/start [post]
func (h *VisitHandler) StartVisit(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.visits.Start(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

// FinishVisit godoc
// @Summary 结束就诊（visiting→finished）
// @Tags visits
// @Security BearerAuth
// @Param id path int true "就诊ID"
// @Success 200 {object} Body
// @Router /visits/{id}/finish [post]
func (h *VisitHandler) FinishVisit(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.visits.Finish(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

// CancelVisit godoc
// @Summary 退号（waiting→cancelled）
// @Tags visits
// @Security BearerAuth
// @Param id path int true "就诊ID"
// @Success 200 {object} Body
// @Router /visits/{id}/cancel [post]
func (h *VisitHandler) CancelVisit(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.visits.Cancel(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

// ---- 病历 ----

// GetMedicalRecord godoc
// @Summary 就诊病历详情
// @Tags medical-records
// @Security BearerAuth
// @Param id path int true "就诊ID"
// @Success 200 {object} Body
// @Router /visits/{id}/medical-record [get]
func (h *VisitHandler) GetMedicalRecord(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	rec, err := h.records.Get(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, rec)
}

// UpsertMedicalRecord godoc
// @Summary 保存就诊病历（一就诊一病历，可反复保存；含多诊断）
// @Tags medical-records
// @Security BearerAuth
// @Accept json
// @Param id path int true "就诊ID"
// @Param body body service.MedicalRecordInput true "病历"
// @Success 200 {object} Body
// @Router /visits/{id}/medical-record [post]
func (h *VisitHandler) UpsertMedicalRecord(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	var req service.MedicalRecordInput
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	rec, err := h.records.Upsert(c.Request.Context(), id, req, middleware.UserNameFromCtx(c))
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, rec)
}

// ---- 合并结算 ----

// CreateCharge godoc
// @Summary 按就诊生成合并结算单（聚合药费+诊疗项目费）
// @Tags charges
// @Security BearerAuth
// @Accept json
// @Param id path int true "就诊ID"
// @Param body body chargeCreateRequest true "结算参数"
// @Success 200 {object} Body
// @Router /visits/{id}/charge [post]
func (h *VisitHandler) CreateCharge(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	var req chargeCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	ch, err := h.charges.Create(c.Request.Context(), id, req.Discount, middleware.UserNameFromCtx(c))
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, ch)
}

// ListCharges godoc
// @Summary 结算单列表
// @Tags charges
// @Security BearerAuth
// @Param patient_id query int false "患者ID"
// @Param visit_id query int false "就诊ID"
// @Param status query string false "状态 pending/paid/refunded"
// @Param start query string false "开始时间"
// @Param end query string false "结束时间"
// @Param page query int false "页码"
// @Param page_size query int false "每页条数"
// @Success 200 {object} Body
// @Router /charges [get]
func (h *VisitHandler) ListCharges(c *gin.Context) {
	var q pagination.Query
	if err := c.ShouldBindQuery(&q); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	q.Normalize()
	f := service.ChargeFilter{
		PatientID: int64(atoi(c.Query("patient_id"))),
		VisitID:   int64(atoi(c.Query("visit_id"))),
		Status:    c.Query("status"),
	}
	if t := parseTime(c.Query("start")); t != nil {
		f.Start = *t
	}
	if t := parseTime(c.Query("end")); t != nil {
		f.End = *t
	}
	list, total, err := h.charges.List(c.Request.Context(), f, q.Page, q.PageSize)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, pagination.Of(list, total, &q))
}

// GetCharge godoc
// @Summary 结算单详情（含明细）
// @Tags charges
// @Security BearerAuth
// @Param id path int true "结算单ID"
// @Success 200 {object} Body
// @Router /charges/{id} [get]
func (h *VisitHandler) GetCharge(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	ch, err := h.charges.Get(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, ch)
}

// PayCharge godoc
// @Summary 收费（pending→paid）
// @Tags charges
// @Security BearerAuth
// @Accept json
// @Param id path int true "结算单ID"
// @Param body body chargePayRequest true "收费参数"
// @Success 200 {object} Body
// @Router /charges/{id}/pay [post]
func (h *VisitHandler) PayCharge(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	var req chargePayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.charges.Pay(c.Request.Context(), id, req.PaidAmount, middleware.UserNameFromCtx(c)); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

// RefundCharge godoc
// @Summary 退费（paid→refunded）
// @Tags charges
// @Security BearerAuth
// @Param id path int true "结算单ID"
// @Success 200 {object} Body
// @Router /charges/{id}/refund [post]
func (h *VisitHandler) RefundCharge(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.charges.Refund(c.Request.Context(), id, middleware.UserNameFromCtx(c)); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

type chargeCreateRequest struct {
	Discount int64 `json:"discount"` // 优惠金额（分），默认 0
}

type chargePayRequest struct {
	PaidAmount int64 `json:"paid_amount"` // 实收金额（分），默认=应收
}
