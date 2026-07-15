package challenges

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

// IsParticipant проверяет активное участие пользователя в конкурсе (для чтения контестантом).
func (r *Repo) IsParticipant(ctx context.Context, userID, contestID string) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM contest_participants
		  WHERE user_id=$1 AND contest_id=$2 AND left_at IS NULL)`,
		userID, contestID).Scan(&ok)
	return ok, err
}

const challengeCols = `c.id, c.contest_id, c.title, c.slug, c.short_description,
	c.full_description, c.instructions, c.status, c.sort_order, c.open_at,
	c.deadline_at, c.close_at, c.settings, c.current_schema_version,
	c.created_at, c.updated_at, c.published_at, c.archived_at,
	(SELECT count(*) FROM challenge_fields f
	   WHERE f.challenge_id = c.id AND f.deleted_at IS NULL
	     AND f.schema_version_to IS NULL)`

func scanChallenge(row pgx.Row) (*Challenge, error) {
	var c Challenge
	err := row.Scan(&c.ID, &c.ContestID, &c.Title, &c.Slug, &c.ShortDescription,
		&c.FullDescription, &c.Instructions, &c.Status, &c.SortOrder, &c.OpenAt,
		&c.DeadlineAt, &c.CloseAt, &c.Settings, &c.CurrentSchemaVersion,
		&c.CreatedAt, &c.UpdatedAt, &c.PublishedAt, &c.ArchivedAt, &c.FieldsCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// ByID возвращает испытание с числом активных полей.
func (r *Repo) ByID(ctx context.Context, id string) (*Challenge, error) {
	return scanChallenge(r.pool.QueryRow(ctx,
		`SELECT `+challengeCols+` FROM contest_challenges c WHERE c.id=$1`, id))
}

// ContestIDOf возвращает конкурс испытания (для проверки доступа до полной загрузки).
func (r *Repo) ContestIDOf(ctx context.Context, challengeID string) (string, error) {
	var cid string
	err := r.pool.QueryRow(ctx,
		`SELECT contest_id FROM contest_challenges WHERE id=$1`, challengeID).Scan(&cid)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return cid, err
}

// List возвращает испытания конкурса по порядку сортировки.
// onlyPublished=true оставляет лишь видимые контестанту (PUBLISHED, не архив).
func (r *Repo) List(ctx context.Context, contestID string, onlyPublished bool) ([]Challenge, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+challengeCols+`
		FROM contest_challenges c
		WHERE c.contest_id=$1 AND ($2::bool = false OR c.status='PUBLISHED')
		ORDER BY c.sort_order, c.created_at`, contestID, onlyPublished)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]Challenge, 0)
	for rows.Next() {
		c, err := scanChallenge(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *c)
	}
	return list, rows.Err()
}

// ListForContestant — опубликованные испытания + статус работы конкурсанта (для кабинета).
func (r *Repo) ListForContestant(ctx context.Context, contestID, userID string) ([]Challenge, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+challengeCols+`, COALESCE(s.status, 'NOT_STARTED')
		FROM contest_challenges c
		LEFT JOIN submissions s ON s.challenge_id = c.id AND s.contestant_user_id = $2
		WHERE c.contest_id=$1 AND c.status='PUBLISHED'
		ORDER BY c.sort_order, c.created_at`, contestID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]Challenge, 0)
	for rows.Next() {
		var c Challenge
		if err := rows.Scan(&c.ID, &c.ContestID, &c.Title, &c.Slug, &c.ShortDescription,
			&c.FullDescription, &c.Instructions, &c.Status, &c.SortOrder, &c.OpenAt,
			&c.DeadlineAt, &c.CloseAt, &c.Settings, &c.CurrentSchemaVersion,
			&c.CreatedAt, &c.UpdatedAt, &c.PublishedAt, &c.ArchivedAt, &c.FieldsCount,
			&c.MySubmissionStatus); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

// Fields возвращает активные (не удалённые, текущей версии) поля испытания по порядку.
func (r *Repo) Fields(ctx context.Context, challengeID string) ([]Field, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, challenge_id, field_key, field_type, label, description, help_text,
		       placeholder, required, sort_order, settings, validation, visibility
		FROM challenge_fields
		WHERE challenge_id=$1 AND deleted_at IS NULL AND schema_version_to IS NULL
		ORDER BY sort_order, created_at`, challengeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]Field, 0)
	for rows.Next() {
		var f Field
		if err := rows.Scan(&f.ID, &f.ChallengeID, &f.Key, &f.Type, &f.Label,
			&f.Description, &f.HelpText, &f.Placeholder, &f.Required, &f.SortOrder,
			&f.Settings, &f.Validation, &f.Visibility); err != nil {
			return nil, err
		}
		list = append(list, f)
	}
	return list, rows.Err()
}
