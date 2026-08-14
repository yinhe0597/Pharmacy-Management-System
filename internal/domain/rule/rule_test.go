package rule

import (
	"testing"
	"time"
)

func date(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}

func TestDaysToExpiry(t *testing.T) {
	today := date("2026-08-01")
	cases := []struct {
		expiry string
		want   int
	}{
		{"2026-08-01", 0},
		{"2026-08-02", 1},
		{"2026-07-31", -1},
		{"2026-11-01", 92},
	}
	for _, c := range cases {
		if got := DaysToExpiry(date(c.expiry), today); got != c.want {
			t.Fatalf("DaysToExpiry(%s) = %d, want %d", c.expiry, got, c.want)
		}
	}
}

func TestIsExpired(t *testing.T) {
	today := date("2026-08-01")
	if IsExpired(date("2026-08-01"), today) {
		t.Fatal("当天到期不应判定为过期")
	}
	if !IsExpired(date("2026-07-31"), today) {
		t.Fatal("昨天到期应判定为过期")
	}
	if IsExpired(date("2026-08-02"), today) {
		t.Fatal("明天到期不应判定为过期")
	}
}

func TestIsNearExpiry(t *testing.T) {
	today := date("2026-08-01")
	// 剩余 90 天，阈值 90 → 预警
	if !IsNearExpiry(date("2026-10-30"), today, 90) {
		t.Fatal("剩余90天阈值90应预警")
	}
	// 剩余 91 天，阈值 90 → 不预警
	if IsNearExpiry(date("2026-10-31"), today, 90) {
		t.Fatal("剩余91天阈值90不应预警")
	}
	// 已过期不算近效期
	if IsNearExpiry(date("2026-07-01"), today, 90) {
		t.Fatal("已过期不应判定为近效期")
	}
}

func TestToLDU(t *testing.T) {
	if ToLDU(true, 48, 24) != 48 {
		t.Fatal("拆零行 LDU 应等于数量")
	}
	if ToLDU(false, 5, 24) != 120 {
		t.Fatal("整盒行 LDU 应为 数量×pack_size")
	}
	if ToLDU(false, 5, 0) != 0 {
		t.Fatal("packSize 非法应返回 0")
	}
	if SplitFromLDU(125, 24) != 5 {
		t.Fatal("125 LDU / 24 向下取整应为 5 盒")
	}
	if SplitFromLDU(100, 0) != 0 {
		t.Fatal("packSize 非法应返回 0")
	}
}
