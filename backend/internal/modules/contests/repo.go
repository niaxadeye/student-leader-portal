package contests

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo struct {
	pool *pgxpool.Pool
}

func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

// HasContestAccess: SUPER_ADMIN имеет доступ ко всем; ADMIN — только к scoped-конкурсу.
func (r *Repo) HasContestAccess(ctx context.Context, userID, contestID string, isSuper bool) (bool, error) {
	if isSuper {
		return true, nil
	}
	var ok bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM user_roles ur JOIN roles rl ON rl.id = ur.role_id
			WHERE ur.user_id = $1 AND rl.code = 'ADMIN'
			  AND ur.scope_type = 'CONTEST' AND ur.scope_id = $2)`,
		userID, contestID).Scan(&ok)
	return ok, err
}

// ListForPrincipal: SUPER_ADMIN видит все конкурсы, ADMIN — только назначенные (scoped).
func (r *Repo) ListForPrincipal(ctx context.Context, userID string, isSuper bool, status string) ([]Contest, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT c.id, c.name, c.slug, c.description, c.status, c.start_at, c.end_at,
		       c.timezone, c.image_key, c.created_at, c.updated_at, c.archived_at,
		       (SELECT count(*) FROM contest_participants p
		          WHERE p.contest_id = c.id AND p.left_at IS NULL),
		       (SELECT count(*) FROM contest_challenges ch
		          WHERE ch.contest_id = c.id AND ch.status <> 'ARCHIVED')
		FROM contests c
		WHERE ($1::bool OR c.id IN (
		        SELECT ur.scope_id FROM user_roles ur JOIN roles rl ON rl.id = ur.role_id
		        WHERE ur.user_id = $2 AND rl.code = 'ADMIN' AND ur.scope_type = 'CONTEST'))
		  AND ($3 = '' OR c.status = $3)
		ORDER BY c.created_at DESC`, isSuper, userID, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanContests(rows)
}

// ListForParticipant — конкурсы, где пользователь активный участник (для кабинета конкурсанта).
func (r *Repo) ListForParticipant(ctx context.Context, userID string) ([]Contest, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT c.id, c.name, c.slug, c.description, c.status, c.start_at, c.end_at,
		       c.timezone, c.image_key, c.created_at, c.updated_at, c.archived_at,
		       (SELECT count(*) FROM contest_participants p
		          WHERE p.contest_id = c.id AND p.left_at IS NULL),
		       (SELECT count(*) FROM contest_challenges ch
		          WHERE ch.contest_id = c.id AND ch.status <> 'ARCHIVED')
		FROM contests c
		JOIN contest_participants cp ON cp.contest_id = c.id
		WHERE cp.user_id = $1 AND cp.left_at IS NULL
		  AND c.status IN ('ACTIVE','FINISHED')
		ORDER BY c.created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanContests(rows)
}

func scanContests(rows pgx.Rows) ([]Contest, error) {
	contests := make([]Contest, 0)
	for rows.Next() {
		var c Contest
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.Description, &c.Status,
			&c.StartAt, &c.EndAt, &c.Timezone, &c.ImageKey, &c.CreatedAt, &c.UpdatedAt,
			&c.ArchivedAt, &c.ParticipantsCount, &c.ChallengesCount); err != nil {
			return nil, err
		}
		contests = append(contests, c)
	}
	return contests, rows.Err()
}

// ByID возвращает конкурс с числом участников.
func (r *Repo) ByID(ctx context.Context, id string) (*Contest, error) {
	var c Contest
	err := r.pool.QueryRow(ctx, `
		SELECT c.id, c.name, c.slug, c.description, c.status, c.start_at, c.end_at,
		       c.timezone, c.image_key, c.created_at, c.updated_at, c.archived_at,
		       (SELECT count(*) FROM contest_participants p
		          WHERE p.contest_id = c.id AND p.left_at IS NULL),
		       (SELECT count(*) FROM contest_challenges ch
		          WHERE ch.contest_id = c.id AND ch.status <> 'ARCHIVED')
		FROM contests c WHERE c.id = $1`, id).
		Scan(&c.ID, &c.Name, &c.Slug, &c.Description, &c.Status, &c.StartAt,
			&c.EndAt, &c.Timezone, &c.ImageKey, &c.CreatedAt, &c.UpdatedAt, &c.ArchivedAt,
			&c.ParticipantsCount, &c.ChallengesCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}
