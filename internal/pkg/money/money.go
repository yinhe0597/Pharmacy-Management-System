// Package money 提供金额（分）的整数运算。禁止在业务代码中使用浮点数参与金额计算。
package money

// Cents 表示以「分」为单位的金额。所有金额字段在数据库中以 bigint 存储。
type Cents int64

// Int64 返回原始分值。
func (c Cents) Int64() int64 { return int64(c) }

// Add 金额相加。
func (c Cents) Add(o Cents) Cents { return Cents(int64(c) + int64(o)) }

// Sub 金额相减（调用方保证结果非负）。
func (c Cents) Sub(o Cents) Cents { return Cents(int64(c) - int64(o)) }

// Mul 金额乘以数量，行金额 = 单价 × 数量，整数精确无舍入。
func (c Cents) Mul(qty int64) Cents { return Cents(int64(c) * qty) }

// SplitPrice 将每基本单位的价格换算为每拆零单位的价格。
// 四舍五入规则（半入）：round(a/b) = (a + b/2) / b，落到个位分。
func (c Cents) SplitPrice(packSize int) Cents {
	b := int64(packSize)
	if b <= 0 {
		return 0
	}
	return Cents((int64(c) + b/2) / b)
}

// FromFen 构造分值金额（供测试与外部转换）。
func FromFen(v int64) Cents { return Cents(v) }

// FromYuan 以「元」构造金额（用于便捷录入，仍以分存储）。
func FromYuan(v float64) Cents { return Cents(int64(v*100 + 0.5)) }

// ItemAmount 按处方明细口径计算 qty（LDU）金额（与处方计价同口径，docs/13 §4.3）：
//   - isSplit：强制拆零发药，全部按拆零价 unitPrice；
//   - isSplitAllowed：混合口径，整盒按盒价 retailPrice + 零头按拆零价 unitPrice；
//   - 其余按整盒价 retailPrice。
//
// service 与 pricing 两包共用此实现，保证快照金额与结算净额口径一致。
func ItemAmount(isSplit, isSplitAllowed bool, packSize int, unitPrice, retailPrice, qty int64) int64 {
	switch {
	case isSplit:
		return qty * unitPrice
	case isSplitAllowed:
		pack := int64(packSize)
		boxes := qty / pack
		units := qty % pack
		return boxes*retailPrice + units*unitPrice
	default:
		return qty * retailPrice
	}
}
