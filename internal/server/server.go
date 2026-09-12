// Package server 组装依赖并构建 Gin 引擎。
package server

import (
	"context"
	"log/slog"
	"sync/atomic"
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
	"yaofang/internal/pkg/errs"
	"yaofang/internal/repository"
	"yaofang/internal/service"
	"yaofang/internal/service/patient"
	"yaofang/internal/service/port"
	"yaofang/internal/service/pricing"

	_ "yaofang/docs" // swag 生成的文档
)

// App 应用依赖容器。
type App struct {
	cfg *config.Config
	db  *gorm.DB
	jwt *auth.Manager

	startedAt    time.Time
	shuttingDown atomic.Bool

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
	visits       *service.VisitService
	records      *service.MedicalRecordService
	charges      *service.ChargeService
	settings     *service.SettingService

	patientService port.IPatientService
	pricingService port.IPricingService
}

// NewApp 构建应用依赖。
func NewApp(cfg *config.Config, db *gorm.DB) *App {
	// TTL 兜底必须在构建 Manager 之前（否则显式配 0 会签发即时过期 token）
	if cfg.Auth.TokenTTL <= 0 {
		cfg.Auth.TokenTTL = 720 * time.Hour
	}
	jwtMgr := auth.NewManager(cfg.Auth.JWTSecret, cfg.Auth.TokenTTL)

	inv := service.NewInventoryService(db)
	special := service.NewSpecialDrugService(db)
	interSvc := service.NewInteractionService(db)
	clinicalSvc := service.NewClinicalService(db)
	logSvc := service.NewOperationLogService(db)
	patientSvc := patient.NewPatientService(db)
	pricer := pricing.NewSimplePricingService(db)

	return &App{
		cfg:            cfg,
		db:             db,
		jwt:            jwtMgr,
		startedAt:      time.Now(),
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
		pricingService: pricer,
		visits:         service.NewVisitService(db),
		records:        service.NewMedicalRecordService(db),
		charges:        service.NewChargeService(db, pricer),
		settings:       service.NewSettingService(db),
	}
}

// Inventory 返回库存服务（供调度器使用）。
func (a *App) Inventory() *service.InventoryService { return a.inventory }

// BeginShutdown 标记服务进入关闭流程（/healthz 随即返回 503，便于负载均衡/编排摘流量）。
// 在收到退出信号后、调用 http.Server.Shutdown 之前调用。
func (a *App) BeginShutdown() { a.shuttingDown.Store(true) }

// Engine 构建 Gin 引擎并注册全部路由。
func (a *App) Engine() *gin.Engine {
	if a.cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	// 可信代理：默认不信任任何代理头（ClientIP 取直连地址），防止伪造 X-Forwarded-For 绕过登录限速；
	// 部署在 Nginx/网关之后时经 server.trusted_proxies / YF_SERVER_TRUSTED_PROXIES 配置代理网段。
	if err := r.SetTrustedProxies(a.cfg.Server.TrustedProxies); err != nil {
		slog.Warn("设置可信代理失败，回退为不信任任何代理", "err", err)
		_ = r.SetTrustedProxies(nil)
	}
	r.Use(middleware.Recover(), middleware.RequestID(), middleware.Logger(), middleware.CORS(a.cfg.Server.CORSAllowOrigins))
	if a.cfg.Server.Mode == "release" {
		for _, o := range a.cfg.Server.CORSAllowOrigins {
			if o == "*" {
				slog.Warn("release 模式下 CORS 放行任意来源（*），建议在 server.cors_allow_origins 限定具体域名")
				break
			}
		}
	}

	// 健康检查（无鉴权，供负载均衡 / 容器编排 / K8s 探针使用）
	r.GET("/healthz", a.healthz) // 存活：进程存活即 200，关闭中 503
	r.GET("/readyz", a.readyz)   // 就绪：依赖（数据库）连通才 200，否则 503
	r.GET("/version", a.versionInfo)
	// Swagger 仅在非 release 模式开放（接口契约泄露面最小化）
	if a.cfg.Server.Mode != "release" {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	// 用户状态复查：token 有效但账号已被停用/删除/降权时，按数据库当前状态与角色处理（吊销能力）
	users := repository.NewUserRepo(a.db)
	checkActive := middleware.UserStateChecker(func(ctx context.Context, userID int64) (string, error) {
		u, err := users.GetByID(ctx, userID)
		if err != nil {
			return "", err
		}
		if u.Status != 1 {
			return "", errs.ErrUnauthorized
		}
		return u.Role, nil
	})
	authMW := func() gin.HandlerFunc { return middleware.Auth(a.jwt, checkActive) }
	// 登录限速：同一「IP+用户名」每分钟最多 5 次尝试（防暴力破解）
	loginRate := middleware.LoginRateLimit(5, time.Minute)

	v1 := r.Group("/api/v1")
	authed := v1.Group("", authMW())
	// 计费查看 = 药房人员 ∪ 报表权限（docs/15 M3）
	billingRoles := append(append([]string{}, enum.PharmacyStaff...), enum.ReportAccess...)
	// 角色分组（docs/03 §2 角色矩阵）
	groups := handler.Groups{
		Public:      v1,
		Authed:      authed,
		DrugAdmin:   v1.Group("", authMW(), middleware.RequireRoles(enum.DrugAdmin...)),
		Pharmacy:    v1.Group("", authMW(), middleware.RequireRoles(enum.PharmacyStaff...)),
		Clinical:    v1.Group("", authMW(), middleware.RequireRoles(enum.ClinicalStaff...)),
		Purchase:    v1.Group("", authMW(), middleware.RequireRoles(enum.PurchaseStaff...)),
		Report:      v1.Group("", authMW(), middleware.RequireRoles(enum.ReportAccess...)),
		Billing:     v1.Group("", authMW(), middleware.RequireRoles(billingRoles...)),
		Patient:     v1.Group("", authMW(), middleware.RequireRoles(enum.PatientAdmin...)),
		PatientRead: v1.Group("", authMW(), middleware.RequireRoles(enum.PatientRead...)),
		Charge:      v1.Group("", authMW(), middleware.RequireRoles(enum.ChargeStaff...)),
		UserAdmin:   v1.Group("", authMW(), middleware.RequireRoles(enum.UserAdmin...)),
	}

	handler.NewAuthHandler(a.auth, a.logSvc, loginRate).Register(groups)
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
	handler.NewVisitHandler(a.visits, a.records, a.charges).Register(groups)
	handler.NewSettingHandler(a.settings).Register(groups)

	return r
}
