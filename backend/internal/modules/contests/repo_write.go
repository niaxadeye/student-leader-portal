package contests

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// SetImageKey обновляет ключ обложки (nil очищает). Возвращает прежний ключ,
// чтобы вызывающий мог удалить старый объект из хранилища.
func (r *Repo) SetImageKey(ctx context.Context, id string, key *string, actorID string) (*string, error) {
	var prev *string
	// old CTE фиксирует прежний ключ до UPDATE, чтобы вернуть его для очистки.
	err := r.pool.QueryRow(ctx, `
		WITH old AS (SELECT image_key FROM contests WHERE id=$1)
		UPDATE contests SET image_key=$2, updated_by=$3, updated_at=now()
		WHERE id=$1
		RETURNING (SELECT image_key FROM old)`, id, key, actorID).Scan(&prev)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return prev, err
}

// Create вставляет конкурс в статусе DRAFT, возвращает id.
func (r *Repo) Create(ctx context.Context, c *Contest, actorID string) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx, `
		INSERT INTO contests (name, slug, description, status, start_at, end_at, timezone, created_by, updated_by)
		VALUES ($1,$2,$3,'DRAFT',$4,$5,$6,$7,$7)
		RETURNING id`,
		c.Name, c.Slug, c.Description, c.StartAt, c.EndAt, c.Timezone, actorID).Scan(&id)
	if isUniqueViolation(err) {
		return "", ErrSlugTaken
	}
	return id, err
}

// Update меняет редактируемые поля конкурса.
func (r *Repo) Update(ctx context.Context, id string, c *Contest, actorID string) error {
	ct, err := r.pool.Exec(ctx, `
		UPDATE contests SET name=$2, description=$3, start_at=$4, end_at=$5,
		       timezone=$6, updated_by=$7, updated_at=now()
		WHERE id=$1`, id, c.Name, c.Description, c.StartAt, c.EndAt, c.Timezone, actorID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SetStatus переводит конкурс в новый статус; при ARCHIVED ставит archived_at.
func (r *Repo) SetStatus(ctx context.Context, id, status, actorID string) error {
	// archived передаём отдельным bool-параметром: $2 иначе используется и в
	// присваивании, и в сравнении — pgx не может однозначно вывести тип при describe.
	ct, err := r.pool.Exec(ctx, `
		UPDATE contests SET status=$2, updated_by=$3, updated_at=now(),
		       archived_at = CASE WHEN $4 THEN now() ELSE archived_at END
		WHERE id=$1`, id, status, actorID, status == StatusArchived)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// slugify — упрощённый ASCII/транслит слаг из имени (fallback, если не задан явно).
func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}
