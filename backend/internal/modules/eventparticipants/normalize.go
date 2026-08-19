package eventparticipants

import (
	"net/url"
	"strings"
	"unicode/utf8"
)

const maxSocialURLRunes = 500

type socialNetwork int

const (
	socialVK socialNetwork = iota
	socialTelegram
)

// NormalizeFullName приводит ФИО к форме для поиска: trim, один пробел,
// нижний регистр и объединение ё/е. Исходное ФИО хранится отдельно.
func NormalizeFullName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "ё", "е")
	return strings.Join(strings.Fields(value), " ")
}

func cleanFullName(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func normalizeOptionalIdentifier(value *string) *string {
	if value == nil {
		return nil
	}
	clean := strings.TrimSpace(*value)
	if clean == "" {
		return nil
	}
	return &clean
}

func normalizeOptionalSocialURL(network socialNetwork, value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	raw := strings.TrimSpace(*value)
	if raw == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(raw) > maxSocialURLRunes {
		return nil, ErrValidation
	}
	switch network {
	case socialVK:
		return normalizeVKURL(raw)
	case socialTelegram:
		return normalizeTelegramURL(raw)
	default:
		return nil, ErrValidation
	}
}

func normalizeVKURL(raw string) (*string, error) {
	raw = strings.TrimPrefix(strings.TrimSpace(raw), "@")
	if looksLikeVKScreenName(raw) {
		raw = "https://vk.com/" + raw
	}
	parsed, err := parseHTTPURL(raw)
	if err != nil {
		return nil, err
	}
	host := canonicalHost(parsed.Hostname())
	if host != "vk.com" && host != "vk.ru" {
		return nil, ErrValidation
	}
	return canonicalSocialURL("https://vk.com", parsed)
}

func normalizeTelegramURL(raw string) (*string, error) {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "@") {
		username := strings.TrimPrefix(raw, "@")
		if !telegramUsername(username) {
			return nil, ErrValidation
		}
		canonical := "https://t.me/" + username
		return &canonical, nil
	}
	if telegramUsername(raw) {
		canonical := "https://t.me/" + raw
		return &canonical, nil
	}
	parsed, err := parseHTTPURL(raw)
	if err != nil {
		return nil, err
	}
	switch canonicalHost(parsed.Hostname()) {
	case "t.me", "telegram.me", "telegram.dog":
	default:
		return nil, ErrValidation
	}
	return canonicalSocialURL("https://t.me", parsed)
}

func parseHTTPURL(raw string) (*url.URL, error) {
	if !strings.Contains(raw, "://") {
		raw = "https://" + strings.TrimPrefix(raw, "//")
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.User != nil {
		return nil, ErrValidation
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, ErrValidation
	}
	if parsed.Hostname() == "" {
		return nil, ErrValidation
	}
	return parsed, nil
}

func canonicalSocialURL(origin string, parsed *url.URL) (*string, error) {
	path := strings.Trim(parsed.Path, "/")
	if path == "" || strings.Contains(path, "..") {
		return nil, ErrValidation
	}
	canonical := origin + "/" + path
	if parsed.RawQuery != "" {
		canonical += "?" + parsed.RawQuery
	}
	return &canonical, nil
}

func canonicalHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	host = strings.TrimPrefix(host, "www.")
	host = strings.TrimPrefix(host, "m.")
	return host
}

func looksLikeVKScreenName(value string) bool {
	if value == "" || strings.ContainsAny(value, "/:?#") {
		return false
	}
	lower := strings.ToLower(value)
	if strings.HasPrefix(lower, "id") || strings.HasPrefix(lower, "club") ||
		strings.HasPrefix(lower, "public") || strings.HasPrefix(lower, "event") {
		suffix := strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(lower, "id"), "club"), "public"), "event")
		if suffix != "" && digitsOnly(suffix) {
			return true
		}
	}
	if len(value) < 2 || len(value) > 32 {
		return false
	}
	for _, r := range value {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '.') {
			return false
		}
	}
	return true
}

func telegramUsername(value string) bool {
	if len(value) < 5 || len(value) > 32 {
		return false
	}
	for _, r := range value {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
			return false
		}
	}
	return true
}

func digitsOnly(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
