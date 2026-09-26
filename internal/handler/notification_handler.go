package handler

import (
	"github.com/gin-gonic/gin"

	"yaofang/internal/middleware"
	"yaofang/internal/pkg/errs"
	"yaofang/internal/pkg/pagination"
	"yaofang/internal/service"
)

// NotificationHandler 站内通知接口（所有登录用户）。
type NotificationHandler struct {
	svc *service.NotificationService
}

func NewNotificationHandler(svc *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{svc: svc}
}

func (h *NotificationHandler) Register(g Groups) {
	g.Authed.GET("/notifications", h.List)
	g.Authed.GET("/notifications/unread-count", h.UnreadCount)
	g.Authed.PUT("/notifications/:id/read", h.MarkRead)
	g.Authed.PUT("/notifications/read-all", h.MarkAllRead)
}

// List godoc
// @Summary 我的通知（全员广播 + 定向）
// @Tags notifications
// @Security BearerAuth
// @Param unread query bool false "仅未读"
// @Param page query int false "页码"
// @Param page_size query int false "每页条数"
// @Success 200 {object} Body
// @Router /notifications [get]
func (h *NotificationHandler) List(c *gin.Context) {
	var q pagination.Query
	if err := c.ShouldBindQuery(&q); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	q.Normalize()
	unread := c.Query("unread") == "true" || c.Query("unread") == "1"
	list, total, err := h.svc.List(c.Request.Context(), middleware.UserIDFromCtx(c), unread, q.Page, q.PageSize)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, pagination.Of(list, total, &q))
}

// UnreadCount godoc
// @Summary 我的未读通知数
// @Tags notifications
// @Security BearerAuth
// @Success 200 {object} Body
// @Router /notifications/unread-count [get]
func (h *NotificationHandler) UnreadCount(c *gin.Context) {
	n, err := h.svc.UnreadCount(c.Request.Context(), middleware.UserIDFromCtx(c))
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, gin.H{"unread": n})
}

// MarkRead godoc
// @Summary 标记单条通知已读
// @Tags notifications
// @Security BearerAuth
// @Param id path int true "通知ID"
// @Success 200 {object} Body
// @Router /notifications/{id}/read [put]
func (h *NotificationHandler) MarkRead(c *gin.Context) {
	ok, err := h.svc.MarkRead(c.Request.Context(), middleware.UserIDFromCtx(c), int64(atoi(c.Param("id"))))
	if err != nil {
		Error(c, err)
		return
	}
	if !ok {
		Error(c, errs.ErrNotFound)
		return
	}
	OK(c, nil)
}

// MarkAllRead godoc
// @Summary 全部通知标已读
// @Tags notifications
// @Security BearerAuth
// @Success 200 {object} Body
// @Router /notifications/read-all [put]
func (h *NotificationHandler) MarkAllRead(c *gin.Context) {
	n, err := h.svc.MarkAllRead(c.Request.Context(), middleware.UserIDFromCtx(c))
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, gin.H{"updated": n})
}
