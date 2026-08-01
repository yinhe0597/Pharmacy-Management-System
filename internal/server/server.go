// Package server 组装依赖并构建 Gin 引擎。
package server

import (
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"

	"yaofang/internal/config"
	"yaofang/internal/domain/enum"
	"yaofang/internal/handler"
	"yaofang/internal/middleware"
	"yaofang/internal/pkg/auth"
	"yaofang/internal/service"
	"yaofang/internal/service/patient"
	"yaofang/internal/service/port"
	"yaofang/internal/service/pricing"

	_ "yaofang/docs" // swag 生成的文档
)

// App 应用依赖容器。
type App struct {
	cfg *config.Config
	jwt *auth.Manager

	auth         *service.AuthService
	drug         *service.DrugService
	supplier     *service.SupplierService
	inventory    *service.InventoryService
	purchase     *service.PurchaseService
	prescription *service.PrescriptionService
	special      *service.SpecialDrugService
	pharma       *service.PharmaService
	report       *service.ReportService

	patientService port.IPatientService
	pricingService port.IPricingService
}

// NewApp 构建应用依赖。
func NewApp(cfg *config.Config, db *gorm.DB) *App {
	jwtMgr := auth.NewManager(cfg.Auth.JWTSecret, cfg.Auth.TokenTTL)
	if cfg.Auth.TokenTTL <= 0 {
		cfg.Auth.TokenTTL = 720 * time.Hour
	}

	inv := service.NewInventoryService(db)
	special := service.NewSpecialDrugService(db)

	return &App{
		cfg:            cfg,
		jwt:            jwtMgr,
		auth:           service.NewAuthService(db, jwtMgr),
		drug:           service.NewDrugService(db),
		supplier:       service.NewSupplierService(db),
		inventory:      inv,
		purchase:       service.NewPurchaseService(db, inv),
		prescription:   service.NewPrescriptionService(db, inv, special),
		special:        special,
		pharma:         service.NewPharmaService(db),
		report:         service.NewReportService(db),
		patientService: patient.NewSimplePatientService(),
		pricingService: pricing.NewSimplePricingService(db),
	}
}

// Inventory 返回库存服务（供调度器使用）。
func (a *App) Inventory() *service.InventoryService { return a.inventory }

// Engine 构建 Gin 引擎并注册全部路由。
func (a *App) Engine() *gin.Engine {
	if a.cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(middleware.Recover(), middleware.RequestID(), middleware.Logger())

	// 探活
	r.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
	r.GET("/readyz", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ready"}) })
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := r.Group("/api/v1")
	authed := v1.Group("", middleware.Auth(a.jwt))
	adminOnly := v1.Group("", middleware.Auth(a.jwt), middleware.RequireRoles(enum.RoleAdmin))

	handler.NewAuthHandler(a.auth).Register(v1, authed, adminOnly)
	handler.NewDrugHandler(a.drug).Register(authed, authed, adminOnly)
	handler.NewSupplierHandler(a.supplier).Register(authed, authed, adminOnly)
	handler.NewInventoryHandler(a.inventory).Register(authed, authed, adminOnly)
	handler.NewPurchaseHandler(a.purchase, a.inventory).Register(authed, authed, adminOnly)
	handler.NewPrescriptionHandler(a.prescription).Register(authed, authed, adminOnly)
	handler.NewSpecialDrugHandler(a.special).Register(authed, authed, adminOnly)
	handler.NewPharmaServiceHandler(a.pharma).Register(authed, authed, adminOnly)
	handler.NewReportHandler(a.report).Register(authed, authed, adminOnly)

	return r
}
