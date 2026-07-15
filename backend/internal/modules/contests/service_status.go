package contests

import (
	"context"
	"strings"
)

// Update редактирует конкурс (нужен доступ к конкурсу).
func (s *Service) Update(ctx context.Context, a Actor, id string, in CreateInput) (*Contest, error) {
	if err := s.ensureAccess(ctx, a, id); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, ErrValidation
	}
	tz := in.Timezone
	if tz == "" {
		tz = "Europe/Moscow"
	}
	c := &Contest{Name: name, Description: in.Desc, StartAt: in.StartAt, EndAt: in.EndAt, Timezone: tz}
	if err := s.repo.Update(ctx, id, c, a.UserID); err != nil {
		return nil, err
	}
	s.audit.Log(ctx, a.UserID, "CONTEST_UPDATED", "contest", id, nil)
	return s.repo.ByID(ctx, id)
}

// allowedTransitions задаёт допустимые переходы статуса конкурса (SITE.md §9).
var allowedTransitions = map[string]map[string]bool{
	StatusDraft:    {StatusActive: true, StatusArchived: true},
	StatusActive:   {StatusFinished: true, StatusArchived: true},
	StatusFinished: {StatusArchived: true},
	StatusArchived: {},
}

// Transition меняет статус конкурса с проверкой допустимости перехода.
func (s *Service) Transition(ctx context.Context, a Actor, id, target string) (*Contest, error) {
	if err := s.ensureAccess(ctx, a, id); err != nil {
		return nil, err
	}
	cur, err := s.repo.ByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if cur.Status == target {
		return cur, nil
	}
	if !allowedTransitions[cur.Status][target] {
		return nil, ErrBadStatus
	}
	if err := s.repo.SetStatus(ctx, id, target, a.UserID); err != nil {
		return nil, err
	}
	s.audit.Log(ctx, a.UserID, "CONTEST_STATUS_CHANGED", "contest", id,
		map[string]any{"from": cur.Status, "to": target})
	return s.repo.ByID(ctx, id)
}
