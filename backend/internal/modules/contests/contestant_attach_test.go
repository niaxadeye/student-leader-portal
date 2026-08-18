package contests

import (
	"errors"
	"testing"
)

func TestCanAttachExistingContestant(t *testing.T) {
	t.Parallel()
	owner := Actor{UserID: "super-a", IsSuper: true}
	mega := Actor{UserID: "mega", IsMega: true}
	cb := func(id string) *string { return &id }

	own := existingAccount{ID: "u1", CreatedBy: cb("super-a"), Roles: []string{"CONTESTANT"}}
	foreign := existingAccount{ID: "u2", CreatedBy: cb("super-b"), Roles: []string{"CONTESTANT"}}
	admin := existingAccount{ID: "u3", CreatedBy: cb("super-a"), Roles: []string{"ADMIN"}}
	megaAcc := existingAccount{ID: "u4", CreatedBy: nil, Roles: []string{"MEGA_ADMIN"}}
	emptyOwn := existingAccount{ID: "u5", CreatedBy: cb("super-a"), Roles: nil}

	if err := canAttachExistingContestant(owner, own); err != nil {
		t.Fatalf("own contestant: %v", err)
	}
	if err := canAttachExistingContestant(owner, emptyOwn); err != nil {
		t.Fatalf("own empty roles: %v", err)
	}
	if err := canAttachExistingContestant(owner, foreign); !errors.Is(err, ErrLoginConflict) {
		t.Fatalf("foreign contestant: %v", err)
	}
	if err := canAttachExistingContestant(owner, admin); !errors.Is(err, ErrLoginConflict) {
		t.Fatalf("own admin hijack: %v", err)
	}
	if err := canAttachExistingContestant(mega, megaAcc); !errors.Is(err, ErrLoginConflict) {
		t.Fatalf("mega must not attach privileged login: %v", err)
	}
	if err := canAttachExistingContestant(mega, foreign); err != nil {
		t.Fatalf("mega attach foreign contestant: %v", err)
	}
}
