// Package sanitize 提供敏感信息脱敏工具，防止敏感数据落入日志。
package sanitize

import "strings"

// MaskIDCard 脱敏身份证号/就诊卡号：保留前 4 位与后 4 位，中间以 * 填充。
func MaskIDCard(s string) string {
	r := []rune(s)
	if len(r) <= 8 {
		return strings.Repeat("*", len(r))
	}
	return string(r[:4]) + strings.Repeat("*", len(r)-8) + string(r[len(r)-4:])
}

// MaskPhone 脱敏手机号：保留前 3 位与后 4 位。
func MaskPhone(s string) string {
	r := []rune(s)
	if len(r) < 7 {
		return strings.Repeat("*", len(r))
	}
	return string(r[:3]) + strings.Repeat("*", len(r)-7) + string(r[len(r)-4:])
}
