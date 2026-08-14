package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"yaofang/internal/middleware"
	"yaofang/internal/pkg/errs"
	"yaofang/internal/pkg/pagination"
	"yaofang/internal/service"
)

// PurchaseHandler 采购接口。
type PurchaseHandler struct {
	svc *service.PurchaseService
	inv *service.InventoryService
}

// NewPurchaseHandler 构建采购 Handler。
func NewPurchaseHandler(svc *service.PurchaseService, inv *service.InventoryService) *PurchaseHandler {
	return &PurchaseHandler{svc: svc, inv: inv}
}

// Register 注册路由。
func (h *PurchaseHandler) Register(g Groups) {
	g.Authed.GET("/purchase/suggestions", h.Suggestions)
	g.Purchase.POST("/purchase-orders", h.CreateOrder)
	g.Authed.GET("/purchase-orders", h.ListOrders)
	g.Authed.GET("/purchase-orders/:id", h.GetOrder)
	g.Purchase.POST("/purchase-orders/:id/submit", h.SubmitOrder)
	g.Purchase.POST("/purchase-orders/:id/cancel", h.CancelOrder)
	g.Purchase.POST("/purchase-orders/:id/receive", h.Receive)
	g.Authed.GET("/purchase-receipts", h.ListReceipts)
	g.Authed.GET("/purchase-receipts/:id", h.GetReceipt)
	g.Purchase.POST("/purchase-receipts/:id/complete", h.CompleteReceipt)
}

type createOrderRequest struct {
	SupplierID int64                 `json:"supplier_id" binding:"required"`
	ExpectedAt *time.Time            `json:"expected_at"`
	Remarks    string                `json:"remarks"`
	Items      []service.POItemInput `json:"items" binding:"required"`
}

// CreateOrder godoc
// @Summary 创建采购单
// @Tags purchase
// @Accept json
// @Security BearerAuth
// @Param body body createOrderRequest true "采购单"
// @Success 200 {object} Body
// @Router /purchase-orders [post]
func (h *PurchaseHandler) CreateOrder(c *gin.Context) {
	var req createOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	po, err := h.svc.CreateOrder(c.Request.Context(), req.SupplierID, req.Items, req.ExpectedAt, req.Remarks, middleware.UserIDFromCtx(c))
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, po)
}

// ListOrders godoc
// @Summary 采购单列表
// @Tags purchase
// @Security BearerAuth
// @Param supplier_id query int false "供应商ID"
// @Param status query string false "状态"
// @Param page query int false "页码"
// @Param page_size query int false "每页条数"
// @Success 200 {object} Body
// @Router /purchase-orders [get]
func (h *PurchaseHandler) ListOrders(c *gin.Context) {
	var q pagination.Query
	if err := c.ShouldBindQuery(&q); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	q.Normalize()
	list, total, err := h.svc.ListOrders(c.Request.Context(), int64(atoi(c.Query("supplier_id"))), c.Query("status"), q.Page, q.PageSize)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, pagination.Of(list, total, &q))
}

// GetOrder godoc
// @Summary 采购单详情
// @Tags purchase
// @Security BearerAuth
// @Param id path int true "采购单ID"
// @Success 200 {object} Body
// @Router /purchase-orders/{id} [get]
func (h *PurchaseHandler) GetOrder(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	detail, err := h.svc.GetOrder(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, detail)
}

// SubmitOrder godoc
// @Summary 提交采购单
// @Tags purchase
// @Security BearerAuth
// @Param id path int true "采购单ID"
// @Success 200 {object} Body
// @Router /purchase-orders/{id}/submit [post]
func (h *PurchaseHandler) SubmitOrder(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.SubmitOrder(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

// CancelOrder godoc
// @Summary 作废采购单
// @Tags purchase
// @Security BearerAuth
// @Param id path int true "采购单ID"
// @Success 200 {object} Body
// @Router /purchase-orders/{id}/cancel [post]
func (h *PurchaseHandler) CancelOrder(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.CancelOrder(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

type receiveRequest struct {
	Items []service.ReceiveItemInput `json:"items" binding:"required"`
}

// Receive godoc
// @Summary 采购收货（质检）
// @Tags purchase
// @Accept json
// @Security BearerAuth
// @Param id path int true "采购单ID"
// @Param body body receiveRequest true "收货明细"
// @Success 200 {object} Body
// @Router /purchase-orders/{id}/receive [post]
func (h *PurchaseHandler) Receive(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	var req receiveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	receipt, err := h.svc.Receive(c.Request.Context(), id, req.Items, middleware.UserIDFromCtx(c))
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, receipt)
}

// ListReceipts godoc
// @Summary 收货单列表
// @Tags purchase
// @Security BearerAuth
// @Param supplier_id query int false "供应商ID"
// @Param status query string false "状态"
// @Param page query int false "页码"
// @Param page_size query int false "每页条数"
// @Success 200 {object} Body
// @Router /purchase-receipts [get]
func (h *PurchaseHandler) ListReceipts(c *gin.Context) {
	var q pagination.Query
	if err := c.ShouldBindQuery(&q); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	q.Normalize()
	list, total, err := h.svc.ListReceipts(c.Request.Context(), int64(atoi(c.Query("supplier_id"))), c.Query("status"), q.Page, q.PageSize)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, pagination.Of(list, total, &q))
}

// GetReceipt godoc
// @Summary 收货单详情
// @Tags purchase
// @Security BearerAuth
// @Param id path int true "收货单ID"
// @Success 200 {object} Body
// @Router /purchase-receipts/{id} [get]
func (h *PurchaseHandler) GetReceipt(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	detail, err := h.svc.GetReceipt(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, detail)
}

// CompleteReceipt godoc
// @Summary 收货确认入库
// @Tags purchase
// @Security BearerAuth
// @Param id path int true "收货单ID"
// @Success 200 {object} Body
// @Router /purchase-receipts/{id}/complete [post]
func (h *PurchaseHandler) CompleteReceipt(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.CompleteReceipt(c.Request.Context(), id, middleware.UserIDFromCtx(c), middleware.UserNameFromCtx(c)); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

// Suggestions godoc
// @Summary 采购计划建议
// @Tags purchase
// @Security BearerAuth
// @Success 200 {object} Body
// @Router /purchase/suggestions [get]
func (h *PurchaseHandler) Suggestions(c *gin.Context) {
	list, err := h.inv.PurchaseSuggestions(c.Request.Context())
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, list)
}
