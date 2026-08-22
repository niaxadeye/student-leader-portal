package challenges

import "context"

// SchemaJSON собирает активную схему испытания (для preview и снапшотов).
func (s *Service) SchemaJSON(ctx context.Context, challengeID string) (map[string]any, error) {
	fields, err := s.repo.Fields(ctx, challengeID)
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(fields))
	for i := range fields {
		items = append(items, FieldMap(&fields[i]))
	}
	return map[string]any{"fields": items}, nil
}

// AdminSchemaPreview возвращает схему для админского preview (с проверкой доступа).
func (s *Service) AdminSchemaPreview(ctx context.Context, a Actor, challengeID string) (map[string]any, error) {
	if _, err := s.AdminGet(ctx, a, challengeID); err != nil {
		return nil, err
	}
	return s.SchemaJSON(ctx, challengeID)
}

// ContestantGet возвращает испытание с полями для участника конкурса.
func (s *Service) ContestantGet(ctx context.Context, a Actor, challengeID string) (*Challenge, []Field, error) {
	c, err := s.repo.ByID(ctx, challengeID)
	if err != nil {
		return nil, nil, err
	}
	if err := s.ensureContestantChallenge(ctx, a, c); err != nil {
		return nil, nil, err
	}
	_, _, briefing, err := s.loadResolved(ctx, challengeID, a.UserID)
	if err != nil {
		return nil, nil, err
	}
	if contestantSeesBriefing(briefing) {
		if !briefing.Visible {
			briefing.BodyText = ""
			briefing.Files = []BriefingFile{}
		}
		c.Briefing = &briefing
	}
	if !(c.Status == StatusPublished && c.AcceptsSubmissions) {
		return c, []Field{}, nil
	}
	fields, err := s.repo.Fields(ctx, challengeID)
	if err != nil {
		return nil, nil, err
	}
	return c, fields, nil
}

// ContestantList возвращает видимые испытания конкурса для участника.
func (s *Service) ContestantList(ctx context.Context, a Actor, contestID string) ([]Challenge, error) {
	if !a.IsSuper {
		ok, err := s.repo.IsParticipant(ctx, a.UserID, contestID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrForbidden
		}
	}
	list, err := s.repo.ListForContestant(ctx, contestID, a.UserID)
	if err != nil {
		return nil, err
	}
	for i := range list {
		_, _, briefing, err := s.loadResolved(ctx, list[i].ID, a.UserID)
		if err != nil {
			return nil, err
		}
		if contestantSeesBriefing(briefing) {
			if !briefing.Visible {
				briefing.BodyText = ""
				briefing.Files = []BriefingFile{}
			}
			list[i].Briefing = &briefing
		}
	}
	return list, nil
}

// ensureContestantChallenge: не архив, участник конкурса (или суперадмин).
func (s *Service) ensureContestantChallenge(ctx context.Context, a Actor, c *Challenge) error {
	if c.Status == StatusArchived {
		return ErrNotFound
	}
	if a.IsSuper {
		return nil
	}
	ok, err := s.repo.IsParticipant(ctx, a.UserID, c.ContestID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	if c.Status == StatusPublished && c.AcceptsSubmissions {
		return nil
	}
	_, _, briefing, err := s.loadResolved(ctx, c.ID, a.UserID)
	if err != nil {
		return err
	}
	if contestantSeesBriefing(briefing) {
		return nil
	}
	return ErrNotFound
}

// FieldMap сериализует поле в JSON-совместимую структуру (SITE.md §11.3).
func FieldMap(f *Field) map[string]any {
	return map[string]any{
		"id": f.ID, "key": f.Key, "type": f.Type, "label": f.Label,
		"description": f.Description, "help_text": f.HelpText, "placeholder": f.Placeholder,
		"required": f.Required, "sort_order": f.SortOrder,
		"settings": orEmpty(f.Settings), "validation": orEmpty(f.Validation),
		"visibility": orEmpty(f.Visibility),
	}
}

func orEmpty(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	return m
}
