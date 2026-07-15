package useradmin

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrLoginTaken = errors.New("login already taken")

type Repo struct {
	pool *pgxpool.Pool
}

func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

// List возвращает страницу пользователей и общее число (для пагинации, SITE.md §43).
// Фильтры: поиск по login/full_name, роль, статус. Роли догружаются одним запросом.
func (r *Repo) List(ctx context.Context, f ListFilter) ([]User, int, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT u.id, u.login, u.full_name, u.email, u.organization, u.org_name, u.status,
		       u.must_change_password, u.last_login_at, u.created_at, count(*) OVER()
		FROM users u
		WHERE u.deleted_at IS NULL
		  AND ($1='' OR u.login ILIKE '%'||$1||'%' OR u.full_name ILIKE '%'||$1||'%')
		  AND ($2='' OR u.status=$2)
		  AND ($3='' OR EXISTS (SELECT 1 FROM user_roles ur JOIN roles rl ON rl.id=ur.role_id
		                        WHERE ur.user_id=u.id AND rl.code=$3))
		  AND ($6::bool OR u.created_by = $7)
		ORDER BY u.created_at DESC
		LIMIT $4 OFFSET $5`, f.Search, f.Status, f.Role, f.Limit, f.Offset, f.AllOwners, f.OwnerID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var users []User
	var ids []string
	total := 0
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Login, &u.FullName, &u.Email, &u.Organization, &u.OrgName,
			&u.Status, &u.MustChange, &u.LastLoginAt, &u.CreatedAt, &total); err != nil {
			return nil, 0, err
		}
		u.Roles = []RoleAssignment{}
		users = append(users, u)
		ids = append(ids, u.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if err := r.attachRoles(ctx, users, ids); err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

// attachRoles догружает роли для набора пользователей (без N+1).
func (r *Repo) attachRoles(ctx context.Context, users []User, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	idx := make(map[string]int, len(users))
	for i := range users {
		idx[users[i].ID] = i
	}
	rows, err := r.pool.Query(ctx, `
		SELECT ur.user_id, rl.code, ur.scope_type, ur.scope_id, ur.access_level
		FROM user_roles ur JOIN roles rl ON rl.id=ur.role_id
		WHERE ur.user_id = ANY($1)`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var uid string
		var ra RoleAssignment
		if err := rows.Scan(&uid, &ra.Code, &ra.ScopeType, &ra.ScopeID, &ra.AccessLevel); err != nil {
			return err
		}
		if i, ok := idx[uid]; ok {
			users[i].Roles = append(users[i].Roles, ra)
		}
	}
	return rows.Err()
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// ByID возвращает пользователя с ролями (nil-ошибка ErrUserNotFound, если нет).
func (r *Repo) ByID(ctx context.Context, id string) (*User, error) {
	var u User
	err := r.pool.QueryRow(ctx, `
		SELECT id, login, full_name, email, organization, org_name, status,
		       must_change_password, last_login_at, created_at
		FROM users WHERE id=$1 AND deleted_at IS NULL`, id).
		Scan(&u.ID, &u.Login, &u.FullName, &u.Email, &u.Organization, &u.OrgName, &u.Status,
			&u.MustChange, &u.LastLoginAt, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	roles, err := r.rolesOf(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	u.Roles = roles
	return &u, nil
}

func (r *Repo) rolesOf(ctx context.Context, userID string) ([]RoleAssignment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT rl.code, ur.scope_type, ur.scope_id, ur.access_level
		FROM user_roles ur JOIN roles rl ON rl.id=ur.role_id
		WHERE ur.user_id=$1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []RoleAssignment{}
	for rows.Next() {
		var ra RoleAssignment
		if err := rows.Scan(&ra.Code, &ra.ScopeType, &ra.ScopeID, &ra.AccessLevel); err != nil {
			return nil, err
		}
		out = append(out, ra)
	}
	return out, rows.Err()
}
