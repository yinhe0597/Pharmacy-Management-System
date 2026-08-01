// Package service 实现领域服务层（业务规则与事务编排的唯一入口）。
package service

import (
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"yaofang/internal/model"
	"yaofang/internal/pkg/auth"
	"yaofang/internal/pkg/errs"
	"yaofang/internal/repository"
)

// AuthService 用户与鉴权服务。
type AuthService struct {
	db  *gorm.DB
	jwt *auth.Manager
}

// NewAuthService 构建鉴权服务。
func NewAuthService(db *gorm.DB, jwt *auth.Manager) *AuthService {
	return &AuthService{db: db, jwt: jwt}
}

// Login 校验账号密码并签发 JWT。
func (s *AuthService) Login(ctx context.Context, username, password string) (string, *model.User, error) {
	u, err := repository.NewUserRepo(s.db).GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil, errs.ErrBadRequest
		}
		return "", nil, err
	}
	if u.Status != 1 {
		return "", nil, errs.New(9002, "账号已停用", 403)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return "", nil, errs.ErrBadRequest
	}
	token, err := s.jwt.Generate(u.ID, u.Username, u.Name, u.Role)
	if err != nil {
		return "", nil, err
	}
	return token, u, nil
}

// Profile 返回用户信息（不含密码）。
func (s *AuthService) Profile(ctx context.Context, userID int64) (*model.User, error) {
	u, err := repository.NewUserRepo(s.db).GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, err
	}
	return u, nil
}

// CreateUser 新建用户。
func (s *AuthService) CreateUser(ctx context.Context, u *model.User, password string) (*model.User, error) {
	if u.Username == "" || password == "" {
		return nil, errs.ErrBadRequest
	}
	if _, err := repository.NewUserRepo(s.db).GetByUsername(ctx, u.Username); err == nil {
		return nil, errs.New(9006, "用户名已存在", 409)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	u.PasswordHash = string(hash)
	u.Status = 1
	if err := repository.NewUserRepo(s.db).Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

// UpdateUser 更新用户信息。
func (s *AuthService) UpdateUser(ctx context.Context, id int64, name, role, phone string, status int, newPassword string) error {
	u, err := repository.NewUserRepo(s.db).GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.ErrNotFound
		}
		return err
	}
	u.Name = name
	u.Role = role
	u.Phone = phone
	if status > 0 {
		u.Status = status
	}
	if newPassword != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		u.PasswordHash = string(hash)
	}
	return repository.NewUserRepo(s.db).Update(ctx, u)
}

// DeleteUser 删除用户（禁止删除自身）。
func (s *AuthService) DeleteUser(ctx context.Context, id, selfID int64) error {
	if id == selfID {
		return errs.New(9007, "不能删除当前登录用户", 400)
	}
	return repository.NewUserRepo(s.db).Delete(ctx, id)
}

// ListUsers 分页查询用户。
func (s *AuthService) ListUsers(ctx context.Context, role, keyword string, page, pageSize int) ([]model.User, int64, error) {
	return repository.NewUserRepo(s.db).List(ctx, role, keyword, (page-1)*pageSize, pageSize)
}
