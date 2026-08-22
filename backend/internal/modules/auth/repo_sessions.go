package auth

import (
	"context"
	"errors"
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

// RotateRefreshAtomically блокирует предъявленный refresh и его сессию, затем
// в одной транзакции одноразово потребляет старое звено и создаёт новое.
// Повторное предъявление уже использованного звена отзывает всё семейство до
// возврата ошибки вызывающему коду.
func (r *Repo) RotateRefreshAtomically(
	ctx context.Context,
	oldTokenHash, newJTI, newTokenHash string,
	newExpiresAt, now time.Time,
) (*refreshRow, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	row := &refreshRow{}
	err = tx.QueryRow(ctx, `
		SELECT rt.id, rt.session_id, s.user_id, s.token_family_id, rt.used_at, rt.revoked_at,
		       rt.expires_at, s.revoked_at, s.expires_at
		FROM refresh_tokens rt JOIN auth_sessions s ON s.id = rt.session_id
		WHERE rt.token_hash = $1
		FOR UPDATE OF rt, s`, oldTokenHash).
		Scan(&row.ID, &row.SessionID, &row.UserID, &row.FamilyID, &row.UsedAt, &row.RevokedAt,
			&row.ExpiresAt, &row.SessionRevoke, &row.SessionExp)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRefreshReused // неизвестный токен трактуем как компрометацию
	}
	if err != nil {
		return nil, err
	}

	if row.UsedAt != nil || row.RevokedAt != nil {
		if err := revokeFamilyTx(ctx, tx, row.FamilyID, "refresh_reuse_detected", now); err != nil {
			return nil, err
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}
		return row, ErrRefreshReused
	}
	if row.SessionRevoke != nil || !row.SessionExp.After(now) || !row.ExpiresAt.After(now) {
		return row, ErrSessionExpired
	}

	command, err := tx.Exec(ctx, `
		UPDATE refresh_tokens SET used_at = $2
		WHERE id = $1 AND used_at IS NULL AND revoked_at IS NULL`, row.ID, now)
	if err != nil {
		return nil, err
	}
	if command.RowsAffected() != 1 {
		if err := revokeFamilyTx(ctx, tx, row.FamilyID, "refresh_reuse_detected", now); err != nil {
			return nil, err
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}
		return row, ErrRefreshReused
	}

	if _, err = tx.Exec(ctx, `
		INSERT INTO refresh_tokens (session_id, jti, token_hash, rotated_from_id, expires_at)
		VALUES ($1,$2,$3,$4,$5)`, row.SessionID, newJTI, newTokenHash, row.ID, newExpiresAt); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `UPDATE auth_sessions SET last_used_at = $2 WHERE id = $1`, row.SessionID, now); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return row, nil
}

// RevokeFamily отзывает все сессии семейства и их токены (reuse detection).
func (r *Repo) RevokeFamily(ctx context.Context, familyID, reason string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := revokeFamilyTx(ctx, tx, familyID, reason, time.Now()); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func revokeFamilyTx(ctx context.Context, tx pgx.Tx, familyID, reason string, now time.Time) error {
	if _, err := tx.Exec(ctx, `
		UPDATE auth_sessions
		SET revoked_at = COALESCE(revoked_at, $3), revoke_reason = COALESCE(revoke_reason, $2)
		WHERE token_family_id = $1`, familyID, reason, now); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `
		UPDATE refresh_tokens SET revoked_at = COALESCE(revoked_at, $2)
		WHERE session_id IN (
			SELECT id FROM auth_sessions WHERE token_family_id = $1
		)`, familyID, now)
	return err
}
