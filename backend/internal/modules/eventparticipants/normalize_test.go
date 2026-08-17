package eventparticipants

import "testing"

func TestNormalizeFullName(t *testing.T) {
	t.Parallel()
	tests := map[string]string{
		"  Иванов   Иван Иванович ": "иванов иван иванович",
		"СЁМИНА АЛЁНА":              "семина алена",
		"\tПетров\nПётр ":           "петров петр",
		"":                          "",
	}
	for input, want := range tests {
		if got := NormalizeFullName(input); got != want {
			t.Errorf("NormalizeFullName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestNormalizeOptionalIdentifier(t *testing.T) {
	t.Parallel()
	empty := "   "
	if got := normalizeOptionalIdentifier(&empty); got != nil {
		t.Fatalf("empty identifier = %q, want nil", *got)
	}
	value := "  001-ABC  "
	got := normalizeOptionalIdentifier(&value)
	if got == nil || *got != "001-ABC" {
		t.Fatalf("identifier = %v, want 001-ABC", got)
	}
}
