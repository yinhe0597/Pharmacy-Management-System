package model

import "time"

// Notification 站内通知（预警闭环触达，000036）。
// UserID 为空 = 全员广播（所有登录用户可见）；非空 = 定向到具体用户。
// 已读状态不落本表：查询时由 notification_reads 关联拼装（见 repository.NotificationRow）。
type Notification struct {
	ID         int64     `gorm:"primaryKey" json:"id"`
	UserID     *int64    `json:"user_id"`
	Title      string    `gorm:"size:200;not null" json:"title"`
	Content    string    `gorm:"size:1000;not null;default:''" json:"content"`
	Level      string    `gorm:"size:20;not null;default:info" json:"level"` // info/warning/critical
	Resource   string    `gorm:"size:100;not null;default:''" json:"resource"`
	ResourceID *int64    `json:"resource_id"`
	CreatedAt  time.Time `json:"created_at"`
}

func (Notification) TableName() string { return "notifications" }

// NotificationRead 通知已读回执（000036）。
type NotificationRead struct {
	NotificationID int64     `gorm:"primaryKey" json:"notification_id"`
	UserID         int64     `gorm:"primaryKey" json:"user_id"`
	ReadAt         time.Time `json:"read_at"`
}

func (NotificationRead) TableName() string { return "notification_reads" }

// 通知级别常量。
const (
	NotifyLevelInfo     = "info"
	NotifyLevelWarning  = "warning"
	NotifyLevelCritical = "critical"
)
