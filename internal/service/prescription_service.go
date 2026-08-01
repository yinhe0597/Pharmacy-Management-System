package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"yaofang/internal/domain/enum"
	"yaofang/internal/domain/interaction"
	"yaofang/internal/domain/prescription"
	"yaofang/internal/domain/rule"
	"yaofang/internal/model"
	"yaofang/internal/pkg/errs"
	"yaofang/internal/pkg/money"
	"yaofang/internal/pkg/seq"
	"yaofang/internal/repository"
	"yaofang/internal/service/port"
)

// PrescriptionLocationID 一期处方发药默认库房（门诊药房，种子数据 ID=2）。
const PrescriptionLocationID int64 = 2

// PrescriptionItemInput 处方明细输入。
// Quantity 统一按拆零单位（LDU，如片）计；不可拆零药品按基本单位（盒，pack_size=1 等价）。
// IsSplit=true 表示强制拆零发药（全部按拆零价/拆零库存）；false（默认）为混合发药：
// 整盒部分按盒价、零头部分按拆零价，自动跨整盒+拆零库存分配。
type PrescriptionItemInput struct {
	DrugID         int64  `json:"drug_id"`
	Quantity       int64  `json:"quantity"` // LDU（拆零单位）
	IsSplit        bool   `json:"is_split"` // true=强制拆零
	UsageText      string `json:"usage_text"`
	Frequency      string `json:"frequency"`
	SingleDose     int64  `json:"single_dose"`      // 拆零单位
	TotalDailyDose int64  `json:"total_daily_dose"` // 拆零单位
	Days           int    `json:"days"`
}

// PrescriptionInput 处方录入输入。
type PrescriptionInput struct {
	PatientName      string                  `json:"patient_name"`
	PatientGender    string                  `json:"patient_gender"`
	PatientAge       string                  `json:"patient_age"`
	PatientCardNo    string                  `json:"patient_card_no"`
	Diagnosis        string                  `json:"diagnosis"`
	Department       string                  `json:"department"`
	DoctorName       string                  `json:"doctor_name"`
	PrescriptionType int                     `json:"prescription_type"`
	Remarks          string                  `json:"remarks"`
	Items            []PrescriptionItemInput `json:"items"`
}

// AuditInput 审核输入。
type AuditInput struct {
	Action  string `json:"action"` // pass / reject
	Remarks string `json:"remarks"`
}

// ReturnItemInput 退药输入。
type ReturnItemInput struct {
	ItemID         int64 `json:"item_id"`
	ReturnQuantity int64 `json:"return_quantity"` // 与明细同口径
}

// PrescriptionService 处方服务（状态机与库存联动的唯一入口）。
type PrescriptionService struct {
	db         *gorm.DB
	inventory  *InventoryService
	special    *SpecialDrugService
	interSvc   *InteractionService
	patientSvc port.IPatientService
}

// NewPrescriptionService 构建处方服务。
func NewPrescriptionService(db *gorm.DB, inventory *InventoryService, special *SpecialDrugService, interSvc *InteractionService, patientSvc port.IPatientService) *PrescriptionService {
	return &PrescriptionService{db: db, inventory: inventory, special: special, interSvc: interSvc, patientSvc: patientSvc}
}

// deriveSpecialType 根据明细药品的管制标记推导处方类型。
func deriveSpecialType(items []model.Drug) int {
	max := 0
	for _, d := range items {
		if d.SpecialControlType > max {
			max = d.SpecialControlType
		}
	}
	switch max {
	case enum.SpecialControlNarcotic:
		return enum.PrescriptionTypeNarcotic
	case enum.SpecialControlPsycho:
		// 依据精神分级
		for _, d := range items {
			if d.SpecialControlType == enum.SpecialControlPsycho {
				if d.PsychotropicLevel == enum.PsychoLevelOne {
					return enum.PrescriptionTypePsychoOne
				}
				return enum.PrescriptionTypePsychoTwo
			}
		}
		return enum.PrescriptionTypePsychoTwo
	case enum.SpecialControlToxic:
		return enum.PrescriptionTypeToxic
	case enum.SpecialControlRadio:
		return enum.PrescriptionTypeRadio
	}
	return enum.PrescriptionTypeNormal
}

