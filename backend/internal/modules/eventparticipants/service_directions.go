package eventparticipants

import (
	"context"
	"strings"
	"unicode/utf8"
)

const maxDirectionNameRunes = 80

func (s *Service) ListDirections(ctx context.Context, actor Actor, contestID string) ([]Direction, error) {
	if err := s.ensureDirectionRead(ctx, actor, contestID); err != nil {
		return nil, err
	}
	list, err := s.repo.ListDirections(ctx, contestID)
	if list == nil {
		list = []Direction{}
	}
	return list, err
}

func (s *Service) CreateDirection(ctx context.Context, actor Actor, contestID, name string) (*Direction, error) {
	if err := s.ensureManage(ctx, actor, contestID); err != nil {
		return nil, err
	}
	display, err := normalizeDirectionName(name)
	if err != nil {
		return nil, err
	}
	direction, err := s.repo.CreateDirection(ctx, contestID, display)
	if err == nil && s.audit != nil {
		s.audit.Log(ctx, actor.UserID, "EVENT_DIRECTION_CREATED", "event_direction", direction.ID,
			map[string]any{"contest_id": contestID, "name": direction.Name})
	}
	return direction, err
}

func (s *Service) UpdateDirection(ctx context.Context, actor Actor, contestID, directionID, name string) (*Direction, error) {
	if err := s.ensureManage(ctx, actor, contestID); err != nil {
		return nil, err
	}
	display, err := normalizeDirectionName(name)
	if err != nil {
		return nil, err
	}
	direction, err := s.repo.UpdateDirection(ctx, contestID, directionID, display)
	if err == nil && s.audit != nil {
		s.audit.Log(ctx, actor.UserID, "EVENT_DIRECTION_UPDATED", "event_direction", directionID,
			map[string]any{"contest_id": contestID, "name": direction.Name})
	}
	return direction, err
}

func (s *Service) DeleteDirection(ctx context.Context, actor Actor, contestID, directionID string) error {
	if err := s.ensureManage(ctx, actor, contestID); err != nil {
		return err
	}
	err := s.repo.DeleteDirection(ctx, contestID, directionID)
	if err == nil && s.audit != nil {
		s.audit.Log(ctx, actor.UserID, "EVENT_DIRECTION_DELETED", "event_direction", directionID,
			map[string]any{"contest_id": contestID})
	}
	return err
}

func normalizeDirectionName(name string) (string, error) {
	display := strings.Join(strings.Fields(strings.TrimSpace(name)), " ")
	if display == "" || utf8.RuneCountInString(display) > maxDirectionNameRunes {
		return "", ErrValidation
	}
	return display, nil
}
