package service

import "testing"

func TestIDOrNilAndZero(t *testing.T) {
	if got := idOrNil(0); got != nil {
		t.Fatalf("idOrNil(0) = %v, want nil", got)
	}
	if got := idOrNil(-3); got != nil {
		t.Fatalf("idOrNil(-3) = %v, want nil", got)
	}
	got := idOrNil(42)
	if got == nil || *got != 42 {
		t.Fatalf("idOrNil(42) = %v, want 42", got)
	}
	if got := idOrZero(nil); got != 0 {
		t.Fatalf("idOrZero(nil) = %d, want 0", got)
	}
	v := int64(7)
	if got := idOrZero(&v); got != 7 {
		t.Fatalf("idOrZero(&7) = %d, want 7", got)
	}
}