// Create 处方录入（草稿，pending_review）。
func (s *PrescriptionService) Create(ctx context.Context, input PrescriptionInput) (*model.Prescription, error) {
	if input.PatientName == "" || len(input.Items) == 0 {
		return nil, errs.ErrBadRequest
	}
	items, drugs, err := s.prepareItems(ctx, s.db, input.Items)
	if err != nil {
		return nil, err
	}
	prescType := input.PrescriptionType
	if prescType == 0 {
		prescType = deriveSpecialType(drugs)
	}
	if prescType != enum.PrescriptionTypeNormal {
		if input.PatientCardNo == "" || input.Diagnosis == "" {
			return nil, errs.ErrSpecialDrugRequired
		}
		// 处方限量校验（麻精）
		if err := s.special.CheckPrescriptionLimit(ctx, prescType, input.Items); err != nil {
			return nil, err
		}
	}
	p := &model.Prescription{
		PrescriptionNo:     seq.Next("RX"),
		PatientName:        input.PatientName,
		PatientGender:      input.PatientGender,
		PatientAge:         input.PatientAge,
		PatientCardNo:      input.PatientCardNo,
		Diagnosis:          input.Diagnosis,
		Department:         input.Department,
		DoctorName:         input.DoctorName,
		PrescriptionType:   prescType,
		SpecialControlType: specialControlOf(drugs),
		Source:             enum.PrescriptionSourceManual,
		Status:             prescription.StatusPendingReview.String(),
		Remarks:            input.Remarks,
	}
	var total int64
	for i := range items {
		items[i].LineNo = i + 1
		total += items[i].Amount
	}
	p.TotalAmount = total
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := repository.NewPrescriptionRepo(tx).Create(ctx, p); err != nil {
			return err
		}
		for _, it := range items {
			it.PrescriptionID = p.ID
		}
		if err := repository.NewPrescriptionItemRepo(tx).CreateBatch(ctx, items); err != nil {
			return err
		}
		return s.auditTx(ctx, tx, p.ID, "create", "", prescription.StatusPendingReview.String(), "", "")
	})
	if err != nil {
		return nil, err
	}
	return p, nil
}

// prepareItems 校验并生成明细（价格快照 + 混合计价）。
// 计价规则（docs/13 §4.3）：整盒部分按盒价、零头部分按拆零价，金额精确。
func (s *PrescriptionService) prepareItems(ctx context.Context, db *gorm.DB, inputs []PrescriptionItemInput) ([]*model.PrescriptionItem, []model.Drug, error) {
	drugRepo := repository.NewDrugRepo(db)
	items := make([]*model.PrescriptionItem, 0, len(inputs))
	drugs := make([]model.Drug, 0, len(inputs))
	for _, in := range inputs {
		if in.Quantity <= 0 {
			return nil, nil, errs.ErrBadRequest
		}
		d, err := drugRepo.GetByID(ctx, in.DrugID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, nil, errs.New(1008, "药品不存在", 404)
			}
			return nil, nil, err
		}
		if d.Status != 1 {
			return nil, nil, errs.ErrDrugInactive
		}
		splitAllowed := d.IsSplitAllowed && d.PackSize > 1
		if in.IsSplit && !splitAllowed {
			return nil, nil, errs.ErrSplitNotAllowed
		}
		var unitPrice, amount int64
		switch {
		case in.IsSplit:
			// 强制拆零：全部按拆零价
			unitPrice = d.SplitRetailPrice
			amount = in.Quantity * d.SplitRetailPrice
		case splitAllowed:
			// 混合发药：整盒部分按盒价 + 零头按拆零价
			boxes := in.Quantity / int64(d.PackSize)
			units := in.Quantity % int64(d.PackSize)
			unitPrice = d.SplitRetailPrice
			amount = boxes*d.RetailPrice + units*d.SplitRetailPrice
		default:
			// 不可拆零：整盒
			unitPrice = d.RetailPrice
			amount = in.Quantity * d.RetailPrice
		}
		items = append(items, &model.PrescriptionItem{
			DrugID:   in.DrugID,
			DrugName: d.GenericName, Specification: d.Specification,
			Manufacturer: d.Manufacturer, DosageForm: d.DosageForm,
			BaseUnit: d.BaseUnit, SplitUnit: d.SplitUnit, PackSize: d.PackSize,
			IsSplitAllowed: splitAllowed, IsSplit: in.IsSplit, Quantity: in.Quantity,
			UnitPrice: unitPrice, RetailPrice: d.RetailPrice, Amount: amount,
			UsageText: in.UsageText, Frequency: in.Frequency,
			SingleDose: in.SingleDose, TotalDailyDose: in.TotalDailyDose,
			Days: in.Days,
		})
		drugs = append(drugs, *d)
	}
	return items, drugs, nil
}

