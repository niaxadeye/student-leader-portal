// Package useradmin — админ-действия над учётными записями (SITE.md §5.1–5.2, §19).
package useradmin

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/eazytech/student-leader-cabinet/internal/platform/security"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrRoleNotFound = errors.New("role not found")
	ErrValidation   = errors.New("validation error")
)

// Auditor пишет события аудита.
type Auditor interface {
	Log(ctx context.Context, actorUserID, action, entityType, entityID string, meta map[string]any)
}

type Service struct {
	pool  *pgxpool.Pool
	repo  *Repo
	audit Auditor
}

func NewService(pool *pgxpool.Pool, audit Auditor) *Service {
	return &Service{pool: pool, repo: NewRepo(pool), audit: audit}
}

// ResetPassword ставит новый временный пароль и must_change_password=TRUE.
// Возвращает временный пароль (показать один раз). Завершает все сессии пользователя.
func (s *Service) ResetPassword(ctx context.Context, actorID, userID string) (string, error) {
	temp, err := security.GenerateTempPassword()
	if err != nil {
		return "", err
	}
	hash, err := security.HashPassword(temp)
	if err != nil {
		return "", err
	}
	ct, err := s.pool.Exec(ctx, `
		UPDATE users SET password_hash=$2, must_change_password=TRUE,
		       password_changed_at=now(), failed_login_count=0, locked_until=NULL,
		       updated_at=now()
		WHERE id=$1 AND deleted_at IS NULL`, userID, hash)
	if err != nil {
		return "", err
	}
	if ct.RowsAffected() == 0 {
		return "", ErrUserNotFound
	}
	_, _ = s.pool.Exec(ctx, `UPDATE auth_sessions SET revoked_at=now() WHERE user_id=$1 AND revoked_at IS NULL`, userID)
	s.audit.Log(ctx, actorID, "USER_PASSWORD_RESET", "user", userID, nil)
	return temp, nil
}

// SetStatus блокирует/разблокирует пользователя. При блокировке завершает сессии.
func (s *Service) SetStatus(ctx context.Context, actorID, userID, status string) error {
	ct, err := s.pool.Exec(ctx, `
		UPDATE users SET status=$2, updated_at=now()
		WHERE id=$1 AND deleted_at IS NULL`, userID, status)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	if status == "BLOCKED" {
		_, _ = s.pool.Exec(ctx, `UPDATE auth_sessions SET revoked_at=now() WHERE user_id=$1 AND revoked_at IS NULL`, userID)
	}
	s.audit.Log(ctx, actorID, "USER_STATUS_CHANGED", "user", userID, map[string]any{"status": status})
	return nil
}

// exists для дружелюбной 404 (не используется напрямую, но полезно в тестах).
func (s *Service) exists(ctx context.Context, userID string) bool {
	var id string
	err := s.pool.QueryRow(ctx, `SELECT id FROM users WHERE id=$1`, userID).Scan(&id)
	return !errors.Is(err, pgx.ErrNoRows) && err == nil
}
