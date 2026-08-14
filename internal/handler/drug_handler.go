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

// DrugHandler 药品接口。
type DrugHandler struct {
	svc *service.DrugService
}

// NewDrugHandler 构建药品 Handler。
func NewDrugHandler(svc *service.DrugService) *DrugHandler { return &DrugHandler{svc: svc} }

// Register 注册路由。
func (h *DrugHandler) Register(g Groups) {
	g.Authed.GET("/drugs", h.List)
	g.DrugAdmin.POST("/drugs", h.Create)
	g.Authed.GET("/drugs/:id", h.Get)
	g.DrugAdmin.PUT("/drugs/:id", h.Update)
	g.DrugAdmin.DELETE("/drugs/:id", h.Delete)
	g.DrugAdmin.PATCH("/drugs/:id/status", h.SetStatus)
	g.Authed.GET("/categories", h.ListCategories)
	g.DrugAdmin.POST("/categories", h.CreateCategory)
	g.DrugAdmin.PUT("/categories/:id", h.UpdateCategory)
	g.DrugAdmin.DELETE("/categories/:id", h.DeleteCategory)
	g.Authed.GET("/interactions", h.ListInteractions)
	g.DrugAdmin.POST("/interactions", h.CreateInteraction)
	g.DrugAdmin.PUT("/interactions/:id", h.UpdateInteraction)
	g.DrugAdmin.DELETE("/interactions/:id", h.DeleteInteraction)
}

// Create godoc
// @Summary 新增药品
// @Tags drugs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body model.Drug true "药品信息"
// @Success 200 {object} Body
// @Router /drugs [post]
func (h *DrugHandler) Create(c *gin.Context) {
	var d model.Drug
	if err := c.ShouldBindJSON(&d); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.Create(c.Request.Context(), &d); err != nil {
		Error(c, err)
		return
	}
	OK(c, d)
}

// Get godoc
// @Summary 药品详情
// @Tags drugs
// @Security BearerAuth
// @Param id path int true "药品ID"
// @Success 200 {object} Body
// @Router /drugs/{id} [get]
func (h *DrugHandler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	d, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, d)
}