func specialControlOf(drugs []model.Drug) int {
	max := 0
	for _, d := range drugs {
		if d.SpecialControlType > max {
			max = d.SpecialControlType
		}
	}
	return max
}

// Update 修改处方（仅 pending_review 且无预占时允许）。
func (s *PrescriptionService) Update(ctx context.Context, id int64, input PrescriptionInput) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		p, err := repository.NewPrescriptionRepo(tx).LockForUpdate(ctx, id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errs.ErrNotFound
			}
			return err
		}
		if p.Status != prescription.StatusPendingReview.String() {
			return errs.ErrPrescriptionState
		}
		var n int64
		if err := tx.Model(&model.StockReservation{}).
			Where("ref_type = 'prescription' AND ref_id = ? AND status = 'active'", id).Count(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			return errs.ErrPrescriptionState
		}
		items, drugs, err := s.prepareItems(ctx, tx, input.Items)
		if err != nil {
			return err
		}
		// 重建明细
		if err := tx.Where("prescription_id = ?", id).Delete(&model.PrescriptionItem{}).Error; err != nil {
			return err
		}
		var total int64
		for i := range items {
			items[i].PrescriptionID = id
			items[i].LineNo = i + 1
			total += items[i].Amount
		}
		if err := repository.NewPrescriptionItemRepo(tx).CreateBatch(ctx, items); err != nil {
			return err
		}
		prescType := input.PrescriptionType
		if prescType == 0 {
			prescType = deriveSpecialType(drugs)
		}
		// 与 Create 一致：麻精/特殊处方必须满足卡号、诊断与限量校验，防止修改明细绕过管控
		if prescType != enum.PrescriptionTypeNormal {
			if input.PatientCardNo == "" || input.Diagnosis == "" {
				return errs.ErrSpecialDrugRequired
			}
			if err := s.special.CheckPrescriptionLimit(ctx, prescType, input.Items); err != nil {
				return err
			}
		}
		p.PatientName = input.PatientName
		p.PatientGender = input.PatientGender
		p.PatientAge = input.PatientAge
		p.PatientCardNo = input.PatientCardNo
		p.Diagnosis = input.Diagnosis
		p.Department = input.Department
		p.DoctorName = input.DoctorName
		p.PrescriptionType = prescType
		p.SpecialControlType = specialControlOf(drugs)
		p.TotalAmount = total
		p.Remarks = input.Remarks
		return repository.NewPrescriptionRepo(tx).UpdateBase(ctx, p)
	})
}

// Submit 提交审核：执行库存预占（开单即锁）。
func (s *PrescriptionService) Submit(ctx context.Context, id int64, operatorID int64, operatorName string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		p, err := repository.NewPrescriptionRepo(tx).LockForUpdate(ctx, id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errs.ErrNotFound
			}
			return err
		}
		from := p.Status
		if p.Status == prescription.StatusReviewedRejected.String() {
			// 驳回后重新提交：持久化回 pending_review，再预占
			p.Status = prescription.StatusPendingReview.String()
			if _, err := repository.NewPrescriptionRepo(tx).UpdateStatus(ctx, p, p.Version); err != nil {
				return errs.ErrStateConflict
			}
			if err := s.auditTx(ctx, tx, id, "re_submit", prescription.StatusReviewedRejected.String(), prescription.StatusPendingReview.String(), operatorName, ""); err != nil {
				return err
			}
		} else if p.Status != prescription.StatusPendingReview.String() {
			return errs.ErrPrescriptionState
		}
		// 幂等：已有 active 预占则视为已提交，不再重复预占
		var n int64
		if err := tx.Model(&model.StockReservation{}).
			Where("ref_type = 'prescription' AND ref_id = ? AND status = 'active'", id).Count(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			return nil
		}
		itemRepo := repository.NewPrescriptionItemRepo(tx)
		items, err := itemRepo.ListByPrescription(ctx, id)
		if err != nil {
			return err
		}
		reserveItems := make([]port.ReserveItem, 0, len(items))
		for _, it := range items {
			reserveItems = append(reserveItems, port.ReserveItem{
				DrugID: it.DrugID, LocationID: PrescriptionLocationID,
				// 混合发药：非强制拆零且药品可拆零 → 按 LDU 跨整盒+拆零分配
				IsSplit: it.IsSplit, Mixed: !it.IsSplit && it.IsSplitAllowed,
				Quantity: it.Quantity, ItemID: it.ID,
			})
		}
		results, err := s.inventory.reserveItemsTx(ctx, tx, "prescription", id, reserveItems)
		if err != nil {
			return err
		}
		// 库存不足仅提示不拦截（调配时重新校验）；存在整盒可用时提示可拆零补足
		shortages := ""
		for _, r := range results {
			if r.Shortage > 0 {
				hint := fmt.Sprintf("药品%d缺%d;", r.DrugID, r.Shortage)
				if packs, err := s.inventory.AvailablePacks(ctx, r.DrugID, PrescriptionLocationID); err == nil && packs > 0 {
					hint += fmt.Sprintf("整盒可用%d盒，可拆零后重新提交;", packs)
				}
				shortages += hint
			}
		}
		return s.auditTx(ctx, tx, id, "submit", from, p.Status, operatorName, shortages)
	})
}

