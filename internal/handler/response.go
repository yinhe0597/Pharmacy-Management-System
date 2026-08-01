// Package handler 提供 HTTP 处理器与统一响应封装。
package handler

import (
	"errors"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"

	"yaofang/internal/middleware"
	"yaofang/internal/pkg/errs"
)

// Body 统一响应信封。
type Body struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// OK 成功响应。
func OK(c *gin.Context, data any) {
	c.JSON(200, Body{Code: 0, Message: "ok", Data: data})
}

// Error 统一错误响应：业务错误按错误码/HTTP 映射，未知错误返回 500。
func Error(c *gin.Context, err error) {
	var e *errs.Error
	if errors.As(err, &e) {
		c.JSON(e.HTTP, Body{Code: e.Code, Message: e.Message, Data: nil})
		return
	}
	slog.Error("internal_error", "err", err, "request_id", middleware.RequestIDFromCtx(c))
	c.JSON(errs.ErrInternal.HTTP, Body{Code: errs.ErrInternal.Code, Message: errs.ErrInternal.Message, Data: nil})
}

// requiredPeriod 解析并校验报表期间参数（start/end，RFC3339）。
func requiredPeriod(c *gin.Context) (*time.Time, *time.Time, error) {
	start, end := parseTime(c.Query("start")), parseTime(c.Query("end"))
	if start == nil || end == nil {
		return nil, nil, errs.ErrReportPeriod
	}
	return start, end, nil
}

// nowToday 当前时间。
func nowToday() time.Time { return time.Now() }
