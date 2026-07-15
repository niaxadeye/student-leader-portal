package auth

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo struct {
	pool *pgxpool.Pool
}

func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

// UserByLogin находит активного (не удалённого) пользователя по логину.
func (r *Repo) UserByLogin(ctx context.Context, login string) (*User, error) {
	u := &User{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, login, password_hash, full_name, status, must_change_password,
		       failed_login_count, locked_until
		FROM users WHERE login = $1 AND deleted_at IS NULL`, login).
		Scan(&u.ID, &u.Login, &u.PasswordHash, &u.FullName, &u.Status,
			&u.MustChangePassword, &u.FailedLoginCount, &u.LockedUntil)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvalidCredentials
	}
	return u, err
}

func (r *Repo) UserByID(ctx context.Context, id string) (*User, error) {
	u := &User{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, login, password_hash, full_name, status, must_change_password,
		       failed_login_count, locked_until
		FROM users WHERE id = $1 AND deleted_at IS NULL`, id).
		Scan(&u.ID, &u.Login, &u.PasswordHash, &u.FullName, &u.Status,
			&u.MustChangePassword, &u.FailedLoginCount, &u.LockedUntil)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvalidCredentials
	}
	return u, err
}

// RolesByUser возвращает роли пользователя со scope.
func (r *Repo) RolesByUser(ctx context.Context, userID string) ([]Role, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT rl.code, ur.scope_type, ur.scope_id
		FROM user_roles ur JOIN roles rl ON rl.id = ur.role_id
		WHERE ur.user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var roles []Role
	for rows.Next() {
		var role Role
		if err := rows.Scan(&role.Code, &role.ScopeType, &role.ScopeID); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}

// RecordLoginSuccess сбрасывает счётчик неудач и проставляет last_login_at.
func (r *Repo) RecordLoginSuccess(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE users SET failed_login_count = 0, locked_until = NULL,
		       last_login_at = now(), updated_at = now() WHERE id = $1`, userID)
	return err
}

// RecordLoginFailure инкрементит счётчик и при превышении лимита ставит блокировку.
func (r *Repo) RecordLoginFailure(ctx context.Context, userID string, lockFor time.Duration, threshold int) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE users SET failed_login_count = failed_login_count + 1,
		       locked_until = CASE WHEN failed_login_count + 1 >= $2 THEN now() + $3::interval ELSE locked_until END,
		       updated_at = now()
		WHERE id = $1`, userID, threshold, lockFor.String())
	return err
}

// UpdatePassword меняет хэш, снимает флаг обязательной смены.
func (r *Repo) UpdatePassword(ctx context.Context, userID, hash string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE users SET password_hash = $2, must_change_password = FALSE,
		       password_changed_at = now(), updated_at = now() WHERE id = $1`, userID, hash)
	return err
}
