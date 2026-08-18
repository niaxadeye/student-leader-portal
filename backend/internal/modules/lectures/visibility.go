package lectures

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
