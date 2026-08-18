package eventparticipants

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

func (r *Repo) CreateSession(
	ctx context.Context,
	contestID, participantID, tokenHash, userAgent, ipHash string,
	expiresAt time.Time,
) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx, `
		INSERT INTO participant_sessions
		  (contest_id, event_participant_id, token_hash, user_agent, ip_hash, expires_at)
		VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
		contestID, participantID, tokenHash, userAgent, ipHash, expiresAt).Scan(&id)
	return id, err
}

// AuthenticateSession валидирует session, participant и event одним консистентным
// чтением, затем обновляет last_activity_at в той же транзакции.
func (r *Repo) AuthenticateSession(ctx context.Context, tokenHash string) (*Principal, error) {
	var principal Principal
	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			SELECT s.id,
			       p.id, p.contest_id, p.full_name, p.full_name_normalized, p.birth_date,
			       p.union_card_number, p.sks_barcode, p.status, p.created_at, p.updated_at,
			       p.archived_at, p.direction_id, d.name,
			       c.id, c.slug, c.name, c.status, c.timezone
			FROM participant_sessions s
			JOIN event_participants p ON p.id=s.event_participant_id AND p.contest_id=s.contest_id
			JOIN contests c ON c.id=s.contest_id
			LEFT JOIN event_directions d ON d.id=p.direction_id AND d.contest_id=p.contest_id
			WHERE s.token_hash=$1 AND s.revoked_at IS NULL AND s.expires_at>now()
			  AND p.status='ACTIVE' AND c.status='ACTIVE'
			FOR UPDATE OF s`, tokenHash).Scan(
			&principal.SessionID,
			&principal.Participant.ID, &principal.Participant.ContestID,
			&principal.Participant.FullName, &principal.Participant.FullNameNormalized,
			&principal.Participant.BirthDate, &principal.Participant.UnionCardNumber,
			&principal.Participant.SKSBarcode, &principal.Participant.Status,
			&principal.Participant.CreatedAt, &principal.Participant.UpdatedAt,
			&principal.Participant.ArchivedAt,
			&principal.Participant.DirectionID, &principal.Participant.DirectionName,
			&principal.Event.ID, &principal.Event.Slug, &principal.Event.Name,
			&principal.Event.Status, &principal.Event.Timezone,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrSessionExpired
		}
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE participant_sessions SET last_activity_at=now() WHERE id=$1`, principal.SessionID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return &principal, nil
}

func (r *Repo) RevokeSession(ctx context.Context, tokenHash, reason string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE participant_sessions SET revoked_at=COALESCE(revoked_at, now()), revoke_reason=$2
		WHERE token_hash=$1`, tokenHash, reason)
	return err
}
