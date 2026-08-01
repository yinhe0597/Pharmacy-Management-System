// Package rule 提供与时间、单位、批次选择相关的纯领域规则。
package rule

import "time"

// daysBetween 返回 a 与 b 之间的自然日差（date-only，忽略时刻）。
func daysBetween(a, b time.Time) int {
	ay := time.Date(a.Year(), a.Month(), a.Day(), 0, 0, 0, 0, time.UTC)
	by := time.Date(b.Year(), b.Month(), b.Day(), 0, 0, 0, 0, time.UTC)
	return int(by.Sub(ay).Hours() / 24)
}

// DaysToExpiry 返回从 today 到效期（expiry）的天数。正数=未过期，负数=已过期，0=当天到期。
func DaysToExpiry(expiry, today time.Time) int {
	return daysBetween(today, expiry)
}

// IsExpired 判断批次是否已过期（效期早于今天，即 expiry_date < today）。
func IsExpired(expiry, today time.Time) bool {
	return daysBetween(today, expiry) < 0
}

// IsNearExpiry 判断批次是否处于近效期（今天<=效期<=今天+warningDays，含未过期）。
func IsNearExpiry(expiry, today time.Time, warningDays int) bool {
	d := DaysToExpiry(expiry, today)
	return d >= 0 && d <= warningDays
}

// ToLDU 将某口径数量折算为最小发药单位（LDU）：
// 拆零行直接返回数量；整盒行数量×pack_size。
func ToLDU(isSplit bool, qty int64, packSize int) int64 {
	if isSplit {
		return qty
	}
	if packSize <= 0 {
		return 0
	}
	return qty * int64(packSize)
}

// SplitFromLDU 将 LDU 折算为整盒数量（向下取整）。
func SplitFromLDU(ldu int64, packSize int) int64 {
	if packSize <= 0 {
		return 0
	}
	return ldu / int64(packSize)
}
