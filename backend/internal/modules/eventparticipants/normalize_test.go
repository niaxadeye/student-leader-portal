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

func TestNormalizeSocialURLs(t *testing.T) {
	t.Parallel()
	empty := "  "
	if got, err := normalizeOptionalSocialURL(socialVK, &empty); err != nil || got != nil {
		t.Fatalf("empty vk = %v %v", got, err)
	}

	vkCases := map[string]string{
		"durov":                 "https://vk.com/durov",
		"id1":                   "https://vk.com/id1",
		"vk.com/durov":          "https://vk.com/durov",
		"https://m.vk.ru/club1": "https://vk.com/club1",
		"@public123":            "https://vk.com/public123",
	}
	for input, want := range vkCases {
		got, err := normalizeOptionalSocialURL(socialVK, ptr(input))
		if err != nil || got == nil || *got != want {
			t.Errorf("vk %q = %v %v, want %q", input, got, err, want)
		}
	}

	tgCases := map[string]string{
		"durov":                            "https://t.me/durov",
		"@durov":                           "https://t.me/durov",
		"t.me/durov":                       "https://t.me/durov",
		"https://telegram.me/joinchat/AbC": "https://t.me/joinchat/AbC",
	}
	for input, want := range tgCases {
		got, err := normalizeOptionalSocialURL(socialTelegram, ptr(input))
		if err != nil || got == nil || *got != want {
			t.Errorf("tg %q = %v %v, want %q", input, got, err, want)
		}
	}

	for _, input := range []string{"https://evil.com/durov", "javascript:alert(1)", "https://vk.com"} {
		if _, err := normalizeOptionalSocialURL(socialVK, ptr(input)); err == nil {
			t.Errorf("vk %q must be rejected", input)
		}
	}
	if _, err := normalizeOptionalSocialURL(socialTelegram, ptr("https://evil.com/durov")); err == nil {
		t.Fatal("foreign telegram host must be rejected")
	}
}

func ptr(value string) *string { return &value }
