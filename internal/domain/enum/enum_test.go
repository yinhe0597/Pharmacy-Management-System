package enum

import "testing"

func TestIsValidRole(t *testing.T) {
	for _, r := range AllRoles {
		if !IsValidRole(r) {
			t.Fatalf("合法角色 %s 校验失败", r)
		}
	}
	if IsValidRole("superadmin") || IsValidRole("") {
		t.Fatal("非法角色不应通过校验")
	}
}

func TestIsPharmacistRole(t *testing.T) {
	allowed := []string{RoleAdmin, RolePharmacyDirector, RolePharmacist}
	for _, r := range allowed {
		if !IsPharmacistRole(r) {
			t.Fatalf("药师级角色 %s 校验失败", r)
		}
	}
	denied := []string{RoleDoctor, RoleNurse, RoleBuyer, RoleFinance, ""}
	for _, r := range denied {
		if IsPharmacistRole(r) {
			t.Fatalf("非药师级角色 %s 不应通过", r)
		}
	}
}

func TestIsSpecialControlled(t *testing.T) {
	if !IsSpecialControlled(SpecialControlNarcotic) || !IsSpecialControlled(SpecialControlPsycho) {
		t.Fatal("麻醉/精神应受五专管控")
	}
	if IsSpecialControlled(SpecialControlNone) || IsSpecialControlled(SpecialControlToxic) || IsSpecialControlled(SpecialControlRadio) {
		t.Fatal("无/毒性/放射性不应走麻精五专")
	}
}

func TestRoleGroups(t *testing.T) {
	// 护士无处方开立权；采购员仅在采购组
	for _, g := range []struct {
		name   string
		groups [][]string
		role   string
		want   bool
	}{
		{"nurse 在 PharmacyStaff", [][]string{PharmacyStaff}, RoleNurse, true},
		{"nurse 不在 ClinicalStaff", [][]string{ClinicalStaff}, RoleNurse, false},
		{"buyer 在 PurchaseStaff", [][]string{PurchaseStaff}, RoleBuyer, true},
		{"buyer 不在 PharmacyStaff", [][]string{PharmacyStaff}, RoleBuyer, false},
		{"finance 在 ReportAccess", [][]string{ReportAccess}, RoleFinance, true},
		{"director 在 UserAdmin", [][]string{UserAdmin}, RolePharmacyDirector, true},
	} {
		contains := false
		for _, r := range g.groups[0] {
			if r == g.role {
				contains = true
				break
			}
		}
		if contains != g.want {
			t.Fatalf("%s 期望 %v, got %v", g.name, g.want, contains)
		}
	}
}
