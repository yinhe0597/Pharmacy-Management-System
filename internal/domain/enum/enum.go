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

// 用户角色。
const (
	RoleAdmin      = "admin"      // 管理员
	RolePharmacist = "pharmacist" // 药师
	RoleDispenser  = "dispenser"  // 调配
	RoleChecker    = "checker"    // 核对
	RoleBuyer      = "buyer"      // 采购
)

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
	ErrInteractionCode         = 3003
	ErrInteractionCautionCode  = 30031 // 慎用提示
	ErrInteractionNoteCode     = 30032 // 注意提示
	ErrDoseExceededCode        = 3002
	ErrDuplicateDrugCode       = 3004
	ErrAgeContraindicationCode = 3008
	ErrPregnancyContraindicationCode = 3009
	ErrAllergyContraindicationCode   = 3010
	ErrLactationWarningCode          = 3011
)

// 特殊管制药品是否启用「五专」管理（麻醉/精神）。
func IsSpecialControlled(controlType int) bool {
	return controlType == SpecialControlNarcotic || controlType == SpecialControlPsycho
}
