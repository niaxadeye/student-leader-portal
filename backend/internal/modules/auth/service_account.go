package auth

import (
	"context"

	"github.com/eazytech/student-leader-cabinet/internal/platform/security"
)

// Logout отзывает текущую сессию.
func (s *Service) Logout(ctx context.Context, userID, sessionID string) error {
	s.audit.Log(ctx, userID, "AUTH_LOGOUT", "session", sessionID, nil)
	return s.repo.RevokeSession(ctx, userID, sessionID, "logout")
}

// LogoutAll отзывает все сессии пользователя.
func (s *Service) LogoutAll(ctx context.Context, userID string) error {
	s.audit.Log(ctx, userID, "AUTH_LOGOUT_ALL", "user", userID, nil)
	return s.repo.RevokeAllSessions(ctx, userID, "logout_all")
}

// ChangePassword меняет пароль после проверки старого и отзывает все сессии (SITE.md §16).
func (s *Service) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	if len(newPassword) < minPasswordLen {
		return ErrPasswordTooShort
	}
	u, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		return err
	}
	if err := security.VerifyPassword(oldPassword, u.PasswordHash); err != nil {
		return ErrWrongOldPassword
	}
	hash, err := security.HashPassword(newPassword)
	if err != nil {
		return err
	}
	if err := s.repo.UpdatePassword(ctx, userID, hash); err != nil {
		return err
	}
	// Смена пароля отзывает все сессии — пользователь перелогинится.
	s.audit.Log(ctx, userID, "AUTH_PASSWORD_CHANGED", "user", userID, nil)
	return s.repo.RevokeAllSessions(ctx, userID, "password_changed")
}

// Sessions возвращает активные сессии, помечая текущую.
func (s *Service) Sessions(ctx context.Context, userID, currentSessionID string) ([]Session, error) {
	list, err := s.repo.ListSessions(ctx, userID)
	if err != nil {
		return nil, err
	}
	for i := range list {
		list[i].Current = list[i].ID == currentSessionID
	}
	return list, nil
}

// RevokeSession отзывает конкретную сессию пользователя.
func (s *Service) RevokeSession(ctx context.Context, userID, sessionID string) error {
	s.audit.Log(ctx, userID, "AUTH_SESSION_REVOKED", "session", sessionID, nil)
	return s.repo.RevokeSession(ctx, userID, sessionID, "user_revoked")
}

// Me возвращает пользователя и его роли.
func (s *Service) Me(ctx context.Context, userID string) (*User, []Role, error) {
	u, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	roles, err := s.repo.RolesByUser(ctx, userID)
	return u, roles, err
}