// Update godoc
// @Summary 更新药品
// @Tags drugs
// @Accept json
// @Security BearerAuth
// @Param id path int true "药品ID"
// @Param body body model.Drug true "药品信息"
// @Success 200 {object} Body
// @Router /drugs/{id} [put]
func (h *DrugHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	var d model.Drug
	if err := c.ShouldBindJSON(&d); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.Update(c.Request.Context(), id, &d); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

// Delete godoc
// @Summary 删除药品（有下游自动冻结）
// @Tags drugs
// @Security BearerAuth
// @Param id path int true "药品ID"
// @Success 200 {object} Body
// @Router /drugs/{id} [delete]
func (h *DrugHandler) Delete(c *gin.Context) {
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

type statusRequest struct {
	Status int `json:"status" binding:"required,oneof=0 1"`
}

// SetStatus godoc
// @Summary 启停用药品
// @Tags drugs
// @Accept json
// @Security BearerAuth
// @Param id path int true "药品ID"
// @Param body body statusRequest true "状态"
// @Success 200 {object} Body
// @Router /drugs/{id}/status [patch]
func (h *DrugHandler) SetStatus(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	var req statusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.SetStatus(c.Request.Context(), id, req.Status); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

// List godoc
// @Summary 药品列表
// @Tags drugs
// @Security BearerAuth
// @Param keyword query string false "关键字"
// @Param category_id query int false "分类"
// @Param antibiotic_level query int false "抗生素分级"
// @Param special_control_type query int false "特殊管制类型"
// @Param status query int false "状态"
// @Param page query int false "页码"
// @Param page_size query int false "每页条数"
// @Success 200 {object} Body
// @Router /drugs [get]
func (h *DrugHandler) List(c *gin.Context) {
	var q pagination.Query
	if err := c.ShouldBindQuery(&q); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	q.Normalize()
	f := repository.DrugListFilter{
		Keyword:            c.Query("keyword"),
		AntibioticLevel:    atoi(c.Query("antibiotic_level")),
		SpecialControlType: atoi(c.Query("special_control_type")),
		Status:             atoi(c.Query("status")),
	}
	f.CategoryID = int64(atoi(c.Query("category_id")))
	list, total, err := h.svc.List(c.Request.Context(), f, q.Page, q.PageSize)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, pagination.Of(list, total, &q))
}

// ---- 分类 ----

// CreateCategory godoc
// @Summary 新增分类
// @Tags categories
// @Accept json
// @Security BearerAuth
// @Param body body model.DrugCategory true "分类"
// @Success 200 {object} Body
// @Router /categories [post]
func (h *DrugHandler) CreateCategory(c *gin.Context) {
	var cat model.DrugCategory
	if err := c.ShouldBindJSON(&cat); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.CreateCategory(c.Request.Context(), &cat); err != nil {
		Error(c, err)
		return
	}
	OK(c, cat)
}

// ListCategories godoc
// @Summary 分类列表
// @Tags categories
// @Security BearerAuth
// @Success 200 {object} Body
// @Router /categories [get]
func (h *DrugHandler) ListCategories(c *gin.Context) {
	list, err := h.svc.ListCategories(c.Request.Context())
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, list)
}

// UpdateCategory godoc
// @Summary 更新分类
// @Tags categories
// @Security BearerAuth
// @Param id path int true "分类ID"
// @Param body body model.DrugCategory true "分类"
// @Success 200 {object} Body
// @Router /categories/{id} [put]
func (h *DrugHandler) UpdateCategory(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	var cat model.DrugCategory
	if err := c.ShouldBindJSON(&cat); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.UpdateCategory(c.Request.Context(), id, &cat); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

// DeleteCategory godoc
// @Summary 删除分类
// @Tags categories
// @Security BearerAuth
// @Param id path int true "分类ID"
// @Success 200 {object} Body
// @Router /categories/{id} [delete]
func (h *DrugHandler) DeleteCategory(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.DeleteCategory(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

// ---- 配伍禁忌 ----

// CreateInteraction godoc
// @Summary 新增配伍禁忌
// @Tags interactions
// @Accept json
// @Security BearerAuth
// @Param body body model.DrugInteraction true "禁忌记录"
// @Success 200 {object} Body
// @Router /interactions [post]
func (h *DrugHandler) CreateInteraction(c *gin.Context) {
	var i model.DrugInteraction
	if err := c.ShouldBindJSON(&i); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.CreateInteraction(c.Request.Context(), &i); err != nil {
		Error(c, err)
		return
	}
	OK(c, i)
}

// ListInteractions godoc
// @Summary 配伍禁忌列表
// @Tags interactions
// @Security BearerAuth
// @Param page query int false "页码"
// @Param page_size query int false "每页条数"
// @Success 200 {object} Body
// @Router /interactions [get]
func (h *DrugHandler) ListInteractions(c *gin.Context) {
	var q pagination.Query
	if err := c.ShouldBindQuery(&q); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	q.Normalize()
	list, total, err := h.svc.ListInteractions(c.Request.Context(), c.Query("keyword"), q.Page, q.PageSize)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, pagination.Of(list, total, &q))
}

// UpdateInteraction godoc
// @Summary 更新配伍禁忌
// @Tags interactions
// @Security BearerAuth
// @Param id path int true "禁忌ID"
// @Param body body model.DrugInteraction true "禁忌记录"
// @Success 200 {object} Body
// @Router /interactions/{id} [put]
func (h *DrugHandler) UpdateInteraction(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	var i model.DrugInteraction
	if err := c.ShouldBindJSON(&i); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.UpdateInteraction(c.Request.Context(), id, &i); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

// DeleteInteraction godoc
// @Summary 删除配伍禁忌
// @Tags interactions
// @Security BearerAuth
// @Param id path int true "禁忌ID"
// @Success 200 {object} Body
// @Router /interactions/{id} [delete]
func (h *DrugHandler) DeleteInteraction(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.DeleteInteraction(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
