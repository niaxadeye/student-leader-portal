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
	if !canCreateRole(super, "STAFF") || !canCreateRole(super, "ADMIN") || !canCreateRole(super, "JURY") || !canCreateRole(super, "REMOTE_JURY") || !canCreateRole(super, "CONTESTANT") {
		t.Fatal("super must create STAFF/ADMIN/JURY/REMOTE_JURY/CONTESTANT")
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

func TestNormScopeJuryRequiresContest(t *testing.T) {
	t.Parallel()
	if _, ok := normScope(AssignRoleInput{Role: "JURY", ScopeType: "GLOBAL"}); ok {
		t.Fatal("JURY must not be GLOBAL")
	}
	got, ok := normScope(AssignRoleInput{Role: "JURY", ScopeType: "CONTEST", ScopeID: "c1"})
	if !ok || got.AccessLevel != "" || got.ScopeID != "c1" {
		t.Fatalf("JURY contest: %+v ok=%v", got, ok)
	}
	if _, ok := normScope(AssignRoleInput{Role: "REMOTE_JURY", ScopeType: "GLOBAL"}); ok {
		t.Fatal("REMOTE_JURY must not be GLOBAL")
	}
	got, ok = normScope(AssignRoleInput{Role: "REMOTE_JURY", ScopeType: "CONTEST", ScopeID: "c1"})
	if !ok || got.AccessLevel != "" || got.ScopeID != "c1" {
		t.Fatalf("REMOTE_JURY contest: %+v ok=%v", got, ok)
	}
}
