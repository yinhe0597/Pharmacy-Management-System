// Package errs 定义业务错误类型与统一错误码。
// 错误码分段：9xxx 通用，1xxx 药品/供应商，2xxx 库存/采购，
// 3xxx 处方，4xxx 特殊药品，5xxx 药学服务/报表。
package errs

import (
	"errors"
	"net/http"
)

// Error 是统一业务错误。Code 为业务错误码，HTTP 为映射的 HTTP 状态码。
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	HTTP    int    `json:"-"`
}

func (e *Error) Error() string { return e.Message }

// New 构造业务错误。
func New(code int, message string, httpStatus int) *Error {
	return &Error{Code: code, Message: message, HTTP: httpStatus}
}

// 通用错误（9xxx）
var (
	ErrBadRequest    = New(9001, "参数错误", http.StatusBadRequest)
	ErrUnauthorized  = New(9002, "未认证", http.StatusUnauthorized)
	ErrForbidden     = New(9003, "无权限", http.StatusForbidden)
	ErrNotFound      = New(9004, "资源不存在", http.StatusNotFound)
	ErrStateConflict = New(9005, "状态冲突，请刷新后重试", http.StatusConflict)
	ErrInternal      = New(9500, "系统异常", http.StatusInternalServerError)
)

// 药品/供应商（1xxx）
var (
	ErrDrugCodeExists       = New(1001, "药品编码已存在", http.StatusConflict)
	ErrDrugFrozen           = New(1002, "药品已冻结，存在关联单据，无法删除", http.StatusConflict)
	ErrDrugInactive         = New(1003, "药品已停用", http.StatusBadRequest)
	ErrSupplierCodeExists   = New(1004, "供应商编码已存在", http.StatusConflict)
	ErrDuplicateInteraction = New(1005, "配伍禁忌记录已存在", http.StatusConflict)
)

// 库存/采购（2xxx）
var (
	ErrStockNotEnough        = New(2001, "库存不足", http.StatusConflict)
	ErrBatchLocked           = New(2002, "批次已过期或已锁定", http.StatusConflict)
	ErrNegativeStock         = New(2003, "库存扣减失败，请重试", http.StatusConflict)
	ErrReservationConflict   = New(2004, "预占冲突，请刷新后重试", http.StatusConflict)
	ErrStocktakingInProgress = New(2005, "盘点进行中，禁止库存变动", http.StatusConflict)
	ErrSplitNotAllowed       = New(2006, "该药品不允许拆零", http.StatusBadRequest)
	ErrQCFailed              = New(2007, "存在质检不合格项，禁止入库", http.StatusConflict)
	ErrExpiredLot            = New(2008, "批次已过期，禁止操作", http.StatusConflict)
)

// 处方（3xxx）
var (
	ErrPrescriptionState = New(3001, "处方状态不允许该操作", http.StatusConflict)
	ErrDoseExceeded      = New(3002, "极量超限", http.StatusBadRequest)
	ErrInteraction       = New(3003, "配伍禁忌", http.StatusBadRequest)
	ErrDuplicateDrug     = New(3004, "重复用药提醒", http.StatusBadRequest)
	ErrBatchAllocation   = New(3005, "批次分配不足，无法调配", http.StatusConflict)
	ErrItemNotFound      = New(3006, "处方明细不存在", http.StatusNotFound)
	ErrReturnExceeded    = New(3007, "退药数量超过已发数量", http.StatusBadRequest)
)

// 特殊药品（4xxx）
var (
	ErrSpecialDrugRequired = New(4001, "麻精处方必填项缺失（患者卡号/诊断）", http.StatusBadRequest)
	ErrSpecialDrugLimit    = New(4002, "单张处方剂量超过限量", http.StatusBadRequest)
)

// 药学服务/报表（5xxx）
var (
	ErrReportPeriod = New(5001, "报表参数错误", http.StatusBadRequest)
)

// Is 判断 err 是否为目标业务错误（按 Code 匹配）。
func Is(err error, target *Error) bool {
	var e *Error
	if errors.As(err, &e) {
		return e.Code == target.Code
	}
	return false
}

// HTTPStatus 返回错误对应的 HTTP 状态码；未知错误返回 500。
func HTTPStatus(err error) int {
	var e *Error
	if errors.As(err, &e) {
		return e.HTTP
	}
	return http.StatusInternalServerError
}
