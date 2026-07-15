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

// Access — уровень доступа к конкурсу (docs/RBAC_MULTITENANCY.md §1.2, §3.2).
type Access int

const (
	AccessNone Access = iota // нет доступа
	AccessView               // только чтение
	AccessEdit               // чтение + редактирование
)

// CanView сообщает, разрешено ли хотя бы чтение.
func (a Access) CanView() bool { return a >= AccessView }

// CanEdit сообщает, разрешено ли редактирование.
func (a Access) CanEdit() bool { return a >= AccessEdit }

// AccessLevel вычисляет уровень доступа пользователя к конкурсу (§3.2):
//   - MEGA_ADMIN            → EDIT (полный кросс-арендный доступ, O5);
//   - владелец конкурса     → EDIT;
//   - назначенный ADMIN     → EDIT|VIEW из user_roles.access_level;
//   - иначе                 → None.
func (r *Repo) AccessLevel(ctx context.Context, userID, contestID string, isMega bool) (Access, error) {
	if isMega {
		return AccessEdit, nil
	}
	// owner_user_id даёт владельцу неявный EDIT; для назначенного ADMIN берём access_level.
	// LEFT JOIN, чтобы одним запросом покрыть оба случая и отсутствие доступа.
	var owner bool
	var level *string
	err := r.pool.QueryRow(ctx, `
		SELECT (c.owner_user_id = $1) AS owner,
		       (SELECT ur.access_level FROM user_roles ur JOIN roles rl ON rl.id = ur.role_id
		         WHERE ur.user_id = $1 AND rl.code = 'ADMIN'
		           AND ur.scope_type = 'CONTEST' AND ur.scope_id = $2
		         LIMIT 1) AS level
		FROM contests c WHERE c.id = $2`, userID, contestID).Scan(&owner, &level)
	if errors.Is(err, pgx.ErrNoRows) {
		return AccessNone, ErrNotFound
	}
	if err != nil {
		return AccessNone, err
	}
	if owner {
		return AccessEdit, nil
	}
	switch {
	case level == nil:
		return AccessNone, nil
	case *level == "EDIT":
		return AccessEdit, nil
	default:
		// VIEW или неизвестное значение трактуем как только чтение (безопасный минимум).
		return AccessView, nil
	}
}

// ListForPrincipal — конкурсы в области видимости актора (§3.5):
//   - MEGA_ADMIN — все конкурсы;
//   - иначе — где пользователь владелец (owner_user_id) ИЛИ назначен ADMIN (scoped).
// SUPER_ADMIN попадает сюда как владелец своих конкурсов.
func (r *Repo) ListForPrincipal(ctx context.Context, userID string, isMega bool, status string) ([]Contest, error) {
	// access_level в выборке: мега/владелец → OWNER; иначе уровень из user_roles (EDIT|VIEW).
	rows, err := r.pool.Query(ctx, `
		SELECT c.id, c.name, c.slug, c.description, c.status, c.start_at, c.end_at,
		       c.timezone, c.image_key, c.created_at, c.updated_at, c.archived_at,
		       (SELECT count(*) FROM contest_participants p
		          WHERE p.contest_id = c.id AND p.left_at IS NULL),
		       (SELECT count(*) FROM contest_challenges ch
		          WHERE ch.contest_id = c.id AND ch.status <> 'ARCHIVED'),
		       CASE WHEN $1::bool OR c.owner_user_id = $2 THEN 'OWNER'
		            ELSE (SELECT ur.access_level FROM user_roles ur JOIN roles rl ON rl.id = ur.role_id
		                   WHERE ur.user_id = $2 AND rl.code = 'ADMIN'
		                     AND ur.scope_type = 'CONTEST' AND ur.scope_id = c.id
		                   LIMIT 1)
		       END AS access_level
		FROM contests c
		WHERE ($1::bool
		        OR c.owner_user_id = $2
		        OR c.id IN (
		          SELECT ur.scope_id FROM user_roles ur JOIN roles rl ON rl.id = ur.role_id
		          WHERE ur.user_id = $2 AND rl.code = 'ADMIN' AND ur.scope_type = 'CONTEST'))
		  AND ($3 = '' OR c.status = $3)
		ORDER BY c.created_at DESC`, isMega, userID, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanContests(rows)
}

// ContestViewable — есть ли доступ хотя бы на чтение (EDIT|VIEW). Обёртка над AccessLevel
// для внешних модулей (challenges/submissions), чтобы не тянуть тип Access через границу пакета.
func (r *Repo) ContestViewable(ctx context.Context, userID, contestID string, isMega bool) (bool, error) {
	lvl, err := r.AccessLevel(ctx, userID, contestID, isMega)
	if err != nil {
		return false, err
	}
	return lvl.CanView(), nil
}

// ContestEditable — есть ли доступ на редактирование (владелец, EDIT-админ или мега).
func (r *Repo) ContestEditable(ctx context.Context, userID, contestID string, isMega bool) (bool, error) {
	lvl, err := r.AccessLevel(ctx, userID, contestID, isMega)
	if err != nil {
		return false, err
	}
	return lvl.CanEdit(), nil
}

// IsOwner сообщает, владеет ли пользователь конкурсом (owner_user_id). ErrNotFound, если конкурса нет.
func (r *Repo) IsOwner(ctx context.Context, userID, contestID string) (bool, error) {
	var owner bool
	err := r.pool.QueryRow(ctx,
		`SELECT c.owner_user_id = $1 FROM contests c WHERE c.id = $2`, userID, contestID).Scan(&owner)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, ErrNotFound
	}
	return owner, err
}

// ListForParticipant — конкурсы, где пользователь активный участник (для кабинета конкурсанта).
func (r *Repo) ListForParticipant(ctx context.Context, userID string) ([]Contest, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT c.id, c.name, c.slug, c.description, c.status, c.start_at, c.end_at,
		       c.timezone, c.image_key, c.created_at, c.updated_at, c.archived_at,
		       (SELECT count(*) FROM contest_participants p
		          WHERE p.contest_id = c.id AND p.left_at IS NULL),
		       (SELECT count(*) FROM contest_challenges ch
		          WHERE ch.contest_id = c.id AND ch.status <> 'ARCHIVED'),
		       NULL::text AS access_level
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

// scanContests читает строки с 15-й колонкой access_level (nullable).
func scanContests(rows pgx.Rows) ([]Contest, error) {
	contests := make([]Contest, 0)
	for rows.Next() {
		var c Contest
		var level *string
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.Description, &c.Status,
			&c.StartAt, &c.EndAt, &c.Timezone, &c.ImageKey, &c.CreatedAt, &c.UpdatedAt,
			&c.ArchivedAt, &c.ParticipantsCount, &c.ChallengesCount, &level); err != nil {
			return nil, err
		}
		if level != nil {
			c.AccessLevel = *level
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
