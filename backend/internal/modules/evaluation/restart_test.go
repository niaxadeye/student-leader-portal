package evaluation

import "testing"

func TestResetSessionIdleClearsMatch(t *testing.T) {
	t.Parallel()
	id := "m1"
	sess := Session{State: StateLive, CurrentMatchID: &id, CurrentQuestionNumber: 7}
	resetSessionIdle(&sess)
	if sess.State != StateNotStarted || sess.CurrentMatchID != nil || sess.CurrentQuestionNumber != 1 {
		t.Fatalf("%+v", sess)
	}
}

func TestRestorePhaseOrLive(t *testing.T) {
	t.Parallel()
	live := "p-live"
	phases := []PhaseTemplate{
		{ID: live, MapsToState: StateQuestions},
	}
	sess := Session{CurrentPhaseID: &live, State: StateFinished}
	restorePhaseOrLive(&sess, phases)
	if sess.State != StateQuestions {
		t.Fatalf("got %s", sess.State)
	}
	sess.CurrentPhaseID = nil
	restorePhaseOrLive(&sess, phases)
	if sess.State != StateLive {
		t.Fatalf("empty phase: %s", sess.State)
	}
}
