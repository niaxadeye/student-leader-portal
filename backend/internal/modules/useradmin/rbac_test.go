package useradmin

import "testing"

func TestCanCreateRole(t *testing.T) {
	t.Parallel()
	mega := Actor{UserID: "m", Role: "MEGA_ADMIN"}
	super := Actor{UserID: "s", Role: "SUPER_ADMIN"}
	admin := Actor{UserID: "a", Role: "ADMIN"}
	staff := Actor{UserID: "f", Role: "STAFF"}

	if !canCreateRole(mega, "STAFF") || !canCreateRole(mega, "SUPER_ADMIN") {
		t.Fatal("mega must create any role")
	}
	if !canCreateRole(super, "STAFF") || !canCreateRole(super, "ADMIN") || !canCreateRole(super, "CONTESTANT") {
		t.Fatal("super must create STAFF/ADMIN/CONTESTANT")
	}
	if canCreateRole(super, "SUPER_ADMIN") || canCreateRole(super, "MEGA_ADMIN") {
		t.Fatal("super must not create SUPER/MEGA")
	}
	if canCreateRole(admin, "STAFF") || canCreateRole(staff, "STAFF") || canCreateRole(staff, "ADMIN") {
		t.Fatal("admin/staff must not create users")
	}
}

func TestNormScopeRejectsContestScopedStaff(t *testing.T) {
	t.Parallel()
	if _, ok := normScope(AssignRoleInput{Role: "STAFF", ScopeType: "CONTEST", ScopeID: "c1"}); ok {
		t.Fatal("STAFF must be GLOBAL; contest access is event_staff_permissions")
	}
	got, ok := normScope(AssignRoleInput{Role: "STAFF", ScopeType: "GLOBAL"})
	if !ok || got.ScopeType != ScopeGlobal || got.AccessLevel != "" {
		t.Fatalf("STAFF global: %+v ok=%v", got, ok)
	}
}