// AuditReviewResult 审核结果（含交互检测明细）。
type AuditReviewResult struct {
	Passed   bool                           `json:"passed"`
	Warnings []interaction.PairWarning      `json:"warnings,omitempty"`
	DrugWarnings []interaction.DrugWarning  `json:"drug_warnings,omitempty"`
}

// Review 审核：配伍/极量/重复用药检查；pass→reviewed_passed，reject→reviewed_rejected+释放预占。
// 返回审核明细（含提醒项），供前端展示。
func (s *PrescriptionService) Review(ctx context.Context, id int64, input AuditInput, auditorID int64, auditorName string) (*AuditReviewResult, error) {
	var result *AuditReviewResult
	err := s.db.Transaction(func(tx *gorm.DB) error {
		p, err := repository.NewPrescriptionRepo(tx).LockForUpdate(ctx, id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errs.ErrNotFound
			}
			return err
		}
		if p.Status != prescription.StatusPendingReview.String() {
			return errs.ErrPrescriptionState
		}
		itemRepo := repository.NewPrescriptionItemRepo(tx)
		items, err := itemRepo.ListByPrescription(ctx, id)
		if err != nil {
			return err
		}
		if input.Action == "pass" {
			auditResult, err := s.checkAuditRules(ctx, tx, items, p.PatientID)
			if err != nil {
				return err
			}
			// 构建审核明细结果
			result = &AuditReviewResult{
				Passed:       !auditResult.HasBlocks(),
				Warnings:     auditResult.Warnings,
				DrugWarnings: auditResult.DrugWarnings,
			}
			// 拦截项阻止通过
			if auditResult.HasBlocks() {
				_ = s.interSvc.SaveInteractionResults(ctx, tx, id, auditResult)
				return errs.ErrInteraction
			}
			// 保存提醒项快照
			_ = s.interSvc.SaveInteractionResults(ctx, tx, id, auditResult)
			p.Status = prescription.StatusReviewedPassed.String()
			p.AuditorID = auditorID
			p.AuditorName = auditorName
			now := time.Now()
			p.ReviewedAt = &now
		} else if input.Action == "reject" {
			result = &AuditReviewResult{Passed: false}
			p.Status = prescription.StatusReviewedRejected.String()
			if err := s.releaseReservationsTx(ctx, tx, id); err != nil {
				return err
			}
		} else {
			return errs.ErrBadRequest
		}
		if _, err := repository.NewPrescriptionRepo(tx).UpdateStatus(ctx, p, p.Version); err != nil {
			return errs.ErrStateConflict
		}
		return s.auditTx(ctx, tx, id, "review_"+input.Action,
			prescription.StatusPendingReview.String(), p.Status, auditorName, input.Remarks)
	})
	return result, err
}

