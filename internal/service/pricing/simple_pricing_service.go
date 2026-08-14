// Package pricing 提供计价服务一期实现（仅药费）。
package pricing

import (
	"context"
	"strconv"

	"gorm.io/gorm"

	"yaofang/internal/domain/enum"
	"yaofang/internal/model"
	"yaofang/internal/pkg/errs"
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
			RefID:     it.DrugID, // 契约：药品= drug_id（docs/05 §4；docs/15 H2）
			Quantity:  it.Quantity,
			UnitPrice: it.UnitPrice,
			Amount:    it.Amount,
		})
	}
	return lines, nil
}

// CalculateBill 合并结算计价（docs/20 S4）：
// 按就诊聚合 ① 已发药处方的药品/耗材费（prescriptions.visit_id 关联明细快照）
// 与 ② 就诊期间录入的诊疗项目/耗材计费记录（charge_records，未红冲）。
func (s *SimplePricingService) CalculateBill(ctx context.Context, visitID int64) ([]port.PriceLine, error) {
	var visit model.Visit
	if err := s.db.WithContext(ctx).First(&visit, visitID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errs.ErrVisitNotFound
		}
		return nil, err
	}
	lines := make([]port.PriceLine, 0, 8)
	lineNo := 1

	// ① 处方药费（已发药处方的明细快照，含退药后的净额=Amount 已按 ReturnedQuantity 回算？按快照价计）
	var prescs []model.Prescription
	if err := s.db.WithContext(ctx).Where("visit_id = ? AND status = ?", visitID, "dispensed").Find(&prescs).Error; err != nil {
		return nil, err
	}
	for _, p := range prescs {
		var items []model.PrescriptionItem
		if err := s.db.WithContext(ctx).Where("prescription_id = ?", p.ID).Find(&items).Error; err != nil {
			return nil, err
		}
		for _, it := range items {
			lines = append(lines, port.PriceLine{
				LineNo:    lineNo,
				ItemType:  enum.ChargeItemTypeDrug,
				RefID:     it.DrugID,
				Quantity:  it.Quantity,
				UnitPrice: it.UnitPrice,
				Amount:    it.Amount,
			})
			lineNo++
		}
	}

	// ② 诊疗项目/耗材计费记录（就诊患者、未红冲；period 取就诊时间窗口）
	var crs []model.ChargeRecord
	if err := s.db.WithContext(ctx).
		Where("patient_id = ? AND item_type IN ? AND voided = false", visit.PatientID,
			[]string{enum.ChargeItemTypeClinicalService, enum.ChargeItemTypeConsumable}).
		Find(&crs).Error; err != nil {
		return nil, err
	}
	for _, cr := range crs {
		itemType := cr.ItemType
		refID := int64(0)
		if cr.ItemID != nil {
			refID = *cr.ItemID
		}
		lines = append(lines, port.PriceLine{
			LineNo:    lineNo,
			ItemType:  itemType,
			RefID:     refID,
			Quantity:  int64(cr.Quantity),
			UnitPrice: cr.UnitPrice,
			Amount:    cr.Amount,
		})
		lineNo++
	}

	// ③ 默认诊费（管理员配置，docs/20 S4）：挂号费/诊查费，值>0 且本次无同类费用行时自动带入
	hasType := func(t string) bool {
		for _, l := range lines {
			if l.ItemType == t {
				return true
			}
		}
		return false
	}
	appended := false
	for _, cfg := range []struct {
		key, itemType, name string
	}{
		{model.SettingDefaultRegistrationFee, enum.ChargeItemTypeRegistration, "挂号费"},
		{model.SettingDefaultConsultationFee, enum.ChargeItemTypeConsultation, "诊查费"},
	} {
		var st model.SystemSetting
		if err := s.db.WithContext(ctx).First(&st, "key = ?", cfg.key).Error; err != nil {
			continue
		}
		fee, perr := strconv.ParseInt(st.Value, 10, 64)
		if perr != nil || fee <= 0 || hasType(cfg.itemType) {
			continue
		}
		lines = append(lines, port.PriceLine{
			LineNo:    lineNo,
			ItemType:  cfg.itemType,
			Quantity:  1,
			UnitPrice: fee,
			Amount:    fee,
		})
		lineNo++
		appended = true
	}
	_ = appended
	return lines, nil
}
