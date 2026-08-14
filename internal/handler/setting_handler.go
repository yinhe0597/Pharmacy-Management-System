package handler

import (
	"github.com/gin-gonic/gin"

	"yaofang/internal/middleware"
	"yaofang/internal/pkg/errs"
	"yaofang/internal/service"
)

// SettingHandler 系统设置接口（管理员）。
type SettingHandler struct {
	svc *service.SettingService
}

func NewSettingHandler(svc *service.SettingService) *SettingHandler {
	return &SettingHandler{svc: svc}
}

func (h *SettingHandler) Register(g Groups) {
	g.UserAdmin.GET("/system-settings", h.List)
	g.UserAdmin.PUT("/system-settings/:key", h.Update)
}

// List godoc
// @Summary 系统设置列表（默认诊费配置）
// @Tags system-settings
// @Security BearerAuth
// @Success 200 {object} Body
// @Router /system-settings [get]
func (h *SettingHandler) List(c *gin.Context) {
	list, err := h.svc.List(c.Request.Context())
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, list)
}

type settingUpdateRequest struct {
	Value string `json:"value" binding:"required"`
}

// Update godoc
// @Summary 更新系统设置
// @Tags system-settings
// @Security BearerAuth
// @Accept json
// @Param key path string true "设置键（default_registration_fee/default_consultation_fee）"
// @Param body body settingUpdateRequest true "设置值"
// @Success 200 {object} Body
// @Router /system-settings/{key} [put]
func (h *SettingHandler) Update(c *gin.Context) {
	var req settingUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.Update(c.Request.Context(), c.Param("key"), req.Value, middleware.UserNameFromCtx(c)); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}
