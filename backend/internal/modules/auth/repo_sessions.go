package auth

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// CreateSession + первый refresh-токен в одной транзакции (SITE.md §17: транзакции).
func (r *Repo) CreateSession(ctx context.Context, s *Session, familyID, jti, tokenHash string, refreshExp time.Time) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx, `
		INSERT INTO auth_sessions (user_id, token_family_id, user_agent, ip_hash, expires_at)
		VALUES ($1,$2,$3,$4,$5) RETURNING id, created_at`,
		s.UserID, familyID, s.UserAgent, s.IPHash, s.ExpiresAt).Scan(&s.ID, &s.CreatedAt)
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `
		INSERT INTO refresh_tokens (session_id, jti, token_hash, expires_at)
		VALUES ($1,$2,$3,$4)`, s.ID, jti, tokenHash, refreshExp); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// refreshRow — состояние refresh-токена для проверки при ротации.
type refreshRow struct {
	ID            string
	SessionID     string
	UserID        string
	FamilyID      string
	UsedAt        *time.Time
	RevokedAt     *time.Time
	ExpiresAt     time.Time
	SessionRevoke *time.Time
	SessionExp    time.Time
}

// FindRefresh по хэшу возвращает токен вместе с состоянием сессии.
func (r *Repo) FindRefresh(ctx context.Context, tokenHash string) (*refreshRow, error) {
	row := &refreshRow{}
	err := r.pool.QueryRow(ctx, `
		SELECT rt.id, rt.session_id, s.user_id, s.token_family_id, rt.used_at, rt.revoked_at,
		       rt.expires_at, s.revoked_at, s.expires_at
		FROM refresh_tokens rt JOIN auth_sessions s ON s.id = rt.session_id
		WHERE rt.token_hash = $1`, tokenHash).
		Scan(&row.ID, &row.SessionID, &row.UserID, &row.FamilyID, &row.UsedAt, &row.RevokedAt,
			&row.ExpiresAt, &row.SessionRevoke, &row.SessionExp)
	if err == pgx.ErrNoRows {
		return nil, ErrRefreshReused // неизвестный токен трактуем как компрометацию
	}
	return row, err
}

// RotateRefresh помечает старый токен использованным и создаёт новый (одна транзакция).
func (r *Repo) RotateRefresh(ctx context.Context, oldID, sessionID, jti, tokenHash string, exp time.Time) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `UPDATE refresh_tokens SET used_at = now() WHERE id = $1`, oldID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `
		INSERT INTO refresh_tokens (session_id, jti, token_hash, rotated_from_id, expires_at)
		VALUES ($1,$2,$3,$4,$5)`, sessionID, jti, tokenHash, oldID, exp); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE auth_sessions SET last_used_at = now() WHERE id = $1`, sessionID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// RevokeFamily отзывает все сессии семейства и их токены (reuse detection).
func (r *Repo) RevokeFamily(ctx context.Context, familyID, reason string) error {
	_, err := r.pool.Exec(ctx, `
		WITH fam AS (
			UPDATE auth_sessions SET revoked_at = now(), revoke_reason = $2
			WHERE token_family_id = $1 AND revoked_at IS NULL RETURNING id
		)
		UPDATE refresh_tokens SET revoked_at = now()
		WHERE session_id IN (SELECT id FROM fam) AND revoked_at IS NULL`, familyID, reason)
	return err
}
