package evaluation

import "testing"

func TestVisiblePhasesHidesScoring(t *testing.T) {
	t.Parallel()
	in := []PhaseTemplate{
		{ID: "1", MapsToState: StateLive, ScoringAllowed: false},
		{ID: "2", MapsToState: StateQuestions},
		{ID: "3", MapsToState: StateScoring},
	}
	got := visiblePhases(in)
	if len(got) != 2 || got[0].MapsToState != StateLive || !got[0].ScoringAllowed {
		t.Fatalf("got %+v", got)
	}
}

func TestNextContestantID(t *testing.T) {
	t.Parallel()
	list := []LiveContestant{{UserID: "a"}, {UserID: "b"}, {UserID: "c"}}
	if got := nextContestantID(list, "a"); got != "b" {
		t.Fatalf("got %s", got)
	}
	if got := nextContestantID(list, "c"); got != "" {
		t.Fatalf("last: %s", got)
	}
}
