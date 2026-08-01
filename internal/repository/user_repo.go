// Package repository 封装 GORM 持久化操作，支持事务透传（构造函数接收 *gorm.DB）。
package repository

import (
	"context"

	"gorm.io/gorm"

	"yaofang/internal/model"
)

// UserRepo 用户仓储。
type UserRepo struct {
	db *gorm.DB
}

// NewUserRepo 创建用户仓储。
func NewUserRepo(db *gorm.DB) *UserRepo { return &UserRepo{db: db} }

// Create 新建用户。
func (r *UserRepo) Create(ctx context.Context, u *model.User) error {
	return r.db.WithContext(ctx).Create(u).Error
}

// GetByUsername 按用户名查询（含软删排除）。
func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	var u model.User
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// GetByID 按 ID 查询。
func (r *UserRepo) GetByID(ctx context.Context, id int64) (*model.User, error) {
	var u model.User
	if err := r.db.WithContext(ctx).First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// Update 更新用户（status/name/phone/role/password_hash）。
func (r *UserRepo) Update(ctx context.Context, u *model.User) error {
	return r.db.WithContext(ctx).Model(u).Select("name", "role", "phone", "status", "password_hash", "updated_at").Updates(u).Error
}

// Delete 软删除用户。
func (r *UserRepo) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.User{}, id).Error
}

// List 分页查询用户。
func (r *UserRepo) List(ctx context.Context, role string, keyword string, offset, limit int) ([]model.User, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.User{})
	if role != "" {
		q = q.Where("role = ?", role)
	}
	if keyword != "" {
		q = q.Where("username ILIKE ? OR name ILIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.User
	if err := q.Order("id ASC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}
