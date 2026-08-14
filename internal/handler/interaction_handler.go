package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"yaofang/internal/model"
	"yaofang/internal/pkg/errs"
	"yaofang/internal/pkg/pagination"
	"yaofang/internal/service"
)

// InteractionHandler 药物相互作用规则管理接口。
type InteractionHandler struct {
	svc *service.InteractionService
}

// NewInteractionHandler 构建交互 Handler。
func NewInteractionHandler(svc *service.InteractionService) *InteractionHandler {
	return &InteractionHandler{svc: svc}
}

// Register 注册路由。
func (h *InteractionHandler) Register(g Groups) {
	// 成分交互规则
	g.Authed.GET("/ingredient-interactions", h.ListIngredientInteractions)
	g.DrugAdmin.POST("/ingredient-interactions", h.CreateIngredientInteraction)
	g.DrugAdmin.PUT("/ingredient-interactions/:id", h.UpdateIngredientInteraction)
	g.DrugAdmin.DELETE("/ingredient-interactions/:id", h.DeleteIngredientInteraction)
	// 分类交互规则
	g.Authed.GET("/class-interactions", h.ListClassInteractions)
	g.DrugAdmin.POST("/class-interactions", h.CreateClassInteraction)
	g.DrugAdmin.PUT("/class-interactions/:id", h.UpdateClassInteraction)
	g.DrugAdmin.DELETE("/class-interactions/:id", h.DeleteClassInteraction)
	// 标签交互规则
	g.Authed.GET("/tag-interactions", h.ListTagInteractions)
	g.DrugAdmin.POST("/tag-interactions", h.CreateTagInteraction)
	g.DrugAdmin.PUT("/tag-interactions/:id", h.UpdateTagInteraction)
	g.DrugAdmin.DELETE("/tag-interactions/:id", h.DeleteTagInteraction)
	// 药品成分映射
	g.Authed.GET("/drugs/:id/ingredients", h.ListDrugIngredients)
	g.DrugAdmin.POST("/drugs/:id/ingredients", h.AddDrugIngredient)
	g.DrugAdmin.DELETE("/drug-ingredients/:id", h.RemoveDrugIngredient)
}

// ---- 成分交互规则 ----

// ListIngredientInteractions godoc
// @Summary 成分交互规则列表
// @Tags ingredient-interactions
// @Security BearerAuth
// @Param page query int false "页码"
// @Param page_size query int false "每页条数"
// @Success 200 {object} Body
// @Router /ingredient-interactions [get]
func (h *InteractionHandler) ListIngredientInteractions(c *gin.Context) {
	var q pagination.Query
	if err := c.ShouldBindQuery(&q); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	q.Normalize()
	list, total, err := h.svc.ListIngredientInteractions(c.Request.Context(), (q.Page-1)*q.PageSize, q.PageSize)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, pagination.Of(list, total, &q))
}

// CreateIngredientInteraction godoc
// @Summary 新建成分交互规则
// @Tags ingredient-interactions
// @Accept json
// @Security BearerAuth
// @Param body body model.IngredientInteraction true "成分交互规则"
// @Success 200 {object} Body
// @Router /ingredient-interactions [post]
func (h *InteractionHandler) CreateIngredientInteraction(c *gin.Context) {
	var v model.IngredientInteraction
	if err := c.ShouldBindJSON(&v); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.CreateIngredientInteraction(c.Request.Context(), &v); err != nil {
		Error(c, err)
		return
	}
	OK(c, v)
}

