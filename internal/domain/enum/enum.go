// Package enum 集中定义业务枚举常量，作为唯一取值来源。
package enum

// 库存地点类型。
const (
	LocationTypeWarehouse = 1 // 药库
	LocationTypePharmacy  = 2 // 药房
	LocationTypeDept      = 3 // 科室
)

// 特殊管制药品类型。
const (
	SpecialControlNone     = 0 // 无
	SpecialControlNarcotic = 1 // 麻醉
	SpecialControlPsycho   = 2 // 精神
	SpecialControlToxic    = 3 // 毒性
	SpecialControlRadio    = 4 // 放射性
)

// 精神药品分级。
const (
	PsychoLevelNone = 0
	PsychoLevelOne  = 1 // 一类
	PsychoLevelTwo  = 2 // 二类
)

// 抗生素分级。
const (
	AntibioticNone        = 0 // 非抗生素
	AntibioticNonRestrict = 1 // 非限制
	AntibioticRestrict    = 2 // 限制
	AntibioticSpecial     = 3 // 特殊
)

// 处方类型。
const (
	PrescriptionTypeNormal    = 0 // 普通
	PrescriptionTypeNarcotic  = 1 // 麻醉
	PrescriptionTypePsychoOne = 2 // 精神一类
	PrescriptionTypePsychoTwo = 3 // 精神二类
	PrescriptionTypeToxic     = 4 // 毒性
	PrescriptionTypeRadio     = 5 // 放射性
)

// 处方来源（一期仅 manual，二期扩展）。
const (
	PrescriptionSourceManual = "manual"
)

// 库存流水类型。
const (
	TxnPurchaseIn         = "purchase_in"         // 采购入库
	TxnStockIn            = "stock_in"            // 其他入库
	TxnStockOut           = "stock_out"           // 出库/领用
	TxnDispense           = "dispense"            // 发药
	TxnDispenseReturn     = "dispense_return"     // 退药入库
	TxnStocktakeAdjust    = "stocktake_adjust"    // 盘点调整
	TxnTransferOut        = "transfer_out"        // 调拨出
	TxnTransferIn         = "transfer_in"         // 调拨入
	TxnSplitOut           = "split_out"           // 拆零出（整盒）
	TxnSplitIn            = "split_in"            // 拆零入（拆零）
	TxnWaste              = "waste"               // 报损
	TxnReservation        = "reservation"         // 预占（记账）
	TxnReservationRelease = "reservation_release" // 释放预占（记账）
)

// 预警类型。
const (
	AlertExpiry   = "expiry"    // 近效期
	AlertExpired  = "expired"   // 已过期
	AlertBelowMin = "below_min" // 低于下限
)

// 药房物品类型。药品和耗材统一走药房进销存；诊疗项目（手法复位/静脉注射等）属于临床端，不入药房库存。
const (
	ItemTypeDrug       = "drug"       // 药品
	ItemTypeConsumable = "consumable" // 耗材（注射器/纱布/手套等）
)

// 用户角色（docs/18：护士细化为跟诊护士与药房护士）。
// 调配（发药）与核对（双签）职能由医生/药师兼任，不再设独立角色。
const (
	RoleAdmin            = "admin"             // 管理员
	RolePharmacist       = "pharmacist"        // 药师（含调配/核对职能）
	RoleBuyer            = "buyer"             // 采购
	RoleDoctor           = "doctor"            // 医生（含调配/核对职能）
	RoleClinicNurse      = "clinic_nurse"      // 跟诊护士（诊室：患者管理/核对医嘱/辅助开方/诊疗计费）
	RolePharmacyNurse    = "pharmacy_nurse"    // 药房护士（药房：查询/耗材领用补发/处方执行/辅助核对收费）
	RolePharmacyDirector = "pharmacy_director" // 药房主任
	RoleFinance          = "finance"           // 财务
)

// AllRoles 全部角色列表。
var AllRoles = []string{
	RoleAdmin, RolePharmacist, RoleBuyer,
	RoleDoctor, RoleClinicNurse, RolePharmacyNurse, RolePharmacyDirector, RoleFinance,
}

// IsValidRole 校验角色名是否合法。
func IsValidRole(role string) bool {
	for _, r := range AllRoles {
		if r == role {
			return true
		}
	}
	return false
}

