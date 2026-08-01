package prescription

import "testing"

func TestCanTransitTo(t *testing.T) {
	// 合法流转
	valid := [][2]Status{
		{StatusPendingReview, StatusReviewedPassed},
		{StatusPendingReview, StatusReviewedRejected},
		{StatusPendingReview, StatusCancelled},
		{StatusReviewedPassed, StatusDispensing},
		{StatusReviewedPassed, StatusCancelled},
		{StatusReviewedRejected, StatusPendingReview},
		{StatusReviewedRejected, StatusCancelled},
		{StatusDispensing, StatusDispensed},
		{StatusDispensing, StatusCancelled},
		{StatusDispensed, StatusReturned},
	}
	for _, v := range valid {
		if _, ok := v[0].CanTransitTo(v[1]); !ok {
			t.Fatalf("允许流转 %s→%s 判定失败", v[0], v[1])
		}
	}
	// 非法流转
	invalid := [][2]Status{
		{StatusDispensed, StatusDispensing},
		{StatusDispensed, StatusCancelled},
		{StatusReturned, StatusDispensed},
		{StatusCancelled, StatusPendingReview},
		{StatusPendingReview, StatusDispensing},
	}
	for _, v := range invalid {
		if _, ok := v[0].CanTransitTo(v[1]); ok {
			t.Fatalf("非法流转 %s→%s 未被拦截", v[0], v[1])
		}
	}
}

func TestTerminal(t *testing.T) {
	if !StatusReturned.IsTerminal() || !StatusCancelled.IsTerminal() {
		t.Fatal("returned/cancelled 应为终态")
	}
	if StatusDispensed.IsTerminal() {
		t.Fatal("dispensed 不是终态（可退药）")
	}
}
