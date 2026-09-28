// Package model 定义 GORM 数据模型，与 migrations/ 下 SQL 保持一致。
package model

import (
	"time"

	"gorm.io/gorm"
)

// User 系统用户。角色：admin/pharmacist/buyer/doctor/nurse/pharmacy_director/finance。
type User struct {
	ID           int64  `gorm:"primaryKey" json:"id"`
	Username     string `gorm:"size:50;uniqueIndex;not null" json:"username"`
	PasswordHash string `gorm:"size:100;not null" json:"-"`
	Name         string `gorm:"size:50;not null" json:"name"`
	Role         string `gorm:"size:20;not null" json:"role"`
	Phone        string `gorm:"size:20" json:"phone"`
	Status       int    `gorm:"not null;default:1" json:"status"` // 1启用 0停用
	// TokenVersion 口令版本号，签入 JWT claims 并在每请求与本值比对。
	// 口令轮换时自增，即可一次性作废该用户全部存量 token（JWT 无服务端吊销机制）。
	TokenVersion int64          `gorm:"not null;default:0" json:"-"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (User) TableName() string { return "users" }
