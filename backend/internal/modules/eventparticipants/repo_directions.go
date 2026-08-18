package eventparticipants

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/eazytech/student-leader-cabinet/internal/modules/eventpermissions"
)

func (r *Repo) CanAccessDirections(ctx context.Context, userID, contestID string) (bool, error) {
	var allowed bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM contests c
			WHERE c.id=$2 AND (
				c.owner_user_id=$1 OR EXISTS (
					SELECT 1 FROM event_staff_permissions ep
					WHERE ep.user_id=$1 AND ep.contest_id=c.id
					  AND ep.permission=ANY($3::varchar[]))))`,
		userID, contestID, []string{
			eventpermissions.ParticipantsManage,
			eventpermissions.AttendanceScan,
			eventpermissions.AttendanceManage,
		}).Scan(&allowed)
	return allowed, err
}

func (r *Repo) ListDirections(ctx context.Context, contestID string) ([]Direction, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, contest_id, name, sort_order, created_at, updated_at
		FROM event_directions WHERE contest_id=$1
		ORDER BY sort_order, name, id`, contestID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]Direction, 0)
	for rows.Next() {
		direction, err := scanDirection(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *direction)
	}
	return list, rows.Err()
}

func (r *Repo) CreateDirection(ctx context.Context, contestID, name string) (*Direction, error) {
	normalized := strings.ToLower(name)
	direction, err := scanDirection(r.pool.QueryRow(ctx, `
		INSERT INTO event_directions (contest_id, name, name_normalized, sort_order)
		VALUES ($1, $2, $3, (
			SELECT COALESCE(MAX(sort_order), 0) + 1 FROM event_directions WHERE contest_id=$1
		))
		RETURNING id, contest_id, name, sort_order, created_at, updated_at`,
		contestID, name, normalized))
	if isUniqueViolation(err) {
		return nil, ErrDirectionTaken
	}
	return direction, err
}

func (r *Repo) UpdateDirection(ctx context.Context, contestID, directionID, name string) (*Direction, error) {
	normalized := strings.ToLower(name)
	direction, err := scanDirection(r.pool.QueryRow(ctx, `
		UPDATE event_directions
		SET name=$3, name_normalized=$4, updated_at=now()
		WHERE contest_id=$1 AND id=$2
		RETURNING id, contest_id, name, sort_order, created_at, updated_at`,
		contestID, directionID, name, normalized))
	if errors.Is(err, ErrNotFound) {
		return nil, ErrDirectionNotFound
	}
	if isUniqueViolation(err) {
		return nil, ErrDirectionTaken
	}
	return direction, err
}

func (r *Repo) DeleteDirection(ctx context.Context, contestID, directionID string) error {
	ct, err := r.pool.Exec(ctx, `
		DELETE FROM event_directions WHERE contest_id=$1 AND id=$2`, contestID, directionID)
	if isForeignKeyViolation(err) {
		return ErrDirectionInUse
	}
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrDirectionNotFound
	}
	return nil
}

func (r *Repo) EnsureDirection(ctx context.Context, contestID, name string) (*Direction, error) {
	normalized := strings.ToLower(name)
	direction, err := scanDirection(r.pool.QueryRow(ctx, `
		INSERT INTO event_directions (contest_id, name, name_normalized, sort_order)
		VALUES ($1, $2, $3, (
			SELECT COALESCE(MAX(sort_order), 0) + 1 FROM event_directions WHERE contest_id=$1
		))
		ON CONFLICT (contest_id, name_normalized)
		DO UPDATE SET name=EXCLUDED.name, updated_at=now()
		RETURNING id, contest_id, name, sort_order, created_at, updated_at`,
		contestID, name, normalized))
	if isUniqueViolation(err) {
		return nil, ErrDirectionTaken
	}
	return direction, err
}

func (r *Repo) DirectionInContest(ctx context.Context, contestID, directionID string) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM event_directions WHERE contest_id=$1 AND id=$2)`,
		contestID, directionID).Scan(&ok)
	return ok, err
}

func scanDirection(row rowScanner) (*Direction, error) {
	var direction Direction
	err := row.Scan(&direction.ID, &direction.ContestID, &direction.Name,
		&direction.SortOrder, &direction.CreatedAt, &direction.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &direction, nil
}

func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}