// checkAuditRules 配伍/极量/重复用药检查。
// 返回结构化审核结果，包含拦截项和提醒项。
// 引擎负责全部策略（显式药品对、成分级、分类级、标签级）的匹配和去重。
func (s *PrescriptionService) checkAuditRules(ctx context.Context, db *gorm.DB, items []model.PrescriptionItem, patientID int64) (*interaction.AuditResult, error) {
	if len(items) == 0 {
		return nil, errs.ErrBadRequest
	}
	// 优先使用新引擎（多层匹配）
	if s.interSvc != nil {
		return s.interSvc.CheckPrescription(ctx, items, patientID, s.patientSvc)
	}
	// 回退：兼容旧逻辑（仅在未注入交互服务时使用）
	drugIDs := make([]int64, 0, len(items))
	for _, it := range items {
		drugIDs = append(drugIDs, it.DrugID)
	}
	interactions, err := repository.NewInteractionRepo(db).ListByDrugIDs(ctx, drugIDs)
	if err != nil {
		return nil, err
	}
	result := &interaction.AuditResult{Passed: true}
	inSet := make(map[int64]struct{}, len(drugIDs))
	for _, id := range drugIDs {
		inSet[id] = struct{}{}
	}
	drugRepo := repository.NewDrugRepo(db)
	for _, inter := range interactions {
		_, aIn := inSet[inter.DrugAID]
		_, bIn := inSet[inter.DrugBID]
		if aIn && bIn {
			da, _ := drugRepo.GetByID(ctx, inter.DrugAID)
			db2, _ := drugRepo.GetByID(ctx, inter.DrugBID)
			nameA, nameB := "", ""
			if da != nil {
				nameA = da.GenericName
			}
			if db2 != nil {
				nameB = db2.GenericName
			}
			f := interaction.InteractionFinding{
				DrugAID:       inter.DrugAID,
				DrugBID:       inter.DrugBID,
				DrugAName:     nameA,
				DrugBName:     nameB,
				Strategy:      enum.InteractionStrategyExplicit,
				Level:         inter.Level,
				Mechanism:     inter.Mechanism,
				EvidenceLevel: inter.EvidenceLevel,
				Description:   inter.Description,
			}
			result.AddInteraction(f)
			if inter.Level == enum.InteractionLevelContraindication {
				result.AddBlock(f.ToBlock().Code, f.ToBlock().Message, f.ToBlock().Detail)
			} else {
				result.AddWarning(f.ToPairWarning())
			}
		}
	}
	// 极量检查
	for _, it := range items {
		d, err := drugRepo.GetByID(ctx, it.DrugID)
		if err != nil {
			continue
		}
		if d.MaxSingleDose > 0 && it.SingleDose > 0 && it.SingleDose > d.MaxSingleDose {
			result.AddBlock(enum.ErrDoseExceededCode, "极量超限",
				d.GenericName+"：单次剂量超过最大单次剂量")
		}
		if d.MaxDailyDose > 0 && it.TotalDailyDose > 0 && it.TotalDailyDose > d.MaxDailyDose {
			result.AddBlock(enum.ErrDoseExceededCode, "极量超限",
				d.GenericName+"：日总剂量超过最大日剂量")
		}
	}
	return result, nil
}

// CheckAuditRules 公开方法：供外部调用处方审核规则检查。
func (s *PrescriptionService) CheckAuditRules(ctx context.Context, items []model.PrescriptionItem, patientID int64) (*interaction.AuditResult, error) {
	return s.checkAuditRules(ctx, s.db, items, patientID)
}

// Dispense 调配：审核通过后进入调配中（核对预占可用）。
func (s *PrescriptionService) Dispense(ctx context.Context, id int64, operatorID int64, operatorName string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		p, err := repository.NewPrescriptionRepo(tx).LockForUpdate(ctx, id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errs.ErrNotFound
			}
			return err
		}
		if p.Status != prescription.StatusReviewedPassed.String() {
			return errs.ErrPrescriptionState
		}
		items, err := repository.NewPrescriptionItemRepo(tx).ListByPrescription(ctx, id)
		if err != nil {
			return err
		}
		// 校验每项预占充足（按明细汇总 LDU，支持整盒+拆零混合）
		resvs, err := repository.NewStockReservationRepo(tx).ListActiveByRef(ctx, "prescription", id)
		if err != nil {
			return err
		}
		got := make(map[int64]int64) // itemID → 已预占 LDU
		itemMap := make(map[int64]*model.PrescriptionItem, len(items))
		for i := range items {
			itemMap[items[i].ID] = &items[i]
		}
		for _, r := range resvs {
			it, ok := itemMap[r.ItemID]
			if !ok {
				continue
			}
			if r.NeedSplit {
				// 待拆盒：该盒对本明细的 LDU 覆盖为其 splitUnits
				got[r.ItemID] += r.SplitUnits
				continue
			}
			got[r.ItemID] += rule.ToLDU(r.IsSplit, r.Quantity, it.PackSize)
		}
		for _, it := range items {
			if got[it.ID] < it.Quantity {
				return errs.ErrBatchAllocation
			}
		}
		p.Status = prescription.StatusDispensing.String()
		p.DispensingPharmacistID = operatorID
		p.DispensingPharmacistName = operatorName
		if _, err := repository.NewPrescriptionRepo(tx).UpdateStatus(ctx, p, p.Version); err != nil {
			return errs.ErrStateConflict
		}
		return s.auditTx(ctx, tx, id, "dispense", prescription.StatusReviewedPassed.String(), prescription.StatusDispensing.String(), operatorName, "")
	})
}

