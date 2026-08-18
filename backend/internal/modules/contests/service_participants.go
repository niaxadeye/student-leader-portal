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
	Created      bool
}

// AddContestant создаёт конкурсанта с временным паролем и привязывает к конкурсу.
// Существующий логин не захватывается: свой CONTESTANT-only — тихая привязка без смены пароля;
// чужой или привилегированный — 409 без UUID.
func (s *Service) AddContestant(ctx context.Context, a Actor, contestID string, in AddContestantInput) (*AddContestantResult, error) {
	if err := s.ensureOwnerOrMega(ctx, a, contestID); err != nil {
		return nil, err
	}
	login := strings.TrimSpace(in.Login)
	name := strings.TrimSpace(in.FullName)
	if login == "" || name == "" {
		return nil, ErrValidation
	}
	existing, err := s.repo.LookupLogin(ctx, login)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		if err := canAttachExistingContestant(a, *existing); err != nil {
			return nil, err
		}
		if err := s.repo.AttachContestant(ctx, contestID, existing.ID); err != nil {
			return nil, err
		}
		s.audit.Log(ctx, a.UserID, "CONTESTANT_ATTACHED", "contest", contestID,
			map[string]any{"user_id": existing.ID, "login": login})
		return &AddContestantResult{UserID: existing.ID, Login: login, Created: false}, nil
	}
	temp, err := security.GenerateTempPassword()
	if err != nil {
		return nil, err
	}
	hash, err := security.HashPassword(temp)
	if err != nil {
		return nil, err
	}
	userID, err := s.repo.InsertContestantUser(ctx, a.UserID, NewContestant{
		Login: login, FullName: name, Organization: strings.TrimSpace(in.Organization), PasswordHash: hash,
	})
	if err != nil {
		return nil, err
	}
	if err := s.repo.AttachContestant(ctx, contestID, userID); err != nil {
		return nil, err
	}
	s.audit.Log(ctx, a.UserID, "CONTESTANT_ADDED", "contest", contestID,
		map[string]any{"user_id": userID, "login": login})
	return &AddContestantResult{UserID: userID, Login: login, TempPassword: temp, Created: true}, nil
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
