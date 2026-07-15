package challenges

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

// Create вставляет испытание в статусе DRAFT в конец списка конкурса, возвращает id.
func (r *Repo) Create(ctx context.Context, c *Challenge, actorID string) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx, `
		INSERT INTO contest_challenges
		  (contest_id, title, slug, short_description, full_description, instructions,
		   status, sort_order, open_at, deadline_at, close_at, created_by, updated_by)
		VALUES ($1,$2,$3,$4,$5,$6,'DRAFT',
		  (SELECT coalesce(max(sort_order),0)+1 FROM contest_challenges WHERE contest_id=$1),
		  $7,$8,$9,$10,$10)
		RETURNING id`,
		c.ContestID, c.Title, c.Slug, c.ShortDescription, c.FullDescription,
		c.Instructions, c.OpenAt, c.DeadlineAt, c.CloseAt, actorID).Scan(&id)
	if isUniqueViolation(err) {
		return "", ErrSlugTaken
	}
	return id, err
}

// Update меняет редактируемую мету испытания (без slug и статуса).
func (r *Repo) Update(ctx context.Context, id string, c *Challenge, actorID string) error {
	ct, err := r.pool.Exec(ctx, `
		UPDATE contest_challenges SET title=$2, short_description=$3, full_description=$4,
		       instructions=$5, open_at=$6, deadline_at=$7, close_at=$8,
		       updated_by=$9, updated_at=now()
		WHERE id=$1`,
		id, c.Title, c.ShortDescription, c.FullDescription, c.Instructions,
		c.OpenAt, c.DeadlineAt, c.CloseAt, actorID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SetStatus переводит испытание в новый статус, проставляя published_at/archived_at.
// published/archived передаём отдельными bool-параметрами: иначе $2 используется
// и в присваивании (varchar), и в сравнении (text) — pgx не выводит тип при describe.
func (r *Repo) SetStatus(ctx context.Context, id, status, actorID string) error {
	ct, err := r.pool.Exec(ctx, `
		UPDATE contest_challenges SET status=$2, updated_by=$3, updated_at=now(),
		       published_at = CASE WHEN $4 AND published_at IS NULL
		                           THEN now() ELSE published_at END,
		       archived_at  = CASE WHEN $5 THEN now() ELSE archived_at END
		WHERE id=$1`, id, status, actorID, status == StatusPublished, status == StatusArchived)
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

// cyrTranslit — таблица транслитерации кириллицы для слагов (заголовки на русском).
var cyrTranslit = map[rune]string{
	'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d", 'е': "e", 'ё': "e",
	'ж': "zh", 'з': "z", 'и': "i", 'й': "y", 'к': "k", 'л': "l", 'м': "m",
	'н': "n", 'о': "o", 'п': "p", 'р': "r", 'с': "s", 'т': "t", 'у': "u",
	'ф': "f", 'х': "h", 'ц': "ts", 'ч': "ch", 'ш': "sh", 'щ': "sch",
	'ъ': "", 'ы': "y", 'ь': "", 'э': "e", 'ю': "yu", 'я': "ya",
}

// slugify — ASCII-слаг из заголовка с транслитерацией кириллицы
// (fallback, если слаг не задан явно).
func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	prevDash := false
	writeDash := func() {
		if !prevDash && b.Len() > 0 {
			b.WriteRune('-')
			prevDash = true
		}
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		case cyrTranslit[r] != "":
			b.WriteString(cyrTranslit[r])
			prevDash = false
		default:
			writeDash()
		}
	}
	return strings.Trim(b.String(), "-")
}