package useradmin

import (
	"errors"
	"testing"
)

func ptr(s string) *string { return &s }

func TestCanManageUserMatrix(t *testing.T) {
	t.Parallel()
	mega := Actor{UserID: "mega", Role: "MEGA_ADMIN"}
	superA := Actor{UserID: "super-a", Role: "SUPER_ADMIN"}
	admin := Actor{UserID: "admin", Role: "ADMIN"}
	staff := Actor{UserID: "staff", Role: "STAFF"}

	ownAdmin := AccessTarget{ID: "u-admin", CreatedBy: ptr("super-a"), Roles: []string{"ADMIN"}}
	ownContestant := AccessTarget{ID: "u-c", CreatedBy: ptr("super-a"), Roles: []string{"CONTESTANT"}}
	foreignContestant := AccessTarget{ID: "u-cf", CreatedBy: ptr("super-b"), Roles: []string{"CONTESTANT"}}
	foreignAdmin := AccessTarget{ID: "u-af", CreatedBy: ptr("super-b"), Roles: []string{"ADMIN"}}
	otherSuper := AccessTarget{ID: "super-b", CreatedBy: ptr("mega"), Roles: []string{"SUPER_ADMIN"}}
	megaTarget := AccessTarget{ID: "mega", CreatedBy: nil, Roles: []string{"MEGA_ADMIN"}}
	privInContest := AccessTarget{ID: "u-priv", CreatedBy: ptr("super-b"), Roles: []string{"ADMIN", "CONTESTANT"}}

	cases := []struct {
		name      string
		actor     Actor
		target    AccessTarget
		inContest bool
		action    ManageAction
		ok        bool
	}{
		{"ADMIN reset own contestant", admin, ownContestant, true, ActionReset, false},
		{"ADMIN block any", admin, ownContestant, true, ActionStatus, false},
		{"STAFF reset", staff, ownContestant, true, ActionReset, false},
		{"SUPER A cannot reset SUPER B user", superA, foreignAdmin, false, ActionReset, false},
		{"SUPER A cannot reset other SUPER", superA, otherSuper, false, ActionReset, false},
		{"SUPER A cannot reset MEGA", superA, megaTarget, false, ActionReset, false},
		{"SUPER A can reset own contestant", superA, ownContestant, false, ActionReset, true},
		{"SUPER A can reset own ADMIN", superA, ownAdmin, false, ActionReset, true},
		{"SUPER A can reset contestant in own contest", superA, foreignContestant, true, ActionReset, true},
		{"SUPER A cannot grant role to foreign contestant in own contest", superA, foreignContestant, true, ActionGrant, false},
		{"SUPER A cannot reset privileged account just because they share a contest", superA, privInContest, true, ActionReset, false},
		{"SUPER A cannot view foreign ADMIN", superA, foreignAdmin, false, ActionView, false},
		{"MEGA can reset foreign ADMIN", mega, foreignAdmin, false, ActionReset, true},
		{"MEGA can block other SUPER", mega, otherSuper, false, ActionStatus, true},
		{"MEGA can grant on any user", mega, foreignContestant, false, ActionGrant, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := CanManageUser(tc.actor, tc.target, tc.inContest, tc.action)
			if tc.ok && err != nil {
				t.Fatalf("want allow, got %v", err)
			}
			if !tc.ok && !errors.Is(err, ErrForbidden) {
				t.Fatalf("want forbidden, got %v", err)
			}
		})
	}
}

func TestNormScopeRejectsGlobalAdmin(t *testing.T) {
	t.Parallel()
	if _, ok := normScope(AssignRoleInput{Role: "ADMIN", ScopeType: "GLOBAL"}); ok {
		t.Fatal("global ADMIN must be rejected")
	}
	if _, ok := normScope(AssignRoleInput{Role: "ADMIN", ScopeType: "CONTEST", ScopeID: "c1"}); ok {
		t.Fatal("ADMIN+CONTEST without access_level must be rejected")
	}
	got, ok := normScope(AssignRoleInput{Role: "ADMIN", ScopeType: "CONTEST", ScopeID: "c1", AccessLevel: "EDIT"})
	if !ok || got.AccessLevel != AccessEdit {
		t.Fatalf("ADMIN contest EDIT: %+v ok=%v", got, ok)
	}
}