// UpdateIngredientInteraction godoc
// @Summary 更新成分交互规则
// @Tags ingredient-interactions
// @Accept json
// @Security BearerAuth
// @Param id path int true "规则ID"
// @Param body body model.IngredientInteraction true "成分交互规则"
// @Success 200 {object} Body
// @Router /ingredient-interactions/{id} [put]
func (h *InteractionHandler) UpdateIngredientInteraction(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	var v model.IngredientInteraction
	if err := c.ShouldBindJSON(&v); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	v.ID = id
	if err := h.svc.UpdateIngredientInteraction(c.Request.Context(), &v); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

// DeleteIngredientInteraction godoc
// @Summary 删除成分交互规则
// @Tags ingredient-interactions
// @Security BearerAuth
// @Param id path int true "规则ID"
// @Success 200 {object} Body
// @Router /ingredient-interactions/{id} [delete]
func (h *InteractionHandler) DeleteIngredientInteraction(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.DeleteIngredientInteraction(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

// ---- 分类交互规则 ----

// ListClassInteractions godoc
// @Summary 分类交互规则列表
// @Tags class-interactions
// @Security BearerAuth
// @Param page query int false "页码"
// @Param page_size query int false "每页条数"
// @Success 200 {object} Body
// @Router /class-interactions [get]
func (h *InteractionHandler) ListClassInteractions(c *gin.Context) {
	var q pagination.Query
	if err := c.ShouldBindQuery(&q); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	q.Normalize()
	list, total, err := h.svc.ListClassInteractions(c.Request.Context(), (q.Page-1)*q.PageSize, q.PageSize)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, pagination.Of(list, total, &q))
}

// CreateClassInteraction godoc
// @Summary 新建分类交互规则
// @Tags class-interactions
// @Accept json
// @Security BearerAuth
// @Param body body model.ClassInteractionRule true "分类交互规则"
// @Success 200 {object} Body
// @Router /class-interactions [post]
func (h *InteractionHandler) CreateClassInteraction(c *gin.Context) {
	var v model.ClassInteractionRule
	if err := c.ShouldBindJSON(&v); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.CreateClassInteraction(c.Request.Context(), &v); err != nil {
		Error(c, err)
		return
	}
	OK(c, v)
}

// UpdateClassInteraction godoc
// @Summary 更新分类交互规则
// @Tags class-interactions
// @Accept json
// @Security BearerAuth
// @Param id path int true "规则ID"
// @Param body body model.ClassInteractionRule true "分类交互规则"
// @Success 200 {object} Body
// @Router /class-interactions/{id} [put]
func (h *InteractionHandler) UpdateClassInteraction(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	var v model.ClassInteractionRule
	if err := c.ShouldBindJSON(&v); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	v.ID = id
	if err := h.svc.UpdateClassInteraction(c.Request.Context(), &v); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

// DeleteClassInteraction godoc
// @Summary 删除分类交互规则
// @Tags class-interactions
// @Security BearerAuth
// @Param id path int true "规则ID"
// @Success 200 {object} Body
// @Router /class-interactions/{id} [delete]
func (h *InteractionHandler) DeleteClassInteraction(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.DeleteClassInteraction(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

// ---- 标签交互规则 ----

// ListTagInteractions godoc
// @Summary 标签交互规则列表
// @Tags tag-interactions
// @Security BearerAuth
// @Param page query int false "页码"
// @Param page_size query int false "每页条数"
// @Success 200 {object} Body
// @Router /tag-interactions [get]
func (h *InteractionHandler) ListTagInteractions(c *gin.Context) {
	var q pagination.Query
	if err := c.ShouldBindQuery(&q); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	q.Normalize()
	list, total, err := h.svc.ListTagInteractions(c.Request.Context(), (q.Page-1)*q.PageSize, q.PageSize)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, pagination.Of(list, total, &q))
}

// CreateTagInteraction godoc
// @Summary 新建标签交互规则
// @Tags tag-interactions
// @Accept json
// @Security BearerAuth
// @Param body body model.TagInteraction true "标签交互规则"
// @Success 200 {object} Body
// @Router /tag-interactions [post]
func (h *InteractionHandler) CreateTagInteraction(c *gin.Context) {
	var v model.TagInteraction
	if err := c.ShouldBindJSON(&v); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.CreateTagInteraction(c.Request.Context(), &v); err != nil {
		Error(c, err)
		return
	}
	OK(c, v)
}

// UpdateTagInteraction godoc
// @Summary 更新标签交互规则
// @Tags tag-interactions
// @Accept json
// @Security BearerAuth
// @Param id path int true "规则ID"
// @Param body body model.TagInteraction true "标签交互规则"
// @Success 200 {object} Body
// @Router /tag-interactions/{id} [put]
func (h *InteractionHandler) UpdateTagInteraction(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	var v model.TagInteraction
	if err := c.ShouldBindJSON(&v); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	v.ID = id
	if err := h.svc.UpdateTagInteraction(c.Request.Context(), &v); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

// DeleteTagInteraction godoc
// @Summary 删除标签交互规则
// @Tags tag-interactions
// @Security BearerAuth
// @Param id path int true "规则ID"
// @Success 200 {object} Body
// @Router /tag-interactions/{id} [delete]
func (h *InteractionHandler) DeleteTagInteraction(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.DeleteTagInteraction(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

// ---- 药品成分映射 ----

// ListDrugIngredients godoc
// @Summary 药品成分列表
// @Tags drug-ingredients
// @Security BearerAuth
// @Param id path int true "药品ID"
// @Success 200 {object} Body
// @Router /drugs/{id}/ingredients [get]
func (h *InteractionHandler) ListDrugIngredients(c *gin.Context) {
	drugID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	list, err := h.svc.ListDrugIngredients(c.Request.Context(), drugID)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, list)
}

// AddDrugIngredient godoc
// @Summary 添加药品成分
// @Tags drug-ingredients
// @Accept json
// @Security BearerAuth
// @Param id path int true "药品ID"
// @Param body body model.DrugIngredient true "成分信息"
// @Success 200 {object} Body
// @Router /drugs/{id}/ingredients [post]
func (h *InteractionHandler) AddDrugIngredient(c *gin.Context) {
	drugID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	var v model.DrugIngredient
	if err := c.ShouldBindJSON(&v); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	v.DrugID = drugID
	if err := h.svc.AddDrugIngredient(c.Request.Context(), &v); err != nil {
		Error(c, err)
		return
	}
	OK(c, v)
}

// RemoveDrugIngredient godoc
// @Summary 删除药品成分
// @Tags drug-ingredients
// @Security BearerAuth
// @Param id path int true "成分映射ID"
// @Success 200 {object} Body
// @Router /drug-ingredients/{id} [delete]
func (h *InteractionHandler) RemoveDrugIngredient(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.RemoveDrugIngredient(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}
