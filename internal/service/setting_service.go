// Package service 系统设置服务（管理员自定义默认诊费，docs/20 S4）。
package service

import (
	"context"
	"errors"
	"strconv"

	"gorm.io/gorm"

	"yaofang/internal/model"
	"yaofang/internal/pkg/errs"
)

// SettingService 系统设置服务。
type SettingService struct {
	db *gorm.DB
}

func NewSettingService(db *gorm.DB) *SettingService { return &SettingService{db: db} }

// List 全部设置项（按 key 排序，稳定输出）。
func (s *SettingService) List(ctx context.Context) ([]model.SystemSetting, error) {
	var list []model.SystemSetting
	if err := s.db.WithContext(ctx).Order("key ASC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// Update 更新设置项。金额类设置要求非负整数（分）。
func (s *SettingService) Update(ctx context.Context, key, value, operator string) error {
	var st model.SystemSetting
	if err := s.db.WithContext(ctx).First(&st, "key = ?", key).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.ErrNotFound
		}
		return err
	}
	// 金额类设置：非负整数校验
	switch key {
	case model.SettingDefaultRegistrationFee, model.SettingDefaultConsultationFee:
		n, err := strconv.ParseInt(value, 10, 64)
		if err != nil || n < 0 {
			return errs.ErrBadRequest
		}
	case model.SettingDefaultReceiveLocation, model.SettingDefaultDispenseLocation:
		// 库房类设置：正整数 + 库房必须存在且启用（000035）
		n, err := strconv.ParseInt(value, 10, 64)
		if err != nil || n <= 0 {
			return errs.ErrBadRequest
		}
		var loc model.InventoryLocation
		if err := s.db.WithContext(ctx).First(&loc, "id = ?", n).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errs.ErrBadRequest
			}
			return err
		}
		if !loc.IsActive {
			return errs.ErrBadRequest
		}
	case model.SettingLogRetentionDays:
		n, err := strconv.ParseInt(value, 10, 64)
		if err != nil || n < 0 {
			return errs.ErrBadRequest
		}
	}
	st.Value = value
	st.UpdatedBy = operator
	if err := s.db.WithContext(ctx).Model(&st).Select("value", "updated_by", "updated_at").Updates(map[string]interface{}{
		"value": value, "updated_by": operator,
	}).Error; err != nil {
		return err
	}
	return nil
}

// LocationID 读取库房类设置。设置缺失/非法/指向不存在或已停用库房时回退 fallback
// （发药/收货不得因配置问题中断；写入侧由 Update 强校验，000035）。
func (s *SettingService) LocationID(ctx context.Context, key string, fallback int64) int64 {
	var st model.SystemSetting
	if err := s.db.WithContext(ctx).First(&st, "key = ?", key).Error; err != nil {
		return fallback
	}
	n, err := strconv.ParseInt(st.Value, 10, 64)
	if err != nil || n <= 0 {
		return fallback
	}
	var loc model.InventoryLocation
	if err := s.db.WithContext(ctx).First(&loc, "id = ?", n).Error; err != nil || !loc.IsActive {
		return fallback
	}
	return n
}
