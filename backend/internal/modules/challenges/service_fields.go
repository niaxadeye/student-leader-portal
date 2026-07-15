package challenges

import (
	"context"
	"strings"
)

// FieldInput — тело создания/редактирования поля.
type FieldInput struct {
	Key         string
	Type        string
	Label       string
	Description *string
	HelpText    *string
	Placeholder *string
	Required    bool
	Settings    map[string]any
	Validation  map[string]any
	Visibility  map[string]any
}

func (in FieldInput) validate() (*Field, bool) {
	key := strings.TrimSpace(in.Key)
	ftype := strings.ToUpper(strings.TrimSpace(in.Type))
	label := strings.TrimSpace(in.Label)
	if key == "" || label == "" || !ValidFieldTypes[ftype] {
		return nil, false
	}
	return &Field{
		Key: key, Type: ftype, Label: label, Description: in.Description,
		HelpText: in.HelpText, Placeholder: in.Placeholder, Required: in.Required,
		Settings: in.Settings, Validation: in.Validation, Visibility: in.Visibility,
	}, true
}

// ListFields возвращает активные поля испытания (админ).
func (s *Service) ListFields(ctx context.Context, a Actor, challengeID string) ([]Field, error) {
	if _, err := s.AdminGet(ctx, a, challengeID); err != nil {
		return nil, err
	}
	return s.repo.Fields(ctx, challengeID)
}

// AddField добавляет поле; на опубликованном испытании версионирует схему.
func (s *Service) AddField(ctx context.Context, a Actor, challengeID string, in FieldInput) (*Field, error) {
	c, err := s.AdminGet(ctx, a, challengeID)
	if err != nil {
		return nil, err
	}
	f, ok := in.validate()
	if !ok {
		return nil, ErrValidation
	}
	id, err := s.repo.CreateField(ctx, challengeID, f, a.UserID)
	if err != nil {
		return nil, err
	}
	if err := s.versionIfPublished(ctx, c, "field added", a.UserID); err != nil {
		return nil, err
	}
	s.audit.Log(ctx, a.UserID, "CHALLENGE_FIELD_ADDED", "challenge", challengeID, map[string]any{"field_id": id})
	f.ID = id
	f.ChallengeID = challengeID
	return f, nil
}

// UpdateField меняет поле; на опубликованном испытании версионирует схему.
func (s *Service) UpdateField(ctx context.Context, a Actor, challengeID, fieldID string, in FieldInput) error {
	c, err := s.AdminGet(ctx, a, challengeID)
	if err != nil {
		return err
	}
	f, ok := in.validate()
	if !ok {
		return ErrValidation
	}
	if err := s.repo.UpdateField(ctx, challengeID, fieldID, f, a.UserID); err != nil {
		return err
	}
	if err := s.versionIfPublished(ctx, c, "field updated", a.UserID); err != nil {
		return err
	}
	s.audit.Log(ctx, a.UserID, "CHALLENGE_FIELD_UPDATED", "challenge", challengeID, map[string]any{"field_id": fieldID})
	return nil
}

// DeleteField — soft delete; на опубликованном испытании версионирует схему.
func (s *Service) DeleteField(ctx context.Context, a Actor, challengeID, fieldID string) error {
	c, err := s.AdminGet(ctx, a, challengeID)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteField(ctx, challengeID, fieldID, a.UserID); err != nil {
		return err
	}
	if err := s.versionIfPublished(ctx, c, "field deleted", a.UserID); err != nil {
		return err
	}
	s.audit.Log(ctx, a.UserID, "CHALLENGE_FIELD_DELETED", "challenge", challengeID, map[string]any{"field_id": fieldID})
	return nil
}

// ReorderFields переставляет поля по списку id.
func (s *Service) ReorderFields(ctx context.Context, a Actor, challengeID string, orderedIDs []string) error {
	c, err := s.AdminGet(ctx, a, challengeID)
	if err != nil {
		return err
	}
	if err := s.repo.ReorderFields(ctx, challengeID, orderedIDs, a.UserID); err != nil {
		return err
	}
	return s.versionIfPublished(ctx, c, "fields reordered", a.UserID)
}

// versionIfPublished для опубликованного испытания увеличивает версию схемы
// и сохраняет новый снапшот (SITE.md §11.4 — версионирование правок).
func (s *Service) versionIfPublished(ctx context.Context, c *Challenge, summary, actorID string) error {
	if c.Status != StatusPublished {
		return nil
	}
	v, err := s.repo.BumpSchemaVersion(ctx, c.ID, actorID)
	if err != nil {
		return err
	}
	return s.snapshot(ctx, c.ID, v, summary, actorID)
}
