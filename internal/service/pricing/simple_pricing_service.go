// Package pricing 提供计价服务一期实现（仅药费）。
package pricing

import (
	"context"
	"strconv"
	"time"

	"gorm.io/gorm"

	"yaofang/internal/domain/enum"
	"yaofang/internal/model"
	"yaofang/internal/pkg/errs"
	"yaofang/internal/pkg/money"
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
// 按就诊精确归集三类费用行：
//
//	① 已发药处方的药品费（prescriptions.visit_id 关联明细快照；部分退药按 ReturnedQuantity 回算净额）
//	② 本次就诊的应收计费项目源（charge_records.visit_id，未红冲、非处方来源）
//	③ 系统设置默认挂号费/诊查费（本次无同类费用行时自动带入）
//
// ② 必须按 visit_id 精确取，不能用 (patient_id, created_at 时间窗口) 猜归属：
// 同一患者其它就诊的计费项会落入窗口被并入本次结算，而挂号前录入的项会被漏掉。
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

	// ① 处方药费（已发药处方的明细快照）：部分退药按 ReturnedQuantity 回算净额，
	// 快照金额 Amount 与退回部分金额同口径（money.ItemAmount，docs/13 §4.3）
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
			returned := it.ReturnedQuantity
			netQty := it.Quantity - returned
			if netQty <= 0 {
				continue // 已全部退回，不计费
			}
			netAmount := it.Amount - money.ItemAmount(it.IsSplit, it.IsSplitAllowed, it.PackSize, it.UnitPrice, it.RetailPrice, returned)
			lines = append(lines, port.PriceLine{
				LineNo:    lineNo,
				ItemType:  enum.ChargeItemTypeDrug,
				RefID:     it.DrugID,
				Quantity:  netQty,
				UnitPrice: it.UnitPrice,
				Amount:    netAmount,
			})
			lineNo++
		}
	}

	// ② 本次就诊的应收计费项目源（charge_records）：
	//   - 未红冲（红冲表示该费用项作废，不得计入账单）
	//   - 排除 ref_type='prescription'：处方药费已由 ① 按快照口径计入，
	//     若这里再取一次即重复计费（ChargePrescription 生成的行属于此类）
	//   - item_type 不再排除 drug：无处方的手工录入药品费（如急救给药）同样要能进账单，
	//     此前被 item_type IN (clinical_service, consumable) 整体排除，导致收入漏记
	//
	// 归集口径：优先按 visit_id 精确匹配；对 visit_id IS NULL 的存量行回退到原时间窗口
	// （注册时刻 → 就诊结束/当前）。回退仅用于 000040 之前录入、尚未补齐就诊关联的历史行；
	// 录入方一旦传 visit_id 即走精确路径，该回退分支会随存量清空而自然失效。
	// 之所以保留回退而不是直接废弃：若只认 visit_id，未传该字段的录入方会**静默漏计费**，
	// 对药房账目而言比错归集更危险。
	windowEnd := time.Now()
	if visit.FinishedAt != nil {
		windowEnd = *visit.FinishedAt
	}
	var crs []model.ChargeRecord
	if err := s.db.WithContext(ctx).
		Where(`voided = false AND ref_type <> ? AND (
		            visit_id = ?
		         OR (visit_id IS NULL AND patient_id = ? AND created_at >= ? AND created_at <= ?)
		      )`,
			"prescription", visitID, visit.PatientID, visit.RegisteredAt, windowEnd).
		Order("id ASC").
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
			LineNo:         lineNo,
			ItemType:       itemType,
			RefID:          refID,
			Quantity:       int64(cr.Quantity),
			UnitPrice:      cr.UnitPrice,
			Amount:         cr.Amount,
			SourceRecordID: cr.ID,
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
