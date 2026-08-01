package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"yaofang/internal/model"
	"yaofang/internal/pkg/errs"
	"yaofang/internal/pkg/pagination"
	"yaofang/internal/service"
)

// SupplierHandler 供应商接口。
type SupplierHandler struct {
	svc *service.SupplierService
}

// NewSupplierHandler 构建供应商 Handler。
func NewSupplierHandler(svc *service.SupplierService) *SupplierHandler {
	return &SupplierHandler{svc: svc}
}

// Register 注册路由。
func (h *SupplierHandler) Register(r *gin.RouterGroup, _ *gin.RouterGroup, _ *gin.RouterGroup) {
	r.GET("/suppliers", h.List)
	r.POST("/suppliers", h.Create)
	r.GET("/suppliers/:id", h.Get)
	r.PUT("/suppliers/:id", h.Update)
	r.DELETE("/suppliers/:id", h.Delete)
	r.GET("/drugs/:id/suppliers", h.ListDrugSuppliers)
	r.POST("/drugs/:id/suppliers", h.BindDrugSupplier)
	r.DELETE("/drug-suppliers/:id", h.DeleteDrugSupplier)
}

// Create godoc
// @Summary 新增供应商
// @Tags suppliers
// @Accept json
// @Security BearerAuth
// @Param body body model.Supplier true "供应商"
// @Success 200 {object} Body
// @Router /suppliers [post]
func (h *SupplierHandler) Create(c *gin.Context) {
	var sp model.Supplier
	if err := c.ShouldBindJSON(&sp); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.Create(c.Request.Context(), &sp); err != nil {
		Error(c, err)
		return
	}
	OK(c, sp)
}

// Get godoc
// @Summary 供应商详情
// @Tags suppliers
// @Security BearerAuth
// @Param id path int true "供应商ID"
// @Success 200 {object} Body
// @Router /suppliers/{id} [get]
func (h *SupplierHandler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	sp, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, sp)
}

// Update godoc
// @Summary 更新供应商
// @Tags suppliers
// @Security BearerAuth
// @Param id path int true "供应商ID"
// @Param body body model.Supplier true "供应商"
// @Success 200 {object} Body
// @Router /suppliers/{id} [put]
func (h *SupplierHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	var sp model.Supplier
	if err := c.ShouldBindJSON(&sp); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.Update(c.Request.Context(), id, &sp); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

// Delete godoc
// @Summary 删除供应商
// @Tags suppliers
// @Security BearerAuth
// @Param id path int true "供应商ID"
// @Success 200 {object} Body
// @Router /suppliers/{id} [delete]
func (h *SupplierHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

// List godoc
// @Summary 供应商列表
// @Tags suppliers
// @Security BearerAuth
// @Param keyword query string false "关键字"
// @Param status query int false "状态"
// @Param page query int false "页码"
// @Param page_size query int false "每页条数"
// @Success 200 {object} Body
// @Router /suppliers [get]
func (h *SupplierHandler) List(c *gin.Context) {
	var q pagination.Query
	if err := c.ShouldBindQuery(&q); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	q.Normalize()
	list, total, err := h.svc.List(c.Request.Context(), c.Query("keyword"), atoi(c.Query("status")), q.Page, q.PageSize)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, pagination.Of(list, total, &q))
}

// ListDrugSuppliers godoc
// @Summary 药品供货关系列表
// @Tags suppliers
// @Security BearerAuth
// @Param id path int true "药品ID"
// @Success 200 {object} Body
// @Router /drugs/{id}/suppliers [get]
func (h *SupplierHandler) ListDrugSuppliers(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	list, err := h.svc.ListDrugSuppliers(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, list)
}

type bindDrugSupplierRequest struct {
	SupplierID    int64 `json:"supplier_id" binding:"required"`
	IsDefault     bool  `json:"is_default"`
	PurchasePrice int64 `json:"purchase_price"`
}

// BindDrugSupplier godoc
// @Summary 绑定药品-供应商关系
// @Tags suppliers
// @Accept json
// @Security BearerAuth
// @Param id path int true "药品ID"
// @Param body body bindDrugSupplierRequest true "关系信息"
// @Success 200 {object} Body
// @Router /drugs/{id}/suppliers [post]
func (h *SupplierHandler) BindDrugSupplier(c *gin.Context) {
	drugID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	var req bindDrugSupplierRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.BindDrug(c.Request.Context(), drugID, req.SupplierID, req.IsDefault, req.PurchasePrice); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

// DeleteDrugSupplier godoc
// @Summary 删除供货关系
// @Tags suppliers
// @Security BearerAuth
// @Param id path int true "关系ID"
// @Success 200 {object} Body
// @Router /drug-suppliers/{id} [delete]
func (h *SupplierHandler) DeleteDrugSupplier(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.DeleteDrugSupplier(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}
