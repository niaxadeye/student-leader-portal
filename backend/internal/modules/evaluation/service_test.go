package evaluation

import "testing"

func TestNormalizeScheme(t *testing.T) {
	t.Parallel()
	in, err := normalizeScheme(SchemeInput{Type: "criteria_scoring"}, "Самопрезентация")
	if err != nil {
		t.Fatal(err)
	}
	if in.Type != TypeCriteriaScoring || in.ScoringUnit != UnitPoints || in.Name != "Самопрезентация" {
		t.Fatalf("got %+v", in)
	}
	lives, err := normalizeScheme(SchemeInput{Type: TypeEliminationLives}, "2 к 1")
	if err != nil {
		t.Fatal(err)
	}
	if lives.ScoringUnit != UnitLives || lives.StartingLives == nil || *lives.StartingLives != 3 {
		t.Fatalf("2k1 defaults: %+v", lives)
	}
	op := "jury-id"
	withOp, err := normalizeScheme(SchemeInput{Type: TypeEliminationLives, OperatorUserID: &op}, "2 к 1")
	if err != nil {
		t.Fatal(err)
	}
	if operatorIDOf(&Scheme{SettingsJSON: withOp.SettingsJSON, Type: TypeEliminationLives}) != op {
		t.Fatalf("operator settings: %s", withOp.SettingsJSON)
	}
	tooMany := 99
	if _, err := normalizeScheme(SchemeInput{Type: TypeEliminationLives, StartingLives: &tooMany}, "x"); err == nil {
		t.Fatal("expected lives validation")
	}
	max := 100.0
	num, err := normalizeScheme(SchemeInput{Type: TypeNumericResult, MaxScore: &max}, "Квест")
	if err != nil {
		t.Fatal(err)
	}
	if num.ScoringUnit != UnitPoints || num.MaxScore == nil || *num.MaxScore != 100 || num.MinScore == nil || *num.MinScore != 0 {
		t.Fatalf("numeric defaults: %+v", num)
	}
	if _, err := normalizeScheme(SchemeInput{Type: TypeNumericResult}, "x"); err == nil {
		t.Fatal("expected max_score required")
	}
	if HasLiveSession(TypeNumericResult) || !HasLiveSession(TypeCriteriaScoring) || HasLiveSession(TypeRemoteCriteria) {
		t.Fatal("live session flags")
	}
	if !UsesCriteria(TypeRemoteCriteria) || !UsesCriteria(TypeCriteriaScoring) || UsesCriteria(TypeNumericResult) {
		t.Fatal("criteria flags")
	}
	if !ExclusiveChallengeJury(TypeRemoteCriteria) || ExclusiveChallengeJury(TypeCriteriaScoring) {
		t.Fatal("exclusive jury flags")
	}
	remote, err := normalizeScheme(SchemeInput{Type: TypeRemoteCriteria}, "Эссе")
	if err != nil {
		t.Fatal(err)
	}
	if remote.Type != TypeRemoteCriteria || remote.ScoringUnit != UnitPoints {
		t.Fatalf("remote defaults: %+v", remote)
	}
	if _, err := normalizeScheme(SchemeInput{Type: "UNKNOWN"}, "x"); err == nil {
		t.Fatal("expected validation")
	}
}

func TestNormalizeCriterion(t *testing.T) {
	t.Parallel()
	in, err := normalizeCriterion(CriterionInput{Title: "Содержание"})
	if err != nil {
		t.Fatal(err)
	}
	if in.MinScore != 1 || in.MaxScore != 10 || in.Weight != 1 {
		t.Fatalf("defaults: %+v", in)
	}
	if _, err := normalizeCriterion(CriterionInput{Title: "x", MinScore: 5, MaxScore: 1}); err == nil {
		t.Fatal("expected min>max fail")
	}
}

func TestDefaultUnit(t *testing.T) {
	t.Parallel()
	if defaultUnit(TypeEliminationLives) != UnitLives {
		t.Fatal("lives")
	}
	if defaultUnit(TypeCriteriaScoring) != UnitPoints {
		t.Fatal("points")
	}
}
