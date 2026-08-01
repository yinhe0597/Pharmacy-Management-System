package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"yaofang/internal/middleware"
	"yaofang/internal/model"
	"yaofang/internal/pkg/errs"
	"yaofang/internal/pkg/pagination"
	"yaofang/internal/service"
)

// AuthHandler 用户与鉴权接口。
type AuthHandler struct {
	svc *service.AuthService
}

// NewAuthHandler 构建鉴权 Handler。
func NewAuthHandler(svc *service.AuthService) *AuthHandler { return &AuthHandler{svc: svc} }

// Register 注册路由。
func (h *AuthHandler) Register(r *gin.RouterGroup, authed *gin.RouterGroup, adminOnly *gin.RouterGroup) {
	r.POST("/auth/login", h.Login)
	authed.GET("/auth/profile", h.Profile)
	adminOnly.POST("/users", h.CreateUser)
	adminOnly.PUT("/users/:id", h.UpdateUser)
	adminOnly.DELETE("/users/:id", h.DeleteUser)
	adminOnly.GET("/users", h.ListUsers)
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login godoc
// @Summary 登录
// @Tags auth
// @Accept json
// @Produce json
// @Param body body loginRequest true "账号密码"
// @Success 200 {object} Body
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	token, user, err := h.svc.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, gin.H{"token": token, "user": user})
}

// Profile godoc
// @Summary 当前用户信息
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} Body
// @Router /auth/profile [get]
func (h *AuthHandler) Profile(c *gin.Context) {
	u, err := h.svc.Profile(c.Request.Context(), middleware.UserIDFromCtx(c))
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, u)
}

type createUserRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Name     string `json:"name" binding:"required"`
	Role     string `json:"role" binding:"required"`
	Phone    string `json:"phone"`
}

// CreateUser godoc
// @Summary 新建用户
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body createUserRequest true "用户信息"
// @Success 200 {object} Body
// @Router /users [post]
func (h *AuthHandler) CreateUser(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	u, err := h.svc.CreateUser(c.Request.Context(), &model.User{Username: req.Username, Name: req.Name, Role: req.Role, Phone: req.Phone}, req.Password)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, u)
}

type updateUserRequest struct {
	Name     string `json:"name"`
	Role     string `json:"role"`
	Phone    string `json:"phone"`
	Status   int    `json:"status"`
	Password string `json:"password"`
}

// UpdateUser godoc
// @Summary 更新用户
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "用户ID"
// @Param body body updateUserRequest true "用户信息"
// @Success 200 {object} Body
// @Router /users/{id} [put]
func (h *AuthHandler) UpdateUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	var req updateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.UpdateUser(c.Request.Context(), id, req.Name, req.Role, req.Phone, req.Status, req.Password); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

// DeleteUser godoc
// @Summary 删除用户
// @Tags auth
// @Security BearerAuth
// @Param id path int true "用户ID"
// @Success 200 {object} Body
// @Router /users/{id} [delete]
func (h *AuthHandler) DeleteUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.DeleteUser(c.Request.Context(), id, middleware.UserIDFromCtx(c)); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

// ListUsers godoc
// @Summary 用户列表
// @Tags auth
// @Security BearerAuth
// @Param role query string false "角色"
// @Param keyword query string false "关键字"
// @Param page query int false "页码"
// @Param page_size query int false "每页条数"
// @Success 200 {object} Body
// @Router /users [get]
func (h *AuthHandler) ListUsers(c *gin.Context) {
	var q pagination.Query
	if err := c.ShouldBindQuery(&q); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	q.Normalize()
	list, total, err := h.svc.ListUsers(c.Request.Context(), c.Query("role"), c.Query("keyword"), q.Page, q.PageSize)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, pagination.Of(list, total, &q))
}
