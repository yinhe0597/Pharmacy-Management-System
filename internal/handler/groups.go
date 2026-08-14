package handler

import "github.com/gin-gonic/gin"

// Groups 路由分组容器。按职责把鉴权中间件分组注入各 Handler 的 Register，
// 使每个业务模块的写操作受角色约束（docs/03 §2 角色矩阵）。
type Groups struct {
	// Public 无需鉴权（/api/v1，仅登录等白名单）
	Public *gin.RouterGroup
	// Authed 任意已登录用户（只读查询）
	Authed *gin.RouterGroup
	// DrugAdmin 药品/分类/配伍/交互规则/特殊药品目录 写权限
	DrugAdmin *gin.RouterGroup
	// Pharmacy 库存写/处方执行/药学服务/计费录入（药房工作人员）
	Pharmacy *gin.RouterGroup
	// Clinical 处方开立权限（医生/药师/药房主任/管理员）
	Clinical *gin.RouterGroup
	// Purchase 采购单/收货/供应商权限
	Purchase *gin.RouterGroup
	// Report 报表访问权限
	Report *gin.RouterGroup
	// UserAdmin 用户管理权限
	UserAdmin *gin.RouterGroup
}
