package evaluation

import "testing"

func TestScoreInRange(t *testing.T) {
	t.Parallel()
	if !scoreInRange(8, 1, 10) {
		t.Fatal("8 in 1-10")
	}
	if scoreInRange(0, 1, 10) {
		t.Fatal("0 out")
	}
	if scoreInRange(11, 1, 10) {
		t.Fatal("11 out")
	}
	if scoreInRange(8.5, 1, 10) {
		t.Fatal("fraction not allowed on integer scale")
	}
	if !scoreInRange(2.5, 1.5, 3.5) {
		t.Fatal("fraction ok on fractional scale")
	}
}

func TestScoringEditable(t *testing.T) {
	t.Parallel()
	if !scoringEditable(EditWhileActive, StateLive) {
		t.Fatal("live")
	}
	if scoringEditable(EditWhileActive, StateFinished) {
		t.Fatal("finished should lock")
	}
	if !scoringEditable(EditAlways, StateFinished) {
		t.Fatal("always")
	}
}

func TestResolveScorecardContestant(t *testing.T) {
	t.Parallel()
	cur := "live"
	if got := resolveScorecardContestant("  other  ", &cur); got != "other" {
		t.Fatalf("manual: %s", got)
	}
	if got := resolveScorecardContestant("", &cur); got != "live" {
		t.Fatalf("follow: %s", got)
	}
	if got := resolveScorecardContestant("", nil); got != "" {
		t.Fatalf("empty: %s", got)
	}
}

func TestMean(t *testing.T) {
	t.Parallel()
	if mean(nil) != nil {
		t.Fatal("empty")
	}
	got := mean([]float64{10, 20})
	if got == nil || *got != 15 {
		t.Fatalf("got %v", got)
	}
	sum := sumFloats([]float64{10, 9})
	if sum == nil || *sum != 19 {
		t.Fatalf("sum %v", sum)
	}
}

func f64(v float64) *float64 { return &v }

func TestCompetitionRanks(t *testing.T) {
	t.Parallel()
	got := competitionRanks([]*float64{f64(10), f64(9), f64(9), f64(8)})
	want := []int{1, 2, 2, 4}
	for i, w := range want {
		if got[i] == nil || *got[i] != w {
			t.Fatalf("%d: got %v want %d", i, got[i], w)
		}
	}
	got = competitionRanks([]*float64{f64(10), f64(9), f64(9), f64(9), f64(8)})
	want = []int{1, 2, 2, 2, 5}
	for i, w := range want {
		if got[i] == nil || *got[i] != w {
			t.Fatalf("triple: %d got %v want %d", i, got[i], w)
		}
	}
	got = competitionRanks([]*float64{nil, f64(10), f64(10)})
	if got[0] != nil {
		t.Fatal("unscored")
	}
	if got[1] == nil || *got[1] != 1 || got[2] == nil || *got[2] != 1 {
		t.Fatalf("tied first: %v %v", got[1], got[2])
	}
}
