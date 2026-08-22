package evaluation

import (
	"testing"
	"time"
)

func TestTimerRemainingRunning(t *testing.T) {
	t.Parallel()
	started := time.Date(2026, 8, 21, 18, 0, 0, 0, time.UTC)
	now := started.Add(90 * time.Second)
	dur := 480
	got := timerRemaining(now, &dur, &started, nil, 0)
	if got == nil || *got != 390 {
		t.Fatalf("got %v want 390", got)
	}
}

func TestTimerRemainingPausedFreezes(t *testing.T) {
	t.Parallel()
	started := time.Date(2026, 8, 21, 18, 0, 0, 0, time.UTC)
	paused := started.Add(30 * time.Second)
	now := started.Add(5 * time.Minute)
	dur := 120
	got := timerRemaining(now, &dur, &started, &paused, 0)
	if got == nil || *got != 90 {
		t.Fatalf("paused remaining: %v", got)
	}
}

func TestTimerRemainingOvertime(t *testing.T) {
	t.Parallel()
	started := time.Date(2026, 8, 21, 18, 0, 0, 0, time.UTC)
	now := started.Add(150 * time.Second)
	dur := 120
	got := timerRemaining(now, &dur, &started, nil, 0)
	if got == nil || *got != -30 {
		t.Fatalf("overtime: %v want -30", got)
	}
}

func TestPhaseElapsedIncludesOvertime(t *testing.T) {
	t.Parallel()
	started := time.Date(2026, 8, 21, 18, 0, 0, 0, time.UTC)
	now := started.Add(500 * time.Second)
	got := phaseElapsed(now, &started, nil, 0)
	if got == nil || *got != 500 {
		t.Fatalf("elapsed: %v want 500", got)
	}
}

func TestPhaseElapsedPausedFreezes(t *testing.T) {
	t.Parallel()
	started := time.Date(2026, 8, 21, 18, 0, 0, 0, time.UTC)
	paused := started.Add(90 * time.Second)
	now := started.Add(10 * time.Minute)
	got := phaseElapsed(now, &started, &paused, 0)
	if got == nil || *got != 90 {
		t.Fatalf("paused elapsed: %v want 90", got)
	}
}

func TestSpeechPhaseActive(t *testing.T) {
	t.Parallel()
	speechID := "p-speech"
	qID := "p-q"
	phases := []PhaseTemplate{
		{ID: speechID, MapsToState: StateLive},
		{ID: qID, MapsToState: StateQuestions},
	}
	sess := Session{State: StateLive, CurrentPhaseID: &speechID}
	if !speechPhaseActive(&sess, phases) {
		t.Fatal("LIVE must count as speech")
	}
	sess.State = StatePaused
	if !speechPhaseActive(&sess, phases) {
		t.Fatal("paused speech phase must still record")
	}
	sess.CurrentPhaseID = &qID
	if speechPhaseActive(&sess, phases) {
		t.Fatal("paused questions must not count as speech")
	}
	sess.State = StateApplause
	sess.CurrentPhaseID = nil
	if speechPhaseActive(&sess, phases) {
		t.Fatal("applause is not speech")
	}
}

func TestSessionLocked(t *testing.T) {
	t.Parallel()
	if sessionLocked(StateNotStarted) || sessionLocked(StatePreparing) {
		t.Fatal("draft session must be writable")
	}
	if !sessionLocked(StateLive) || !sessionLocked(StateFinished) {
		t.Fatal("live/finished must lock scheme")
	}
	if drawLocked(StateNotStarted) || drawLocked(StatePreparing) {
		t.Fatal("draw must stay editable before start")
	}
	if !drawLocked(StateLive) || !drawLocked(StatePaused) {
		t.Fatal("draw must lock during trial")
	}
}
