package service

import (
	"testing"

	"yaofang/internal/model"
)

// TestItemAmount 处方快照价金额计算（与 prepareItems 同口径，docs/13 §4.3）。
func TestItemAmount(t *testing.T) {
	cases := []struct {
		name string
		item model.PrescriptionItem
		qty  int64
		want int64
	}{
		{
			name: "强制拆零：全部按拆零价",
			item: model.PrescriptionItem{IsSplit: true, IsSplitAllowed: true, PackSize: 24,
				UnitPrice: 100, RetailPrice: 2400},
			qty:  12,
			want: 12 * 100,
		},
		{
			name: "混合发药：整盒按盒价 + 零头按拆零价",
			item: model.PrescriptionItem{IsSplit: false, IsSplitAllowed: true, PackSize: 24,
				UnitPrice: 100, RetailPrice: 2400},
			qty:  36, // 1盒24片 + 12片零头
			want: 1*2400 + 12*100,
		},
		{
			name: "混合发药：恰好整盒",
			item: model.PrescriptionItem{IsSplit: false, IsSplitAllowed: true, PackSize: 24,
				UnitPrice: 100, RetailPrice: 2400},
			qty:  48,
			want: 2 * 2400,
		},
		{
			name: "不可拆零：全部按盒价（pack_size=1 等价）",
			item: model.PrescriptionItem{IsSplit: false, IsSplitAllowed: false, PackSize: 1,
				UnitPrice: 2400, RetailPrice: 2400},
			qty:  3,
			want: 3 * 2400,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := itemAmount(&c.item, c.qty)
			if got != c.want {
				t.Fatalf("itemAmount = %d, want %d", got, c.want)
			}
		})
	}
}
