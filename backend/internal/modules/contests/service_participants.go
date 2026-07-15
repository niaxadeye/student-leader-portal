package contests

import (
	"context"
	"strings"

	"github.com/eazytech/student-leader-cabinet/internal/platform/security"
)

// Participants возвращает участников конкурса (нужен доступ к конкурсу).
func (s *Service) Participants(ctx context.Context, a Actor, contestID string) ([]Participant, error) {
	if err := s.ensureView(ctx, a, contestID); err != nil {
		return nil, err
	}
	return s.repo.Participants(ctx, contestID)
}

// AddContestantInput — данные нового конкурсанта.
type AddContestantInput struct {
	Login        string
	FullName     string
	Organization string
}

// AddContestantResult — созданный конкурсант с временным паролем (показать один раз).
type AddContestantResult struct {
	UserID       string
	Login        string
	TempPassword string
}

// AddContestant создаёт конкурсанта с временным паролем и привязывает к конкурсу.
func (s *Service) AddContestant(ctx context.Context, a Actor, contestID string, in AddContestantInput) (*AddContestantResult, error) {
	// Управление составом участников — только владелец или мега (§3.6): EDIT-админ получает 403.
	if err := s.ensureOwnerOrMega(ctx, a, contestID); err != nil {
		return nil, err
	}
	login := strings.TrimSpace(in.Login)
	name := strings.TrimSpace(in.FullName)
	if login == "" || name == "" {
		return nil, ErrValidation
	}
	temp, err := security.GenerateTempPassword()
	if err != nil {
		return nil, err
	}
	hash, err := security.HashPassword(temp)
	if err != nil {
		return nil, err
	}
	userID, err := s.repo.AddContestant(ctx, contestID, NewContestant{
		Login: login, FullName: name, Organization: strings.TrimSpace(in.Organization), PasswordHash: hash,
	})
	if err != nil {
		return nil, err
	}
	s.audit.Log(ctx, a.UserID, "CONTESTANT_ADDED", "contest", contestID,
		map[string]any{"user_id": userID, "login": login})
	return &AddContestantResult{UserID: userID, Login: login, TempPassword: temp}, nil
}

// RemoveContestant отвязывает участника от конкурса (soft).
func (s *Service) RemoveContestant(ctx context.Context, a Actor, contestID, userID string) error {
	if err := s.ensureOwnerOrMega(ctx, a, contestID); err != nil {
		return err
	}
	if err := s.repo.RemoveParticipant(ctx, contestID, userID); err != nil {
		return err
	}
	s.audit.Log(ctx, a.UserID, "CONTESTANT_REMOVED", "contest", contestID,
		map[string]any{"user_id": userID})
	return nil
}
