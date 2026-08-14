// Package model 系统设置模型（管理员自定义默认诊费等，docs/20 S4）。
package model

import "time"

// SystemSetting 系统设置项（key-value）。
type SystemSetting struct {
	Key         string    `gorm:"primaryKey;size:50" json:"key"`
	Value       string    `gorm:"size:200;not null;default:''" json:"value"`
	Description string    `gorm:"size:200" json:"description"`
	UpdatedBy   string    `gorm:"size:50" json:"updated_by"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (SystemSetting) TableName() string { return "system_settings" }

// 设置键常量。
const (
	SettingDefaultRegistrationFee = "default_registration_fee" // 默认挂号费（分）
	SettingDefaultConsultationFee = "default_consultation_fee" // 默认诊查费（分）
)
