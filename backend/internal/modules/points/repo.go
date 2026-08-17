package points

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/eazytech/student-leader-cabinet/internal/modules/audit"
	"github.com/eazytech/student-leader-cabinet/internal/modules/eventpermissions"
)

type txAuditor interface {
	LogEntryTx(ctx context.Context, tx pgx.Tx, entry audit.Entry) error
}

type Repo struct {
	pool  *pgxpool.Pool
	audit txAuditor
}

func NewRepo(pool *pgxpool.Pool, auditor txAuditor) *Repo {
	return &Repo{pool: pool, audit: auditor}
}

func (r *Repo) CanManage(ctx context.Context, userID, contestID string) (bool, error) {
	var allowed bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM contests c
			WHERE c.id=$2 AND (
				c.owner_user_id=$1 OR EXISTS (
					SELECT 1 FROM event_staff_permissions ep
					WHERE ep.user_id=$1 AND ep.contest_id=c.id
					  AND ep.permission=ANY($3::varchar[]))))`,
		userID, contestID, eventpermissions.GrantsFor(PermissionManagePoints)).Scan(&allowed)
	return allowed, err
}

func (r *Repo) ParticipantExists(ctx context.Context, contestID, participantID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM event_participants WHERE contest_id=$1 AND id=$2
		)`, contestID, participantID).Scan(&exists)
	return exists, err
}

func (r *Repo) LedgerBalance(ctx context.Context, contestID, participantID string) (int64, error) {
	var balance int64
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0)::bigint
		FROM points_ledger WHERE contest_id=$1 AND event_participant_id=$2`,
		contestID, participantID).Scan(&balance)
	return balance, err
}

func (r *Repo) ActiveHolds(ctx context.Context, contestID, participantID string) (int64, error) {
	var reserved int64
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0)::bigint FROM points_holds
		WHERE contest_id=$1 AND event_participant_id=$2 AND status='ACTIVE'`,
		contestID, participantID).Scan(&reserved)
	return reserved, err
}

func (r *Repo) List(
	ctx context.Context,
	contestID, participantID string,
	limit, offset int,
) ([]Entry, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `
		SELECT count(*) FROM points_ledger
		WHERE contest_id=$1 AND event_participant_id=$2`, contestID, participantID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, contest_id, event_participant_id, amount, type, source_type,
		       source_id, description, created_by_user_id, idempotency_key, created_at
		FROM points_ledger
		WHERE contest_id=$1 AND event_participant_id=$2
		ORDER BY created_at DESC, id DESC LIMIT $3 OFFSET $4`,
		contestID, participantID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	entries := make([]Entry, 0)
	for rows.Next() {
		entry, err := scanEntry(rows)
		if err != nil {
			return nil, 0, err
		}
		entries = append(entries, *entry)
	}
	return entries, total, rows.Err()
}

func (r *Repo) Append(ctx context.Context, input AppendInput) (*Entry, bool, error) {
	return appendEntry(ctx, r.pool, input)
}

// AppendTx используется доменными транзакциями attendance/tasks/merch, чтобы
// запись источника и начисление/списание фиксировались атомарно.
func (r *Repo) AppendTx(ctx context.Context, tx pgx.Tx, input AppendInput) (*Entry, bool, error) {
	return appendEntry(ctx, tx, input)
}

func (r *Repo) AppendAdminAdjustment(
	ctx context.Context,
	actorUserID string,
	input AppendInput,
) (entry *Entry, created bool, err error) {
	err = pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		entry, created, err = appendEntry(ctx, tx, input)
		if err != nil || !created {
			return err
		}
		return r.audit.LogEntryTx(ctx, tx, audit.Entry{
			ActorUserID: actorUserID,
			ContestID:   input.ContestID,
			Action:      "EVENT_POINTS_ADMIN_ADJUSTED",
			EntityType:  "points_ledger",
			EntityID:    entry.ID,
			Metadata: map[string]any{
				"participant_id":  input.EventParticipantID,
				"amount":          input.Amount,
				"reason":          input.Description,
				"idempotency_key": input.IdempotencyKey,
			},
		})
	})
	return entry, created, err
}

type queryRower interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func appendEntry(ctx context.Context, query queryRower, input AppendInput) (*Entry, bool, error) {
	entry, err := scanEntry(query.QueryRow(ctx, `
		INSERT INTO points_ledger
		  (contest_id, event_participant_id, amount, type, source_type, source_id,
		   description, created_by_user_id, idempotency_key)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (contest_id, idempotency_key) DO NOTHING
		RETURNING id, contest_id, event_participant_id, amount, type, source_type,
		          source_id, description, created_by_user_id, idempotency_key, created_at`,
		input.ContestID, input.EventParticipantID, input.Amount, input.Type,
		input.SourceType, input.SourceID, input.Description, input.CreatedByUserID,
		input.IdempotencyKey))
	if err == nil {
		return entry, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, false, err
	}

	existing, err := scanEntry(query.QueryRow(ctx, `
		SELECT id, contest_id, event_participant_id, amount, type, source_type,
		       source_id, description, created_by_user_id, idempotency_key, created_at
		FROM points_ledger WHERE contest_id=$1 AND idempotency_key=$2`,
		input.ContestID, input.IdempotencyKey))
	if err != nil {
		return nil, false, err
	}
	if !sameOperation(existing, input) {
		return nil, false, ErrIdempotencyConflict
	}
	return existing, false, nil
}

func sameOperation(entry *Entry, input AppendInput) bool {
	return entry.ContestID == input.ContestID &&
		entry.EventParticipantID == input.EventParticipantID &&
		entry.Amount == input.Amount &&
		entry.Type == input.Type &&
		optionalEqual(entry.SourceType, input.SourceType) &&
		optionalEqual(entry.SourceID, input.SourceID) &&
		entry.Description == input.Description &&
		optionalEqual(entry.CreatedByUserID, input.CreatedByUserID)
}

func optionalEqual(left, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanEntry(row rowScanner) (*Entry, error) {
	var entry Entry
	err := row.Scan(&entry.ID, &entry.ContestID, &entry.EventParticipantID, &entry.Amount,
		&entry.Type, &entry.SourceType, &entry.SourceID, &entry.Description,
		&entry.CreatedByUserID, &entry.IdempotencyKey, &entry.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &entry, nil
}
