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

// ContestantGet возвращает опубликованное испытание с полями для участника конкурса.
func (s *Service) ContestantGet(ctx context.Context, a Actor, challengeID string) (*Challenge, []Field, error) {
	c, err := s.repo.ByID(ctx, challengeID)
	if err != nil {
		return nil, nil, err
	}
	if err := s.ensureParticipant(ctx, a, c); err != nil {
		return nil, nil, err
	}
	fields, err := s.repo.Fields(ctx, challengeID)
	if err != nil {
		return nil, nil, err
	}
	return c, fields, nil
}

// ContestantList возвращает опубликованные испытания конкурса для участника.
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
	return s.repo.ListForContestant(ctx, contestID, a.UserID)
}

// ensureParticipant: испытание видно только PUBLISHED и только участнику (или суперадмину).
func (s *Service) ensureParticipant(ctx context.Context, a Actor, c *Challenge) error {
	if c.Status != StatusPublished {
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
	return nil
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
