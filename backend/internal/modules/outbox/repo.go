package outbox

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo struct {
	pool *pgxpool.Pool
}

func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

// ClaimPending атомарно берёт до limit готовых к обработке событий и помечает их
// заблокированными за worker'ом (FOR UPDATE SKIP LOCKED — SITE.md §15).
func (r *Repo) ClaimPending(ctx context.Context, worker string, limit int) ([]Event, error) {
	rows, err := r.pool.Query(ctx, `
		UPDATE outbox_events SET locked_at=now(), locked_by=$1
		WHERE id IN (
			SELECT id FROM outbox_events
			WHERE status='PENDING' AND available_at <= now()
			ORDER BY available_at
			FOR UPDATE SKIP LOCKED
			LIMIT $2)
		RETURNING id, event_type, aggregate_type, aggregate_id, payload, attempts`,
		worker, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Event, 0)
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.EventType, &e.AggregateType, &e.AggregateID,
			&e.Payload, &e.Attempts); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// MarkSent переводит событие в SENT.
func (r *Repo) MarkSent(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE outbox_events SET status='SENT', processed_at=now(),
		  locked_at=NULL, locked_by=NULL, last_error=NULL
		WHERE id=$1`, id)
	return err
}

// MarkFailed фиксирует ошибку: инкремент attempts, backoff в available_at,
// после maxAttempts — статус DEAD (SITE.md §15).
func (r *Repo) MarkFailed(ctx context.Context, id string, attempts, maxAttempts int, backoff time.Duration, errMsg string) error {
	dead := attempts >= maxAttempts
	_, err := r.pool.Exec(ctx, `
		UPDATE outbox_events SET
		  attempts=$2,
		  status = CASE WHEN $3 THEN 'DEAD' ELSE 'PENDING' END,
		  available_at = now() + $4::interval,
		  locked_at=NULL, locked_by=NULL,
		  last_error=$5
		WHERE id=$1`,
		id, attempts, dead, backoff.String(), truncate(errMsg, 1000))
	return err
}

// ReleaseStale возвращает в PENDING события, зависшие в блокировке дольше ttl
// (worker упал между claim и результатом).
func (r *Repo) ReleaseStale(ctx context.Context, ttl time.Duration) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE outbox_events SET locked_at=NULL, locked_by=NULL
		WHERE status='PENDING' AND locked_at IS NOT NULL AND locked_at < now() - $1::interval`,
		ttl.String())
	return err
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
