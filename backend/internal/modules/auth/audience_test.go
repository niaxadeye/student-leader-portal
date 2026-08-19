package auth

import "testing"

func TestRoleAllowedForAudience(t *testing.T) {
	t.Parallel()
	staff := []Role{{Code: "ADMIN"}}
	contestant := []Role{{Code: "CONTESTANT"}}
	both := []Role{{Code: "STAFF"}, {Code: "CONTESTANT"}}

	if !roleAllowedForAudience("", staff) || !roleAllowedForAudience("admin", staff) {
		t.Fatal("admin audience should accept ADMIN")
	}
	if roleAllowedForAudience("contestant", staff) {
		t.Fatal("contestant audience should reject ADMIN")
	}
	if !roleAllowedForAudience("contestant", contestant) {
		t.Fatal("contestant audience should accept CONTESTANT")
	}
	if roleAllowedForAudience("admin", contestant) {
		t.Fatal("admin audience should reject CONTESTANT")
	}
	if !roleAllowedForAudience("admin", both) || !roleAllowedForAudience("contestant", both) {
		t.Fatal("dual-role user can enter either portal")
	}
	if roleAllowedForAudience("unknown", staff) {
		t.Fatal("unknown audience must fail")
	}
}
