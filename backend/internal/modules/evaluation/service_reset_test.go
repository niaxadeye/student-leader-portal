package evaluation

import (
	"context"
	"errors"
	"testing"
)

type fakePasswords struct{ ok bool }

func (f fakePasswords) VerifyUserPassword(_ context.Context, _, _ string) error {
	if !f.ok {
		return errors.New("mismatch")
	}
	return nil
}

func TestConfirmDestructive(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	a := Actor{UserID: "u1"}
	s := &Service{}
	if err := s.confirmDestructive(ctx, a, "secret"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("nil verifier: %v", err)
	}
	s.passwords = fakePasswords{ok: false}
	if err := s.confirmDestructive(ctx, a, ""); !errors.Is(err, ErrWrongPassword) {
		t.Fatalf("empty: %v", err)
	}
	if err := s.confirmDestructive(ctx, a, "secret"); !errors.Is(err, ErrWrongPassword) {
		t.Fatalf("bad password: %v", err)
	}
	s.passwords = fakePasswords{ok: true}
	if err := s.confirmDestructive(ctx, a, "secret"); err != nil {
		t.Fatal(err)
	}
}

func TestUniqueIDs(t *testing.T) {
	t.Parallel()
	got := uniqueIDs([]string{" a ", "b", "a", "", "b"})
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("%v", got)
	}
}

func TestResetSessionIdle(t *testing.T) {
	t.Parallel()
	now := true
	sess := Session{
		State: StateFinished, AccumulatedPauseSeconds: 12, CurrentQuestionNumber: 4,
	}
	if now {
		id := "x"
		sess.CurrentContestantUserID = &id
		sess.CurrentPerformanceID = &id
		sess.CurrentMatchID = &id
	}
	resetSessionIdle(&sess)
	if sess.State != StateNotStarted || sess.CurrentContestantUserID != nil || sess.CurrentMatchID != nil || sess.CurrentQuestionNumber != 1 {
		t.Fatalf("%+v", sess)
	}
}