// ConfirmDispense 发药确认：核销预占实扣、写发药记录、联动专账、双人核对。
func (s *PrescriptionService) ConfirmDispense(ctx context.Context, id int64, checkerID int64, checkerName string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		p, err := repository.NewPrescriptionRepo(tx).LockForUpdate(ctx, id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errs.ErrNotFound
			}
			return err
		}
		if p.Status != prescription.StatusDispensing.String() {
			return errs.ErrPrescriptionState
		}
		itemRepo := repository.NewPrescriptionItemRepo(tx)
		items, err := itemRepo.ListByPrescription(ctx, id)
		if err != nil {
			return err
		}
		// 先取预占批次快照（用于发药记录），再核销库存
		resvRepo := repository.NewStockReservationRepo(tx)
		resvs, err := resvRepo.ListActiveByRef(ctx, "prescription", id)
		if err != nil {
			return err
		}
		records := buildDispenseRecords(p, items, resvs, checkerID)
		if len(records) > 0 {
			if err := repository.NewDispenseRecordRepo(tx).CreateBatch(ctx, records); err != nil {
				return err
			}
		}
		// 核销预占并实扣库存（以该单据预占为准，混合整盒+拆零批次）
		if err := s.inventory.consumeTx(ctx, tx, "prescription", id, nil); err != nil {
			return err
		}
		// 更新明细状态与主单
		for _, it := range items {
			status := "dispensed"
			if it.Quantity == 0 {
				status = "pending"
			}
			if err := itemRepo.UpdateDispensed(ctx, it.ID, it.Quantity, status); err != nil {
				return err
			}
		}
		p.Status = prescription.StatusDispensed.String()
		p.CheckerID = checkerID
		p.CheckerName = checkerName
		now := time.Now()
		p.DispensedAt = &now
		if _, err := repository.NewPrescriptionRepo(tx).UpdateStatus(ctx, p, p.Version); err != nil {
			return errs.ErrStateConflict
		}
		if err := s.auditTx(ctx, tx, id, "confirm_dispense", prescription.StatusDispensing.String(), prescription.StatusDispensed.String(), checkerName, ""); err != nil {
			return err
		}
		// 特殊药品专账联动
		if p.SpecialControlType != enum.SpecialControlNone {
			return s.special.WriteDispenseLedgerTx(ctx, tx, p)
		}
		return nil
	})
}

// buildDispenseRecords 依据预占记录快照生成发药记录。
func buildDispenseRecords(p *model.Prescription, items []model.PrescriptionItem, resvs []model.StockReservation, checkerID int64) []*model.PrescriptionDispenseRecord {
	itemByID := make(map[int64]*model.PrescriptionItem, len(items))
	for i := range items {
		itemByID[items[i].ID] = &items[i]
	}
	records := make([]*model.PrescriptionDispenseRecord, 0, len(resvs))
	for _, r := range resvs {
		it, ok := itemByID[r.ItemID]
		if !ok {
			continue
		}
		if r.NeedSplit {
			// 自动拆零盒：患者取 splitUnits 片（按拆零价），记录为拆零形态
			records = append(records, &model.PrescriptionDispenseRecord{
				PrescriptionID: p.ID, ItemID: r.ItemID, InventoryID: r.InventoryID,
				DrugID: r.DrugID, BatchNo: r.BatchNo, ExpiryDate: r.ExpiryDate,
				IsSplit: true, Quantity: r.SplitUnits,
				UnitPrice: it.UnitPrice, Amount: it.UnitPrice * r.SplitUnits,
				DispensedBy: checkerID,
			})
			continue
		}
		// 按批次形态取快照价：整盒批按盒价，拆零批按拆零价（与明细混合计价一致）
		unitPrice := it.UnitPrice
		if !r.IsSplit {
			unitPrice = it.RetailPrice
		}
		records = append(records, &model.PrescriptionDispenseRecord{
			PrescriptionID: p.ID, ItemID: r.ItemID, InventoryID: r.InventoryID,
			DrugID: r.DrugID, BatchNo: r.BatchNo, ExpiryDate: r.ExpiryDate,
			IsSplit: r.IsSplit, Quantity: r.Quantity,
			UnitPrice: unitPrice, Amount: unitPrice * r.Quantity,
			DispensedBy: checkerID,
		})
	}
	return records
}

