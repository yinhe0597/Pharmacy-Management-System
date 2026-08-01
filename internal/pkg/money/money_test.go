package money

import "testing"

func TestSplitPrice(t *testing.T) {
	cases := []struct {
		name     string
		pack     int64
		packSize int
		want     int64
	}{
		{"整除", 2400, 24, 100},     // 100分/片
		{"四舍五入入", 2500, 24, 104},  // 104.17 → 104
		{"四舍五入入2", 2412, 24, 101}, // 100.5 → 101（半入）
		{"四舍五入舍", 2499, 24, 104},  // 104.125 → 104
		{"整包", 1000, 1, 1000},
		{"packSize非法", 1000, 0, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Cents(c.pack).SplitPrice(c.packSize).Int64()
			if got != c.want {
				t.Fatalf("SplitPrice(%d, %d) = %d, want %d", c.pack, c.packSize, got, c.want)
			}
		})
	}
}

func TestMul(t *testing.T) {
	got := Cents(2400).Mul(3).Int64()
	if got != 7200 {
		t.Fatalf("Mul = %d, want 7200", got)
	}
}
