// Package money 提供金额（分）的整数运算。禁止在业务代码中使用浮点数参与金额计算。
package money

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

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
//
// 已知取整残差（docs/04 §拆零计价）：逐片计价会产生累计舍入偏差——整盒全部按片发药时，
// 片价之和可能略高于盒价。例：1 盒 = 24 片、盒价 1000 分 → 单片 42 分 → 24×42 = 1008 分（高 8 分）。
// 混合发药（整盒按盒价 + 零头按片价）不受影响，仅「整单按片计价」场景存在该偏差。
func (c Cents) SplitPrice(packSize int) Cents {
	b := int64(packSize)
	if b <= 0 {
		return 0
	}
	return Cents((int64(c) + b/2) / b)
}

// FromFen 构造分值金额（供测试与外部转换）。
func FromFen(v int64) Cents { return Cents(v) }

// ParseYuan 以「元」字符串精确构造金额（用于录入层传入的 "12.34" 这类值，仍以分存储）。
// 全程整数解析，不使用浮点；小数位不足补零，超过两位时对第三位四舍五入（半入）。
// 允许可选正负号与首尾空白；空串、非法字符或数值溢出返回错误。
func ParseYuan(s string) (Cents, error) {
	str := strings.TrimSpace(s)
	if str == "" {
		return 0, errors.New("金额为空")
	}
	neg := false
	switch str[0] {
	case '+':
		str = str[1:]
	case '-':
		neg = true
		str = str[1:]
	}
	intPart, fracPart, hasFrac := strings.Cut(str, ".")
	if intPart == "" {
		intPart = "0"
	}
	if !isDigits(intPart) || (hasFrac && !isDigits(fracPart)) {
		return 0, fmt.Errorf("金额格式非法: %q", s)
	}
	roundUp := false
	if len(fracPart) > 2 {
		roundUp = fracPart[2] >= '5'
		fracPart = fracPart[:2]
	}
	for len(fracPart) < 2 {
		fracPart += "0"
	}
	yuan, err := strconv.ParseInt(intPart, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("金额超出范围: %q", s)
	}
	centsPart, err := strconv.ParseInt(fracPart, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("金额格式非法: %q", s)
	}
	total := yuan*100 + centsPart
	if roundUp {
		total++
	}
	if neg {
		total = -total
	}
	return Cents(total), nil
}

// isDigits 判断字符串是否全为十进制数字（且非空）。
func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// ItemAmount 按处方明细口径计算 qty（LDU）金额（与处方计价同口径，docs/13 §4.3）：
//   - isSplit：强制拆零发药，全部按拆零价 unitPrice；
//   - isSplitAllowed：混合口径，整盒按盒价 retailPrice + 零头按拆零价 unitPrice；
//   - 其余按整盒价 retailPrice。
//
// service 与 pricing 两包共用此实现，保证快照金额与结算净额口径一致。
// packSize <= 0（脏数据）按整盒价计数，packSize == 1（盒=片）按拆零价计数，均不产生除零。
func ItemAmount(isSplit, isSplitAllowed bool, packSize int, unitPrice, retailPrice, qty int64) int64 {
	switch {
	case isSplit:
		return qty * unitPrice
	case isSplitAllowed:
		pack := int64(packSize)
		switch {
		case pack <= 0:
			return qty * retailPrice
		case pack == 1:
			return qty * unitPrice
		}
		boxes := qty / pack
		units := qty % pack
		return boxes*retailPrice + units*unitPrice
	default:
		return qty * retailPrice
	}
}