// Return 退药：按原发药记录批次回补库存，支持整方/部分。
func (s *PrescriptionService) Return(ctx context.Context, id int64, inputs []ReturnItemInput, operatorID int64, operatorName string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		p, err := repository.NewPrescriptionRepo(tx).LockForUpdate(ctx, id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errs.ErrNotFound
			}
			return err
		}
		if p.Status != prescription.StatusDispensed.String() {
			return errs.ErrPrescriptionState
		}
		itemRepo := repository.NewPrescriptionItemRepo(tx)
		items, err := itemRepo.ListByPrescription(ctx, id)
		if err != nil {
			return err
		}
		itemMap := make(map[int64]*model.PrescriptionItem, len(items))
		for i := range items {
			itemMap[items[i].ID] = &items[i]
		}
		dispRepo := repository.NewDispenseRecordRepo(tx)
		if err := s.inventory.checkLocationNotCounting(ctx, tx, PrescriptionLocationID); err != nil {
			return err
		}
		fullyReturned := make(map[int64]bool, len(items))
		for _, in := range inputs {
			it, ok := itemMap[in.ItemID]
			if !ok {
				return errs.ErrItemNotFound
			}
			if in.ReturnQuantity <= 0 || in.ReturnQuantity > it.DispensedQuantity-it.ReturnedQuantity {
				return errs.ErrReturnExceeded
			}
			// 按原发药记录批次回补
			records, err := dispRepo.ListByItem(ctx, in.ItemID)
			if err != nil {
				return err
			}
			// 重建库存行时取进价（库存 unit_price 为进价口径，发药记录存的是零售价）
			drug, derr := repository.NewDrugRepo(tx).GetByID(ctx, it.DrugID)
			if derr != nil && !errors.Is(derr, gorm.ErrRecordNotFound) {
				return derr
			}
			remaining := in.ReturnQuantity
			for _, rec := range records {
				if remaining <= 0 {
					break
				}
				avail := rec.Quantity - rec.ReturnQuantity
				if avail <= 0 {
					continue
				}
				take := remaining
				if take > avail {
					take = avail
				}
				// 回补库存（同批号同口径）
				invRepo := repository.NewInventoryRepo(tx)
				inv, err := invRepo.FindByKey(ctx, rec.DrugID, PrescriptionLocationID, rec.BatchNo, rec.ExpiryDate, rec.IsSplit)
				var before int64
				if err == nil {
					before = inv.Quantity
					if err := invRepo.Add(ctx, inv.ID, take); err != nil {
						return err
					}
					// 回补批次已过期则锁定，禁止再发药
					if rule.IsExpired(rec.ExpiryDate, todayNow()) {
						if err := invRepo.SetStatus(ctx, inv.ID, 2); err != nil {
							return err
						}
					}
				} else if errors.Is(err, gorm.ErrRecordNotFound) {
					before = 0
					// 批次已清空时重建（保留原批次信息）；成本取进价而非发药零售价
					status := 1
					if rule.IsExpired(rec.ExpiryDate, todayNow()) {
						status = 2
					}
					cost := int64(0)
					if drug != nil {
						if rec.IsSplit {
							cost = money.Cents(drug.PurchasePrice).SplitPrice(drug.PackSize).Int64()
						} else {
							cost = drug.PurchasePrice
						}
					}
					if err := invRepo.Create(ctx, &model.Inventory{
						DrugID: rec.DrugID, LocationID: PrescriptionLocationID,
						BatchNo: rec.BatchNo, ExpiryDate: rec.ExpiryDate,
						Quantity: take, IsSplit: rec.IsSplit, UnitPrice: cost, Status: status,
					}); err != nil {
						return err
					}
				} else {
					return err
				}
				expiry := rec.ExpiryDate
				txn := &model.InventoryTransaction{
					TransactionNo: seq.Next("ITN"),
					DrugID:        rec.DrugID, LocationID: PrescriptionLocationID,
					BatchNo: rec.BatchNo, ExpiryDate: &expiry,
					Quantity: take, IsSplit: rec.IsSplit,
					BeforeQuantity: before, AfterQuantity: before + take,
					TxnType: enum.TxnDispenseReturn, RefType: "prescription", RefID: id,
					OperatorID: operatorID, OperatorName: operatorName, Remarks: "退药回补",
				}
				if err := repository.NewInventoryTransactionRepo(tx).Create(ctx, txn); err != nil {
					return err
				}
				if err := dispRepo.UpdateReturnQty(ctx, rec.ID, take); err != nil {
					return err
				}
				remaining -= take
			}
			if remaining > 0 {
				return errs.ErrReturnExceeded
			}
			newReturned := it.ReturnedQuantity + in.ReturnQuantity
			itemStatus := "dispensed"
			if newReturned >= it.DispensedQuantity {
				itemStatus = "returned"
			} else {
				itemStatus = "partially_returned"
			}
			if err := itemRepo.UpdateReturned(ctx, it.ID, in.ReturnQuantity, itemStatus); err != nil {
				return err
			}
			fullyReturned[in.ItemID] = newReturned >= it.DispensedQuantity
		}
		// 仅当「所有明细」均已全部退回时才置整方 returned，未出现在本次输入的明细视作未退
		allReturned := true
		for _, it := range items {
			if !fullyReturned[it.ID] {
				allReturned = false
				break
			}
		}
		if allReturned {
			p.Status = prescription.StatusReturned.String()
			if _, err := repository.NewPrescriptionRepo(tx).UpdateStatus(ctx, p, p.Version); err != nil {
				return errs.ErrStateConflict
			}
		}
		return s.auditTx(ctx, tx, id, "return", prescription.StatusDispensed.String(), p.Status, operatorName, "")
	})
}

