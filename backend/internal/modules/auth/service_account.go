package auth

import (
	"context"
	"strings"

	"github.com/eazytech/student-leader-cabinet/internal/modules/eventpermissions"
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

// VerifyUserPassword подтверждает пароль уже вошедшего пользователя (опасные действия).
func (s *Service) VerifyUserPassword(ctx context.Context, userID, password string) error {
	if strings.TrimSpace(password) == "" {
		return ErrWrongOldPassword
	}
	u, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		return ErrWrongOldPassword
	}
	if u.Status == StatusBlocked {
		return ErrAccountBlocked
	}
	if err := security.VerifyPassword(password, u.PasswordHash); err != nil {
		s.audit.Log(ctx, userID, "AUTH_STEPUP_FAILED", "user", userID, nil)
		return ErrWrongOldPassword
	}
	return nil
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

// Me возвращает пользователя, его роли и staff-permissions на мероприятия.
func (s *Service) Me(ctx context.Context, userID string) (*User, []Role, []eventpermissions.Grant, error) {
	u, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		return nil, nil, nil, err
	}
	roles, err := s.repo.RolesByUser(ctx, userID)
	if err != nil {
		return nil, nil, nil, err
	}
	grants := []eventpermissions.Grant{}
	if s.staff != nil {
		grants, err = s.staff.GrantsForUser(ctx, userID)
		if err != nil {
			return nil, nil, nil, err
		}
		if grants == nil {
			grants = []eventpermissions.Grant{}
		}
	}
	if s.hasRemoteJury(ctx, userID) {
		hasJury := false
		for _, role := range roles {
			if role.Code == "JURY" {
				hasJury = true
				break
			}
		}
		if !hasJury {
			roles = append(roles, Role{Code: "JURY"})
		}
	}
	return u, roles, grants, nil
}
