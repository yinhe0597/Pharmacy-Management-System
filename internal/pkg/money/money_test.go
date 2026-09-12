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

// TestSplitPriceRoundingDrift 记录并固化「逐片计价」的已知取整残差（docs/04 §拆零计价）：
// 1 盒 24 片、盒价 1000 分 → 单片 42 分 → 24×42 = 1008 分，整单按片发药比盒价高 8 分。
func TestSplitPriceRoundingDrift(t *testing.T) {
	packPrice := Cents(1000)
	unitPrice := packPrice.SplitPrice(24)
	if unitPrice != 42 {
		t.Fatalf("单片价 = %d, want 42", unitPrice)
	}
	total := unitPrice.Mul(24)
	if total != 1008 {
		t.Fatalf("整盒拆零累计 = %d, want 1008（已知取整残差）", total)
	}
	if total-packPrice != 8 {
		t.Fatalf("取整残差 = %d, want 8", int64(total-packPrice))
	}
}

// TestItemAmountEdgeCases ItemAmount 在脏数据/边界包装量下不得 panic（除零）。
func TestItemAmountEdgeCases(t *testing.T) {
	cases := []struct {
		name           string
		isSplit        bool
		isSplitAllowed bool
		packSize       int
		unitPrice      int64
		retailPrice    int64
		qty            int64
		want           int64
	}{
		{"强制拆零", true, true, 24, 100, 2400, 30, 3000},
		{"混合口径-2盒6片", false, true, 24, 100, 2400, 54, 5400},
		{"混合口径-纯整盒", false, true, 24, 100, 2400, 48, 4800},
		{"packSize=0 不除零（按盒价）", false, true, 0, 100, 2400, 5, 12000},
		{"packSize=-1 不除零（按盒价）", false, true, -1, 100, 2400, 5, 12000},
		{"packSize=1 盒=片（按拆零价）", false, true, 1, 100, 2400, 5, 500},
		{"不拆零（按盒价）", false, false, 24, 100, 2400, 5, 12000},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ItemAmount(c.isSplit, c.isSplitAllowed, c.packSize, c.unitPrice, c.retailPrice, c.qty)
			if got != c.want {
				t.Fatalf("ItemAmount(...) = %d, want %d", got, c.want)
			}
		})
	}
}

// TestParseYuan 精确十进制解析（不经过浮点）。
func TestParseYuan(t *testing.T) {
	cases := []struct {
		in      string
		want    int64
		wantErr bool
	}{
		{"12.34", 1234, false},
		{"12", 1200, false},
		{"0.05", 5, false},
		{".5", 50, false},
		{" 12.3 ", 1230, false},
		{"12.345", 1235, false}, // 第三位半入
		{"12.344", 1234, false},
		{"-3.50", -350, false},
		{"+1.00", 100, false},
		{"", 0, true},
		{"abc", 0, true},
		{"1.2.3", 0, true},
		{"12元", 0, true},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got, err := ParseYuan(c.in)
			if c.wantErr {
				if err == nil {
					t.Fatalf("ParseYuan(%q) 期望报错，实际得到 %d", c.in, int64(got))
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseYuan(%q) 报错: %v", c.in, err)
			}
			if int64(got) != c.want {
				t.Fatalf("ParseYuan(%q) = %d, want %d", c.in, int64(got), c.want)
			}
		})
	}
}