// Cancel 作废：释放预占。
func (s *PrescriptionService) Cancel(ctx context.Context, id int64, operatorName string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		p, err := repository.NewPrescriptionRepo(tx).LockForUpdate(ctx, id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errs.ErrNotFound
			}
			return err
		}
		action, ok := prescription.Status(p.Status).CanTransitTo(prescription.StatusCancelled)
		if !ok {
			return errs.ErrPrescriptionState
		}
		if err := s.releaseReservationsTx(ctx, tx, id); err != nil {
			return err
		}
		from := p.Status
		p.Status = prescription.StatusCancelled.String()
		if _, err := repository.NewPrescriptionRepo(tx).UpdateStatus(ctx, p, p.Version); err != nil {
			return errs.ErrStateConflict
		}
		return s.auditTx(ctx, tx, id, action, from, p.Status, operatorName, "作废")
	})
}

// releaseReservationsTx 释放指定处方的全部预占。
func (s *PrescriptionService) releaseReservationsTx(ctx context.Context, tx *gorm.DB, id int64) error {
	resvs, err := repository.NewStockReservationRepo(tx).ListActiveByRef(ctx, "prescription", id)
	if err != nil {
		return err
	}
	if len(resvs) == 0 {
		return nil
	}
	items := make([]port.ReserveItem, 0, len(resvs))
	for _, r := range resvs {
		items = append(items, port.ReserveItem{DrugID: r.DrugID, LocationID: r.LocationID, IsSplit: r.IsSplit, Quantity: r.Quantity})
	}
	return s.inventory.cancelTx(ctx, tx, "prescription", id, items)
}

// auditTx 写状态流转日志。
func (s *PrescriptionService) auditTx(ctx context.Context, tx *gorm.DB, prescID int64, action, from, to, operatorName, remarks string) error {
	return repository.NewPrescriptionAuditRepo(tx).Create(ctx, &model.PrescriptionAuditLog{
		PrescriptionID: prescID, Action: action,
		FromStatus: from, ToStatus: to,
		OperatorName: operatorName, Remarks: remarks,
	})
}

// Get 处方详情（含明细、发药记录、审计日志）。
func (s *PrescriptionService) Get(ctx context.Context, id int64) (*PrescriptionDetail, error) {
	db := s.db
	p, err := repository.NewPrescriptionRepo(db).GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, err
	}
	items, err := repository.NewPrescriptionItemRepo(db).ListByPrescription(ctx, id)
	if err != nil {
		return nil, err
	}
	records, err := repository.NewDispenseRecordRepo(db).ListByPrescription(ctx, id)
	if err != nil {
		return nil, err
	}
	logs, err := repository.NewPrescriptionAuditRepo(db).ListByPrescription(ctx, id)
	if err != nil {
		return nil, err
	}
	return &PrescriptionDetail{Prescription: *p, Items: items, DispenseRecords: records, AuditLogs: logs}, nil
}

// PrescriptionDetail 处方详情聚合。
type PrescriptionDetail struct {
	model.Prescription
	Items           []model.PrescriptionItem           `json:"items"`
	DispenseRecords []model.PrescriptionDispenseRecord `json:"dispense_records"`
	AuditLogs       []model.PrescriptionAuditLog       `json:"audit_logs"`
}

// List 处方列表。
func (s *PrescriptionService) List(ctx context.Context, f repository.PrescriptionFilter, page, pageSize int) ([]model.Prescription, int64, error) {
	return repository.NewPrescriptionRepo(s.db).List(ctx, f, (page-1)*pageSize, pageSize)
}
