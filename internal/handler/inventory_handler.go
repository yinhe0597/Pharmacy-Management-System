package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"yaofang/internal/middleware"
	"yaofang/internal/model"
	"yaofang/internal/pkg/errs"
	"yaofang/internal/pkg/pagination"
	"yaofang/internal/repository"
	"yaofang/internal/service"
)

// InventoryHandler 库存接口。
type InventoryHandler struct {
	svc *service.InventoryService
}

// NewInventoryHandler 构建库存 Handler。
func NewInventoryHandler(svc *service.InventoryService) *InventoryHandler {
	return &InventoryHandler{svc: svc}
}

// Register 注册路由。
func (h *InventoryHandler) Register(r *gin.RouterGroup, _ *gin.RouterGroup, adminOnly *gin.RouterGroup) {
	r.GET("/inventory", h.List)
	r.GET("/inventory/:id", h.GetDetail)
	r.GET("/inventory/locations", h.ListLocations)
	adminOnly.POST("/inventory/locations", h.CreateLocation)
	adminOnly.PUT("/inventory/locations/:id", h.UpdateLocation)
	r.POST("/inventory/transfer", h.Transfer)
	r.POST("/inventory/split", h.Split)
	r.POST("/inventory/split-units", h.SplitUnits)
	r.POST("/inventory/adjust", h.Adjust)
	r.POST("/inventory/stock-in", h.StockIn)
	r.POST("/inventory/requisition", h.Requisition)
	r.GET("/inventory/transactions", h.ListTransactions)
	r.GET("/inventory/expiry-warnings", h.ListExpiryWarnings)
	r.GET("/inventory/stock-warnings", h.ListStockWarnings)
	r.POST("/inventory/alerts/:id/resolve", h.ResolveAlert)
	r.GET("/inventory/drugs/:id/availability", h.Availability)
	r.GET("/inventory/stocktakes", h.ListStocktakes)
	r.POST("/inventory/stocktakes", h.CreateStocktake)
	r.GET("/inventory/stocktakes/:id", h.GetStocktake)
	r.POST("/inventory/stocktakes/:id/start", h.StartStocktake)
	r.POST("/inventory/stocktakes/:id/items", h.EnterCounted)
	r.POST("/inventory/stocktakes/:id/adjust", h.AdjustStocktake)
	r.POST("/inventory/stocktakes/:id/complete", h.CompleteStocktake)
	r.POST("/inventory/stocktakes/:id/cancel", h.CancelStocktake)
}

// List godoc
// @Summary 库存列表
// @Tags inventory
// @Security BearerAuth
// @Param drug_id query int false "药品ID"
// @Param location_id query int false "库房ID"
// @Param batch_no query string false "批号"
// @Param status query int false "状态"
// @Param page query int false "页码"
// @Param page_size query int false "每页条数"
// @Success 200 {object} Body
// @Router /inventory [get]
func (h *InventoryHandler) List(c *gin.Context) {
	var q pagination.Query
	if err := c.ShouldBindQuery(&q); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	q.Normalize()
	f := repository.InventoryListFilter{
		DrugID:     int64(atoi(c.Query("drug_id"))),
		LocationID: int64(atoi(c.Query("location_id"))),
		BatchNo:    c.Query("batch_no"),
		Status:     atoi(c.Query("status")),
	}
	list, total, err := h.svc.ListInventory(c.Request.Context(), f, q.Page, q.PageSize)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, pagination.Of(list, total, &q))
}

// GetDetail godoc
// @Summary 批次库存详情
// @Tags inventory
// @Security BearerAuth
// @Param id path int true "库存ID"
// @Success 200 {object} Body
// @Router /inventory/{id} [get]
func (h *InventoryHandler) GetDetail(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	inv, err := h.svc.GetInventoryDetail(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, inv)
}

// ListLocations godoc
// @Summary 库房列表
// @Tags inventory
// @Security BearerAuth
// @Success 200 {object} Body
// @Router /inventory/locations [get]
func (h *InventoryHandler) ListLocations(c *gin.Context) {
	list, err := h.svc.ListLocations(c.Request.Context())
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, list)
}

