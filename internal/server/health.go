package server

import (
	"context"
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"

	"yaofang/internal/version"
)

// readinessTimeout DB 连通性探测超时。探测必须快速失败，避免探针拖垮请求。
const readinessTimeout = 2 * time.Second

// healthz 存活探针：进程在且未进入关闭流程即 200。
// 编排系统（K8s livenessProbe / 负载均衡）用它判断是否需要重启/摘除实例。
func (a *App) healthz(c *gin.Context) {
	if a.shuttingDown.Load() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "shutting_down",
			"reason": "graceful shutdown in progress",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":         "ok",
		"version":        version.Version,
		"uptime_seconds": int64(time.Since(a.startedAt).Seconds()),
	})
}

// readyz 就绪探针：校验数据库连通性，未就绪返回 503。
// 容器编排用它在流量接入前确认依赖可用；数据库不可达时自动摘除流量。
func (a *App) readyz(c *gin.Context) {
	sqlDB, err := a.db.DB()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unavailable",
			"reason": "sql_db_unavailable",
		})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), readinessTimeout)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unavailable",
			"reason": "database_unreachable",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}

// versionInfo 返回构建版本信息（/version）。
func (a *App) versionInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version":    version.Version,
		"commit":     version.Commit,
		"build_time": version.BuildTime,
		"go_version": runtime.Version(),
	})
}
