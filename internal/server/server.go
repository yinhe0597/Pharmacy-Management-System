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
	interSvc     *service.InteractionService
	clinical     *service.ClinicalService
	logSvc       *service.OperationLogService
	reference    *service.ReferenceService
	patients     *patient.PatientService

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
	interSvc := service.NewInteractionService(db)
	clinicalSvc := service.NewClinicalService(db)
	logSvc := service.NewOperationLogService(db)
	patientSvc := patient.NewPatientService(db)

	return &App{
		cfg:            cfg,
		jwt:            jwtMgr,
		auth:           service.NewAuthService(db, jwtMgr),
		drug:           service.NewDrugService(db),
		supplier:       service.NewSupplierService(db),
		inventory:      inv,
		purchase:       service.NewPurchaseService(db, inv),
		prescription:   service.NewPrescriptionService(db, inv, special, interSvc, patientSvc, clinicalSvc),
		special:        special,
		pharma:         service.NewPharmaService(db),
		report:         service.NewReportService(db),
		interSvc:       interSvc,
		clinical:       clinicalSvc,
		logSvc:         logSvc,
		reference:      service.NewReferenceService(db),
		patients:       patientSvc,
		patientService: patientSvc,
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
	// 计费查看 = 药房人员 ∪ 报表权限（docs/15 M3）
	billingRoles := append(append([]string{}, enum.PharmacyStaff...), enum.ReportAccess...)
	// 角色分组（docs/03 §2 角色矩阵）
	groups := handler.Groups{
		Public:    v1,
		Authed:    authed,
		DrugAdmin: v1.Group("", middleware.Auth(a.jwt), middleware.RequireRoles(enum.DrugAdmin...)),
		Pharmacy:  v1.Group("", middleware.Auth(a.jwt), middleware.RequireRoles(enum.PharmacyStaff...)),
		Clinical:  v1.Group("", middleware.Auth(a.jwt), middleware.RequireRoles(enum.ClinicalStaff...)),
		Purchase:  v1.Group("", middleware.Auth(a.jwt), middleware.RequireRoles(enum.PurchaseStaff...)),
		Report:    v1.Group("", middleware.Auth(a.jwt), middleware.RequireRoles(enum.ReportAccess...)),
		Billing:   v1.Group("", middleware.Auth(a.jwt), middleware.RequireRoles(billingRoles...)),
		UserAdmin: v1.Group("", middleware.Auth(a.jwt), middleware.RequireRoles(enum.UserAdmin...)),
	}

	handler.NewAuthHandler(a.auth, a.logSvc).Register(groups)
	handler.NewDrugHandler(a.drug).Register(groups)
	handler.NewSupplierHandler(a.supplier).Register(groups)
	handler.NewInventoryHandler(a.inventory).Register(groups)
	// 别名：设计文档路径 /drugs/:id/availability → 实际实现在库存模块
	authed.GET("/drugs/:id/availability", handler.NewInventoryHandler(a.inventory).AvailabilityAlias)
	handler.NewPurchaseHandler(a.purchase, a.inventory).Register(groups)
	handler.NewPrescriptionHandler(a.prescription).Register(groups)
	handler.NewSpecialDrugHandler(a.special).Register(groups)
	// 别名：设计文档路径 /special-drugs/reports/usage → 功能已在报表模块
	authed.GET("/special-drugs/reports/usage", handler.NewReportHandler(a.report).SpecialDrugUsageAlias)
	handler.NewPharmaServiceHandler(a.pharma).Register(groups)
	handler.NewReportHandler(a.report).Register(groups)
	handler.NewInteractionHandler(a.interSvc).Register(groups)
	handler.NewClinicalHandler(a.clinical).Register(groups)
	handler.NewReferenceHandler(a.reference).Register(groups)
	handler.NewPatientHandler(a.patients).Register(groups)

	return r
}
