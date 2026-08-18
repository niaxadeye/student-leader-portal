// Package useradmin — админ-действия над учётными записями (SITE.md §5.1–5.2, §19).
package useradmin

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/eazytech/student-leader-cabinet/internal/platform/security"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrRoleNotFound = errors.New("role not found")
	ErrValidation   = errors.New("validation error")
	ErrForbidden    = errors.New("forbidden")
)

// Auditor пишет события аудита.
type Auditor interface {
	Log(ctx context.Context, actorUserID, action, entityType, entityID string, meta map[string]any)
}

// repository — граница доступа к пользователям. Реальный Repo и тестовые фейки.
type repository interface {
	List(ctx context.Context, f ListFilter) ([]User, int, error)
	ByID(ctx context.Context, id string) (*User, error)
	Create(ctx context.Context, nu NewUser, role, scopeType, scopeID, accessLevel string) (string, error)
	UpdateProfile(ctx context.Context, id, fullName string, email, org *string) error
	AssignRole(ctx context.Context, userID, role, scopeType, scopeID, accessLevel string) error
	RemoveRole(ctx context.Context, userID, role, scopeType, scopeID string) error
	AccessTarget(ctx context.Context, userID string) (*AccessTarget, error)
	OwnsContestant(ctx context.Context, actorID, userID string) (bool, error)
	ContestOwnedBy(ctx context.Context, contestID, userID string) (bool, error)
	SetPassword(ctx context.Context, userID, hash string) error
	SetStatus(ctx context.Context, userID, status string) error
	RevokeSessions(ctx context.Context, userID, reason string) error
}

type Service struct {
	repo  repository
	audit Auditor
}

func NewService(pool *pgxpool.Pool, audit Auditor) *Service {
	return &Service{repo: NewRepo(pool), audit: audit}
}

func newService(repo repository, audit Auditor) *Service {
	return &Service{repo: repo, audit: audit}
}

func (s *Service) log(ctx context.Context, actorID, action, entityID string, meta map[string]any) {
	if s.audit == nil {
		return
	}
	s.audit.Log(ctx, actorID, action, "user", entityID, meta)
}

// ensureManage загружает цель и применяет CanManageUser. Запрет пишется в аудит.
func (s *Service) ensureManage(ctx context.Context, a Actor, userID string, action ManageAction) error {
	t, err := s.repo.AccessTarget(ctx, userID)
	if err != nil {
		return err
	}
	inContest := false
	if a.IsSuper() && !a.IsMega() {
		inContest, err = s.repo.OwnsContestant(ctx, a.UserID, userID)
		if err != nil {
			return err
		}
	}
	if err := CanManageUser(a, *t, inContest, action); err != nil {
		s.log(ctx, a.UserID, "USER_ACCESS_DENIED", userID, map[string]any{"action": string(action)})
		return err
	}
	return nil
}

// ResetPassword ставит новый временный пароль и must_change_password=TRUE.
// Возвращает временный пароль (показать один раз). Завершает все сессии пользователя.
func (s *Service) ResetPassword(ctx context.Context, a Actor, userID string) (string, error) {
	if err := s.ensureManage(ctx, a, userID, ActionReset); err != nil {
		return "", err
	}
	temp, err := security.GenerateTempPassword()
	if err != nil {
		return "", err
	}
	hash, err := security.HashPassword(temp)
	if err != nil {
		return "", err
	}
	if err := s.repo.SetPassword(ctx, userID, hash); err != nil {
		return "", err
	}
	_ = s.repo.RevokeSessions(ctx, userID, "password_reset")
	s.log(ctx, a.UserID, "USER_PASSWORD_RESET", userID, nil)
	return temp, nil
}

// SetStatus блокирует/разблокирует пользователя. При блокировке завершает сессии.
func (s *Service) SetStatus(ctx context.Context, a Actor, userID, status string) error {
	if status != "BLOCKED" && status != "ACTIVE" {
		return ErrValidation
	}
	if err := s.ensureManage(ctx, a, userID, ActionStatus); err != nil {
		return err
	}
	if err := s.repo.SetStatus(ctx, userID, status); err != nil {
		return err
	}
	if status == "BLOCKED" {
		_ = s.repo.RevokeSessions(ctx, userID, "blocked")
	}
	s.log(ctx, a.UserID, "USER_STATUS_CHANGED", userID, map[string]any{"status": status})
	return nil
}
