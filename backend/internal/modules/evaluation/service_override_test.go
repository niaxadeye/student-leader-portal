package evaluation

import (
	"strings"
	"testing"
)

func TestNormalizeReason(t *testing.T) {
	t.Parallel()
	if _, err := normalizeReason("  аб  "); err == nil {
		t.Fatal("too short")
	}
	got, err := normalizeReason("  ошибка ввода жюри  ")
	if err != nil || got != "ошибка ввода жюри" {
		t.Fatalf("got %q %v", got, err)
	}
	if _, err := normalizeReason(strings.Repeat("я", MaxOverrideReason+1)); err == nil {
		t.Fatal("too long")
	}
}

func TestScoresEqual(t *testing.T) {
	t.Parallel()
	if !scoresEqual(nil, nil) {
		t.Fatal("nil")
	}
	a, b := 8.0, 8.0
	if !scoresEqual(&a, &b) {
		t.Fatal("equal")
	}
	c := 7.0
	if scoresEqual(&a, &c) || scoresEqual(&a, nil) {
		t.Fatal("not equal")
	}
}
