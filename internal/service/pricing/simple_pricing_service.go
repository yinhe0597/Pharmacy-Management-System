// Package pricing 提供计价服务一期实现（仅药费）。
package pricing

import (
	"context"

	"gorm.io/gorm"

	"yaofang/internal/model"
	"yaofang/internal/service/port"
)

// SimplePricingService 一期仅按处方明细快照计算药费。
type SimplePricingService struct {
	db *gorm.DB
}

// NewSimplePricingService 构建计价服务。
func NewSimplePricingService(db *gorm.DB) *SimplePricingService { return &SimplePricingService{db: db} }

// CalculatePrescriptionAmount 按处方明细计算金额行。
func (s *SimplePricingService) CalculatePrescriptionAmount(ctx context.Context, itemIDs []int64) ([]port.PriceLine, error) {
	if len(itemIDs) == 0 {
		return nil, nil
	}
	var items []model.PrescriptionItem
	if err := s.db.WithContext(ctx).Where("id IN ?", itemIDs).Find(&items).Error; err != nil {
		return nil, err
	}
	lines := make([]port.PriceLine, 0, len(items))
	for _, it := range items {
		lines = append(lines, port.PriceLine{
			LineNo:    it.LineNo,
			ItemType:  "drug",
			RefID:     it.ID,
			Quantity:  it.Quantity,
			UnitPrice: it.UnitPrice,
			Amount:    it.Amount,
		})
	}
	return lines, nil
}