// RoleGroups 预定义角色分组。
var (
	// PharmacyStaff 药房工作人员（库存写/处方执行/调配/核对/药学服务/计费录入）
	PharmacyStaff = []string{RoleAdmin, RolePharmacyDirector, RolePharmacist, RoleDoctor, RolePharmacyNurse}
	// ClinicalStaff 临床人员（处方开立权；含跟诊护士辅助录入医嘱/核对医嘱）
	ClinicalStaff = []string{RoleAdmin, RoleDoctor, RolePharmacist, RolePharmacyDirector, RoleClinicNurse}
	// ReportAccess 报表访问权
	ReportAccess = []string{RoleAdmin, RolePharmacyDirector, RolePharmacist, RoleFinance}
	// UserAdmin 用户管理权（管理员 + 药房主任）
	UserAdmin = []string{RoleAdmin, RolePharmacyDirector}
	// DrugAdmin 药品/分类/配伍/交互规则/特殊药品目录 写权限（药房专业角色）
	DrugAdmin = []string{RoleAdmin, RolePharmacyDirector, RolePharmacist}
	// PurchaseStaff 采购相关人员（采购单/收货/供应商）
	PurchaseStaff = []string{RoleAdmin, RolePharmacyDirector, RolePharmacist, RoleBuyer}
	// PatientAdmin 患者档案写（跟诊护士/医生/主任/管理员，docs/18）
	PatientAdmin = []string{RoleAdmin, RolePharmacyDirector, RoleDoctor, RoleClinicNurse}
	// PatientRead 患者档案读（患者管理与药房人员，docs/18）
	PatientRead = []string{RoleAdmin, RolePharmacyDirector, RoleDoctor, RolePharmacist, RoleClinicNurse, RolePharmacyNurse}
	// ChargeStaff 计费录入/红冲（含双护士，docs/18）
	ChargeStaff = []string{RoleAdmin, RolePharmacyDirector, RolePharmacist, RoleDoctor, RoleClinicNurse, RolePharmacyNurse}
)

// IsPharmacistRole 判断是否为药师级角色（审核/特殊药品双人核对要求）。
func IsPharmacistRole(role string) bool {
	return role == RoleAdmin || role == RolePharmacyDirector || role == RolePharmacist
}

// 配伍等级。
const (
	InteractionLevelContraindication = 1 // 禁忌
	InteractionLevelCaution          = 2 // 慎用
	InteractionLevelNote             = 3 // 注意
)

// 证据等级。
const (
	EvidenceLevelMetaRCT       = 'A' // Meta分析/RCT系统评价
	EvidenceLevelRCT           = 'B' // 随机对照试验
	EvidenceLevelObservational = 'C' // 观察性研究
	EvidenceLevelCaseReport    = 'D' // 病例报告
	EvidenceLevelExpertOnly    = 'E' // 专家经验
)

// 交互检测策略。
const (
	InteractionStrategyExplicit   = "explicit"
	InteractionStrategyIngredient = "ingredient"
	InteractionStrategyClass      = "class"
	InteractionStrategyTag        = "tag"
)

// 禁忌类型。
const (
	ContraindicationTypeAge       = "age"
	ContraindicationTypePregnancy = "pregnancy"
	ContraindicationTypeLactation = "lactation"
	ContraindicationTypeDisease   = "disease"
	ContraindicationTypeAllergy   = "allergy"
)

// 交互严重程度。
const (
	InteractionSeverityBlock   = "block"
	InteractionSeverityWarning = "warning"
)

// --- 错误码常量（供 domain 层引用，避免循环依赖） ---

const (
	ErrInteractionCode               = 3003
	ErrInteractionCautionCode        = 30031 // 慎用提示
	ErrInteractionNoteCode           = 30032 // 注意提示
	ErrDoseExceededCode              = 3002
	ErrDuplicateDrugCode             = 3004
	ErrAgeContraindicationCode       = 3008
	ErrPregnancyContraindicationCode = 3009
	ErrAllergyContraindicationCode   = 3010
	ErrLactationWarningCode          = 3011
)

// 特殊管制药品是否启用「五专」管理（麻醉/精神）。
func IsSpecialControlled(controlType int) bool {
	return controlType == SpecialControlNarcotic || controlType == SpecialControlPsycho
}
