package auth

import "context"

// ListSessions возвращает активные сессии пользователя (SITE.md §16: управление сессиями).
func (r *Repo) ListSessions(ctx context.Context, userID string) ([]Session, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, COALESCE(user_agent,''), COALESCE(ip_hash,''),
		       last_used_at, expires_at, created_at
		FROM auth_sessions
		WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > now()
		ORDER BY last_used_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Session
	for rows.Next() {
		var s Session
		if err := rows.Scan(&s.ID, &s.UserID, &s.UserAgent, &s.IPHash,
			&s.LastUsedAt, &s.ExpiresAt, &s.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// RevokeSession отзывает конкретную сессию пользователя и её токены.
func (r *Repo) RevokeSession(ctx context.Context, userID, sessionID, reason string) error {
	_, err := r.pool.Exec(ctx, `
		WITH s AS (
			UPDATE auth_sessions SET revoked_at = now(), revoke_reason = $3
			WHERE id = $2 AND user_id = $1 AND revoked_at IS NULL RETURNING id
		)
		UPDATE refresh_tokens SET revoked_at = now()
		WHERE session_id IN (SELECT id FROM s) AND revoked_at IS NULL`, userID, sessionID, reason)
	return err
}

// RevokeAllSessions отзывает все сессии пользователя (logout-all, сброс пароля).
func (r *Repo) RevokeAllSessions(ctx context.Context, userID, reason string) error {
	_, err := r.pool.Exec(ctx, `
		WITH s AS (
			UPDATE auth_sessions SET revoked_at = now(), revoke_reason = $2
			WHERE user_id = $1 AND revoked_at IS NULL RETURNING id
		)
		UPDATE refresh_tokens SET revoked_at = now()
		WHERE session_id IN (SELECT id FROM s) AND revoked_at IS NULL`, userID, reason)
	return err
}

// SessionActive проверяет, что сессия жива (для middleware).
func (r *Repo) SessionActive(ctx context.Context, sessionID string) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM auth_sessions
		WHERE id = $1 AND revoked_at IS NULL AND expires_at > now())`, sessionID).Scan(&ok)
	return ok, err
}
