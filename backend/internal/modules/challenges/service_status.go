package challenges

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// allowedTransitions задаёт матрицу переходов статуса испытания.
// В архив можно уйти из любого не-архивного состояния.
var allowedTransitions = map[string]map[string]bool{
	StatusDraft:     {StatusPublished: true, StatusArchived: true},
	StatusPublished: {StatusClosed: true, StatusArchived: true},
	StatusClosed:    {StatusPublished: true, StatusArchived: true},
	StatusArchived:  {},
}

// Transition меняет статус испытания. При публикации фиксирует снапшот схемы.
func (s *Service) Transition(ctx context.Context, a Actor, challengeID, target string) (*Challenge, error) {
	c, err := s.adminGetForEdit(ctx, a, challengeID)
	if err != nil {
		return nil, err
	}
	if !allowedTransitions[c.Status][target] {
		return nil, ErrBadStatus
	}
	if err := s.repo.SetStatus(ctx, challengeID, target, a.UserID); err != nil {
		return nil, err
	}
	if target == StatusPublished {
		if err := s.snapshot(ctx, challengeID, c.CurrentSchemaVersion, "publish", a.UserID); err != nil {
			return nil, err
		}
	}
	s.audit.Log(ctx, a.UserID, "CHALLENGE_STATUS_CHANGED", "challenge", challengeID,
		map[string]any{"from": c.Status, "to": target})
	return s.repo.ByID(ctx, challengeID)
}

// snapshot сохраняет текущую схему испытания как версию.
func (s *Service) snapshot(ctx context.Context, challengeID string, version int, summary, actorID string) error {
	schema, err := s.SchemaJSON(ctx, challengeID)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(schema)
	if err != nil {
		return err
	}
	return s.repo.SaveSnapshot(ctx, challengeID, version, raw, summary, actorID)
}

// Duplicate копирует испытание (мета + активные поля) в новый DRAFT того же конкурса.
func (s *Service) Duplicate(ctx context.Context, a Actor, challengeID string) (*Challenge, error) {
	src, err := s.adminGetForEdit(ctx, a, challengeID)
	if err != nil {
		return nil, err
	}
	fields, err := s.repo.Fields(ctx, challengeID)
	if err != nil {
		return nil, err
	}
	copyTitle := strings.TrimSpace(src.Title) + " (копия)"
	var newID string
	for attempt := 1; attempt <= 32; attempt++ {
		nc := &Challenge{
			ContestID: src.ContestID, Title: copyTitle, Slug: copySlug(src.Title, attempt),
			ShortDescription: src.ShortDescription, FullDescription: src.FullDescription,
			Instructions: src.Instructions, OpenAt: src.OpenAt, DeadlineAt: src.DeadlineAt, CloseAt: src.CloseAt,
		}
		newID, err = s.repo.Create(ctx, nc, a.UserID)
		if err == nil {
			break
		}
		if errors.Is(err, ErrSlugTaken) {
			continue
		}
		return nil, err
	}
	if newID == "" {
		return nil, ErrSlugTaken
	}
	for i := range fields {
		f := fields[i]
		if _, err := s.repo.CreateField(ctx, newID, &f, a.UserID); err != nil {
			return nil, err
		}
	}
	s.audit.Log(ctx, a.UserID, "CHALLENGE_DUPLICATED", "challenge", newID, map[string]any{"source_id": challengeID})
	return s.repo.ByID(ctx, newID)
}

func copySlug(srcTitle string, attempt int) string {
	base := slugify(srcTitle)
	if base == "" {
		base = "challenge"
	}
	if attempt <= 1 {
		return base + "-copy"
	}
	return fmt.Sprintf("%s-copy-%d", base, attempt)
}
