package model

import "time"

// OperationLog 操作日志（管理员审计用）。
type OperationLog struct {
	ID         int64     `gorm:"primaryKey" json:"id"`
	UserID     *int64    `json:"user_id"`
	Username   string    `gorm:"size:50" json:"username"`
	UserRole   string    `gorm:"size:50" json:"user_role"`
	Action     string    `gorm:"size:50;not null;index" json:"action"`
	Resource   string    `gorm:"size:100;not null;index" json:"resource"`
	ResourceID *int64    `json:"resource_id"`
	Method     string    `gorm:"size:10" json:"method"`
	Path       string    `gorm:"size:200" json:"path"`
	IP         string    `gorm:"size:50" json:"ip"`
	Detail     string    `gorm:"type:text" json:"detail"`
	CreatedAt  time.Time `json:"created_at"`
}

func (OperationLog) TableName() string { return "operation_logs" }