// CreateLocation godoc
// @Summary 新建库房
// @Tags inventory
// @Accept json
// @Security BearerAuth
// @Param body body model.InventoryLocation true "库房"
// @Success 200 {object} Body
// @Router /inventory/locations [post]
func (h *InventoryHandler) CreateLocation(c *gin.Context) {
	var l model.InventoryLocation
	if err := c.ShouldBindJSON(&l); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.CreateLocation(c.Request.Context(), &l); err != nil {
		Error(c, err)
		return
	}
	OK(c, l)
}

// UpdateLocation godoc
// @Summary 更新库房
// @Tags inventory
// @Security BearerAuth
// @Param id path int true "库房ID"
// @Param body body model.InventoryLocation true "库房"
// @Success 200 {object} Body
// @Router /inventory/locations/{id} [put]
func (h *InventoryHandler) UpdateLocation(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	var l model.InventoryLocation
	if err := c.ShouldBindJSON(&l); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.UpdateLocation(c.Request.Context(), id, &l); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

type transferRequest struct {
	FromLocationID int64                  `json:"from_location_id" binding:"required"`
	ToLocationID   int64                  `json:"to_location_id" binding:"required"`
	Items          []service.TransferItem `json:"items" binding:"required"`
	Remarks        string                 `json:"remarks"`
}

// Transfer godoc
// @Summary 库存调拨
// @Tags inventory
// @Accept json
// @Security BearerAuth
// @Param body body transferRequest true "调拨请求"
// @Success 200 {object} Body
// @Router /inventory/transfer [post]
func (h *InventoryHandler) Transfer(c *gin.Context) {
	var req transferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.Transfer(c.Request.Context(), req.FromLocationID, req.ToLocationID, req.Items,
		middleware.UserIDFromCtx(c), middleware.UserNameFromCtx(c)); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

type splitRequest struct {
	InventoryID int64 `json:"inventory_id" binding:"required"`
	Packs       int64 `json:"packs" binding:"required"`
}

// Split godoc
// @Summary 拆零
// @Tags inventory
// @Accept json
// @Security BearerAuth
// @Param body body splitRequest true "拆零请求"
// @Success 200 {object} Body
// @Router /inventory/split [post]
func (h *InventoryHandler) Split(c *gin.Context) {
	var req splitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.Split(c.Request.Context(), service.SplitRequest{InventoryID: req.InventoryID, Packs: req.Packs},
		middleware.UserIDFromCtx(c), middleware.UserNameFromCtx(c)); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

type splitUnitsRequest struct {
	InventoryID int64 `json:"inventory_id" binding:"required"`
	Boxes       int64 `json:"boxes" binding:"required"`
	Units       int64 `json:"units" binding:"required"`
	Damaged     int64 `json:"damaged"`
}

// SplitUnits godoc
// @Summary 按片拆零（开盒零头入账 + 破损报损）
// @Tags inventory
// @Accept json
// @Security BearerAuth
// @Param body body splitUnitsRequest true "拆零请求（units+damaged = boxes×包装含量）"
// @Success 200 {object} Body
// @Router /inventory/split-units [post]
func (h *InventoryHandler) SplitUnits(c *gin.Context) {
	var req splitUnitsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.SplitUnits(c.Request.Context(), service.SplitUnitsRequest{
		InventoryID: req.InventoryID, Boxes: req.Boxes, Units: req.Units, Damaged: req.Damaged,
	}, middleware.UserIDFromCtx(c), middleware.UserNameFromCtx(c)); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

type adjustRequest struct {
	InventoryID int64  `json:"inventory_id" binding:"required"`
	Quantity    int64  `json:"quantity" binding:"required"`
	Reason      string `json:"reason"`
}

// Adjust godoc
// @Summary 库存调整（报损/修正）
// @Tags inventory
// @Accept json
// @Security BearerAuth
// @Param body body adjustRequest true "调整请求"
// @Success 200 {object} Body
// @Router /inventory/adjust [post]
func (h *InventoryHandler) Adjust(c *gin.Context) {
	var req adjustRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.Adjust(c.Request.Context(), service.AdjustRequest{InventoryID: req.InventoryID, Quantity: req.Quantity, Reason: req.Reason},
		middleware.UserIDFromCtx(c), middleware.UserNameFromCtx(c)); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

type stockInRequest struct {
	Entries []service.StockEntry `json:"entries" binding:"required"`
}

// StockIn godoc
// @Summary 其他入库
// @Tags inventory
// @Accept json
// @Security BearerAuth
// @Param body body stockInRequest true "入库条目"
// @Success 200 {object} Body
// @Router /inventory/stock-in [post]
func (h *InventoryHandler) StockIn(c *gin.Context) {
	var req stockInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.StockIn(c.Request.Context(), req.Entries, middleware.UserIDFromCtx(c), middleware.UserNameFromCtx(c)); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

type requisitionRequest struct {
	DrugID     int64  `json:"drug_id" binding:"required"`
	LocationID int64  `json:"location_id" binding:"required"`
	Quantity   int64  `json:"quantity" binding:"required"`
	Reason     string `json:"reason"` // 领用原因
}

// Requisition godoc
// @Summary 领用出库（医护内部消耗，不计费）
// @Tags inventory
// @Accept json
// @Security BearerAuth
// @Param body body requisitionRequest true "领用信息"
// @Success 200 {object} Body
// @Router /inventory/requisition [post]
func (h *InventoryHandler) Requisition(c *gin.Context) {
	var req requisitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.Requisition(c.Request.Context(), req.DrugID, req.LocationID, req.Quantity, req.Reason,
		middleware.UserIDFromCtx(c), middleware.UserNameFromCtx(c)); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

// ListTransactions godoc
// @Summary 库存流水
// @Tags inventory
// @Security BearerAuth
// @Param drug_id query int false "药品ID"
// @Param location_id query int false "库房ID"
// @Param txn_type query string false "流水类型"
// @Param start query string false "开始时间"
// @Param end query string false "结束时间"
// @Param page query int false "页码"
// @Param page_size query int false "每页条数"
// @Success 200 {object} Body
// @Router /inventory/transactions [get]
func (h *InventoryHandler) ListTransactions(c *gin.Context) {
	var q pagination.Query
	if err := c.ShouldBindQuery(&q); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	q.Normalize()
	f := repository.TxnListFilter{
		DrugID:     int64(atoi(c.Query("drug_id"))),
		LocationID: int64(atoi(c.Query("location_id"))),
		TxnType:    c.Query("txn_type"),
		Start:      parseTime(c.Query("start")),
		End:        parseTime(c.Query("end")),
	}
	list, total, err := h.svc.ListTransactions(c.Request.Context(), f, q.Page, q.PageSize)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, pagination.Of(list, total, &q))
}

// ListExpiryWarnings godoc
// @Summary 效期预警列表
// @Tags inventory
// @Security BearerAuth
// @Param page query int false "页码"
// @Param page_size query int false "每页条数"
// @Success 200 {object} Body
// @Router /inventory/expiry-warnings [get]
func (h *InventoryHandler) ListExpiryWarnings(c *gin.Context) {
	h.listAlerts(c, "expiry")
}

// ListStockWarnings godoc
// @Summary 库存下限预警列表
// @Tags inventory
// @Security BearerAuth
// @Param page query int false "页码"
// @Param page_size query int false "每页条数"
// @Success 200 {object} Body
// @Router /inventory/stock-warnings [get]
func (h *InventoryHandler) ListStockWarnings(c *gin.Context) {
	h.listAlerts(c, "below_min")
}

func (h *InventoryHandler) listAlerts(c *gin.Context, alertType string) {
	var q pagination.Query
	if err := c.ShouldBindQuery(&q); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	q.Normalize()
	list, total, err := h.svc.ListAlerts(c.Request.Context(), alertType, "open", q.Page, q.PageSize)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, pagination.Of(list, total, &q))
}

// ResolveAlert godoc
// @Summary 处理预警
// @Tags inventory
// @Security BearerAuth
// @Param id path int true "预警ID"
// @Success 200 {object} Body
// @Router /inventory/alerts/{id}/resolve [post]
func (h *InventoryHandler) ResolveAlert(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.ResolveAlert(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

// Availability godoc
// @Summary 药品可用库存
// @Tags inventory
// @Security BearerAuth
// @Param id path int true "药品ID"
// @Success 200 {object} Body
// @Router /inventory/drugs/{id}/availability [get]
func (h *InventoryHandler) Availability(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	list, err := h.svc.GetDrugAvailability(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, list)
}

// AvailabilityAlias 药品可用库存别名（兼容设计文档路径 /drugs/:id/availability）。
func (h *InventoryHandler) AvailabilityAlias(c *gin.Context) {
	h.Availability(c)
}

// CreateStocktake godoc
// @Summary 创建盘点单
// @Tags inventory
// @Accept json
// @Security BearerAuth
// @Param body body createStocktakeRequest true "盘点请求"
// @Success 200 {object} Body
// @Router /inventory/stocktakes [post]
func (h *InventoryHandler) CreateStocktake(c *gin.Context) {
	var req createStocktakeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	st, err := h.svc.CreateStocktake(c.Request.Context(), req.LocationID, req.Type, middleware.UserIDFromCtx(c))
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, st)
}

type createStocktakeRequest struct {
	LocationID int64 `json:"location_id" binding:"required"`
	Type       int   `json:"type" binding:"required,oneof=1 2"` // 1周期 2动态
}

// ListStocktakes godoc
// @Summary 盘点单列表
// @Tags inventory
// @Security BearerAuth
// @Param status query string false "状态"
// @Param page query int false "页码"
// @Param page_size query int false "每页条数"
// @Success 200 {object} Body
// @Router /inventory/stocktakes [get]
func (h *InventoryHandler) ListStocktakes(c *gin.Context) {
	var q pagination.Query
	if err := c.ShouldBindQuery(&q); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	q.Normalize()
	list, total, err := h.svc.ListStocktakes(c.Request.Context(), c.Query("status"), q.Page, q.PageSize)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, pagination.Of(list, total, &q))
}

// GetStocktake godoc
// @Summary 盘点单详情
// @Tags inventory
// @Security BearerAuth
// @Param id path int true "盘点单ID"
// @Success 200 {object} Body
// @Router /inventory/stocktakes/{id} [get]
func (h *InventoryHandler) GetStocktake(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	detail, err := h.svc.GetStocktake(c.Request.Context(), id)
	if err != nil {
		Error(c, err)
		return
	}
	OK(c, detail)
}

// StartStocktake godoc
// @Summary 开始盘点
// @Tags inventory
// @Security BearerAuth
// @Param id path int true "盘点单ID"
// @Success 200 {object} Body
// @Router /inventory/stocktakes/{id}/start [post]
func (h *InventoryHandler) StartStocktake(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.StartStocktake(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

type enterCountedRequest struct {
	Items []service.CountedItem `json:"items" binding:"required"`
}

// EnterCounted godoc
// @Summary 录入实盘数量
// @Tags inventory
// @Accept json
// @Security BearerAuth
// @Param id path int true "盘点单ID"
// @Param body body enterCountedRequest true "实盘明细"
// @Success 200 {object} Body
// @Router /inventory/stocktakes/{id}/items [post]
func (h *InventoryHandler) EnterCounted(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	var req enterCountedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.EnterCounted(c.Request.Context(), id, req.Items); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

// AdjustStocktake godoc
// @Summary 确认差异并调整库存
// @Tags inventory
// @Security BearerAuth
// @Param id path int true "盘点单ID"
// @Success 200 {object} Body
// @Router /inventory/stocktakes/{id}/adjust [post]
func (h *InventoryHandler) AdjustStocktake(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.AdjustStocktake(c.Request.Context(), id, middleware.UserIDFromCtx(c), middleware.UserNameFromCtx(c)); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

// CompleteStocktake godoc
// @Summary 完成盘点
// @Tags inventory
// @Security BearerAuth
// @Param id path int true "盘点单ID"
// @Success 200 {object} Body
// @Router /inventory/stocktakes/{id}/complete [post]
func (h *InventoryHandler) CompleteStocktake(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.CompleteStocktake(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

// CancelStocktake godoc
// @Summary 取消盘点
// @Tags inventory
// @Security BearerAuth
// @Param id path int true "盘点单ID"
// @Success 200 {object} Body
// @Router /inventory/stocktakes/{id}/cancel [post]
func (h *InventoryHandler) CancelStocktake(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		Error(c, errs.ErrBadRequest)
		return
	}
	if err := h.svc.CancelStocktake(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}
	OK(c, nil)
}

func parseTime(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil
	}
	return &t
}
