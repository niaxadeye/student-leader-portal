package eventparticipants

import "strings"

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
