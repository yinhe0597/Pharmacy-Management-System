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
func (h *InteractionHandler) Register(authed *gin.RouterGroup, _ *gin.RouterGroup, _ *gin.RouterGroup) {
	// 成分交互规则
	authed.GET("/ingredient-interactions", h.ListIngredientInteractions)
	authed.POST("/ingredient-interactions", h.CreateIngredientInteraction)
	authed.PUT("/ingredient-interactions/:id", h.UpdateIngredientInteraction)
	authed.DELETE("/ingredient-interactions/:id", h.DeleteIngredientInteraction)
	// 分类交互规则
	authed.GET("/class-interactions", h.ListClassInteractions)
	authed.POST("/class-interactions", h.CreateClassInteraction)
	authed.PUT("/class-interactions/:id", h.UpdateClassInteraction)
	authed.DELETE("/class-interactions/:id", h.DeleteClassInteraction)
	// 标签交互规则
	authed.GET("/tag-interactions", h.ListTagInteractions)
	authed.POST("/tag-interactions", h.CreateTagInteraction)
	authed.PUT("/tag-interactions/:id", h.UpdateTagInteraction)
	authed.DELETE("/tag-interactions/:id", h.DeleteTagInteraction)
	// 药品成分映射
	authed.GET("/drugs/:id/ingredients", h.ListDrugIngredients)
	authed.POST("/drugs/:id/ingredients", h.AddDrugIngredient)
	authed.DELETE("/drug-ingredients/:id", h.RemoveDrugIngredient)
}

// ---- 成分交互规则 ----

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
	if err := h.svc.UpdateIngredientInteraction(c.Request.Context(), &v); err != nil {
		Error(c, err)
		return
	}
	v.ID = id
	OK(c, nil)
}

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
