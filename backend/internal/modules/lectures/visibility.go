package lectures

import (
	"strings"
	"unicode/utf8"
)

const (
	maxPeopleNameRunes = 120
	maxPeoplePerRole   = 20
	maxLocationRunes   = 300
)

func lectureAllowsParticipant(restrictedDirectionIDs []string, participantDirectionID *string) bool {
	if len(restrictedDirectionIDs) == 0 {
		return true
	}
	if participantDirectionID == nil || *participantDirectionID == "" {
		return false
	}
	for _, id := range restrictedDirectionIDs {
		if id == *participantDirectionID {
			return true
		}
	}
	return false
}

func uniqueDirectionIDs(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func normalizePeopleNames(names []string) ([]string, error) {
	seen := make(map[string]struct{}, len(names))
	out := make([]string, 0, len(names))
	for _, raw := range names {
		name := strings.Join(strings.Fields(strings.TrimSpace(raw)), " ")
		if name == "" {
			continue
		}
		if utf8.RuneCountInString(name) > maxPeopleNameRunes {
			return nil, ErrValidation
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, name)
	}
	if len(out) > maxPeoplePerRole {
		return nil, ErrValidation
	}
	return out, nil
}

func normalizeOptionalText(value *string, maxRunes int) (*string, error) {
	if value == nil {
		return nil, nil
	}
	text := strings.Join(strings.Fields(strings.TrimSpace(*value)), " ")
	if text == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(text) > maxRunes {
		return nil, ErrValidation
	}
	return &text, nil
}
