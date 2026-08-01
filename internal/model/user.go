// Package model 定义 GORM 数据模型，与 migrations/ 下 SQL 保持一致。
package model

import (
	"time"

	"gorm.io/gorm"
)

// User 系统用户。角色：admin/pharmacist/dispenser/checker/buyer/doctor/nurse/pharmacy_director/finance。
type User struct {
	ID           int64          `gorm:"primaryKey" json:"id"`
	Username     string         `gorm:"size:50;uniqueIndex;not null" json:"username"`
	PasswordHash string         `gorm:"size:100;not null" json:"-"`
	Name         string         `gorm:"size:50;not null" json:"name"`
	Role         string         `gorm:"size:20;not null" json:"role"`
	Phone        string         `gorm:"size:20" json:"phone"`
	Status       int            `gorm:"not null;default:1" json:"status"` // 1启用 0停用
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (User) TableName() string { return "users" }
