package eventpermissions

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrForbidden  = errors.New("forbidden")
	ErrValidation = errors.New("validation error")
	ErrNotFound   = errors.New("not found")
)

// Grant is the set of STAFF permissions on one contest.
type Grant struct {
	ContestID   string   `json:"contest_id"`
	Permissions []string `json:"permissions"`
}

// Assignment is a STAFF user granted permissions on a contest.
type Assignment struct {
	UserID      string   `json:"user_id"`
	Login       string   `json:"login"`
	FullName    string   `json:"full_name"`
	Permissions []string `json:"permissions"`
}

type Actor struct {
	UserID string
	Role   string
}

func (a Actor) IsMega() bool  { return a.Role == "MEGA_ADMIN" }
func (a Actor) IsSuper() bool { return a.Role == "SUPER_ADMIN" }

type userRef struct {
	CreatedBy *string
	Roles     []string
}

type Repository interface {
	ListByUser(ctx context.Context, userID string) ([]Grant, error)
	ListByContest(ctx context.Context, contestID string) ([]Assignment, error)
	Replace(ctx context.Context, contestID, userID, grantedBy string, permissions []string) error
	Clear(ctx context.Context, contestID, userID string) error
	ContestOwner(ctx context.Context, contestID string) (string, error)
	UserRef(ctx context.Context, userID string) (*userRef, error)
}

type Auditor interface {
	Log(ctx context.Context, actorUserID, action, entityType, entityID string, meta map[string]any)
}

type Repo struct {
	pool *pgxpool.Pool
}

func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

func (r *Repo) ListByUser(ctx context.Context, userID string) ([]Grant, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT contest_id, permission
		FROM event_staff_permissions
		WHERE user_id=$1
		ORDER BY contest_id, permission`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectGrants(rows)
}

func (r *Repo) ListByContest(ctx context.Context, contestID string) ([]Assignment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT ep.user_id, u.login, u.full_name, ep.permission
		FROM event_staff_permissions ep
		JOIN users u ON u.id = ep.user_id
		WHERE ep.contest_id=$1 AND u.deleted_at IS NULL
		ORDER BY u.full_name, ep.permission`, contestID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	idx := map[string]int{}
	out := []Assignment{}
	for rows.Next() {
		var userID, login, name, perm string
		if err := rows.Scan(&userID, &login, &name, &perm); err != nil {
			return nil, err
		}
		i, ok := idx[userID]
		if !ok {
			out = append(out, Assignment{UserID: userID, Login: login, FullName: name, Permissions: []string{}})
			i = len(out) - 1
			idx[userID] = i
		}
		out[i].Permissions = append(out[i].Permissions, perm)
	}
	return out, rows.Err()
}

func (r *Repo) Replace(ctx context.Context, contestID, userID, grantedBy string, permissions []string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
		DELETE FROM event_staff_permissions WHERE user_id=$1 AND contest_id=$2`, userID, contestID); err != nil {
		return err
	}
	for _, perm := range permissions {
		if _, err := tx.Exec(ctx, `
			INSERT INTO event_staff_permissions (user_id, contest_id, permission, granted_by)
			VALUES ($1,$2,$3,$4)`, userID, contestID, perm, grantedBy); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *Repo) Clear(ctx context.Context, contestID, userID string) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM event_staff_permissions WHERE user_id=$1 AND contest_id=$2`, userID, contestID)
	return err
}

func (r *Repo) ContestOwner(ctx context.Context, contestID string) (string, error) {
	var owner *string
	err := r.pool.QueryRow(ctx, `SELECT owner_user_id FROM contests WHERE id=$1`, contestID).Scan(&owner)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if owner == nil || *owner == "" {
		return "", ErrNotFound
	}
	return *owner, nil
}

func (r *Repo) UserRef(ctx context.Context, userID string) (*userRef, error) {
	var createdBy *string
	err := r.pool.QueryRow(ctx, `
		SELECT created_by FROM users WHERE id=$1 AND deleted_at IS NULL`, userID).Scan(&createdBy)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT rl.code FROM user_roles ur JOIN roles rl ON rl.id=ur.role_id
		WHERE ur.user_id=$1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	roles := []string{}
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		roles = append(roles, code)
	}
	return &userRef{CreatedBy: createdBy, Roles: roles}, rows.Err()
}

func collectGrants(rows pgx.Rows) ([]Grant, error) {
	idx := map[string]int{}
	out := []Grant{}
	for rows.Next() {
		var contestID, perm string
		if err := rows.Scan(&contestID, &perm); err != nil {
			return nil, err
		}
		i, ok := idx[contestID]
		if !ok {
			out = append(out, Grant{ContestID: contestID, Permissions: []string{}})
			i = len(out) - 1
			idx[contestID] = i
		}
		out[i].Permissions = append(out[i].Permissions, perm)
	}
	return out, rows.Err()
}

type Service struct {
	repo  Repository
	audit Auditor
}

func NewService(repo Repository, audit Auditor) *Service {
	return &Service{repo: repo, audit: audit}
}

func normalizePermissions(in []string) ([]string, error) {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, raw := range in {
		perm := strings.TrimSpace(raw)
		if perm == "" {
			continue
		}
		if !IsKnown(perm) {
			return nil, ErrValidation
		}
		if seen[perm] {
			continue
		}
		seen[perm] = true
		out = append(out, perm)
	}
	return out, nil
}

func hasRole(roles []string, code string) bool {
	for _, r := range roles {
		if r == code {
			return true
		}
	}
	return false
}

func (s *Service) ensureCanGrant(ctx context.Context, actor Actor, contestID, targetID string) error {
	if targetID == "" || contestID == "" {
		return ErrValidation
	}
	target, err := s.repo.UserRef(ctx, targetID)
	if err != nil {
		return err
	}
	if hasRole(target.Roles, "MEGA_ADMIN") || hasRole(target.Roles, "SUPER_ADMIN") {
		return ErrForbidden
	}
	if !hasRole(target.Roles, "STAFF") {
		return ErrForbidden
	}
	if actor.IsMega() {
		return nil
	}
	if !actor.IsSuper() {
		return ErrForbidden
	}
	owner, err := s.repo.ContestOwner(ctx, contestID)
	if err != nil {
		return err
	}
	if owner != actor.UserID {
		return ErrForbidden
	}
	if target.CreatedBy == nil || *target.CreatedBy != actor.UserID {
		return ErrForbidden
	}
	return nil
}

func (s *Service) ensureCanViewContestStaff(ctx context.Context, actor Actor, contestID string) error {
	if actor.IsMega() {
		return nil
	}
	if !actor.IsSuper() {
		return ErrForbidden
	}
	owner, err := s.repo.ContestOwner(ctx, contestID)
	if err != nil {
		return err
	}
	if owner != actor.UserID {
		return ErrForbidden
	}
	return nil
}

func (s *Service) ensureCanViewUserStaff(ctx context.Context, actor Actor, userID string) error {
	if actor.IsMega() || actor.UserID == userID {
		return nil
	}
	if !actor.IsSuper() {
		return ErrForbidden
	}
	target, err := s.repo.UserRef(ctx, userID)
	if err != nil {
		return err
	}
	if target.CreatedBy == nil || *target.CreatedBy != actor.UserID {
		return ErrForbidden
	}
	return nil
}

func (s *Service) ListByUser(ctx context.Context, actor Actor, userID string) ([]Grant, error) {
	if err := s.ensureCanViewUserStaff(ctx, actor, userID); err != nil {
		return nil, err
	}
	list, err := s.repo.ListByUser(ctx, userID)
	if list == nil {
		list = []Grant{}
	}
	return list, err
}

func (s *Service) GrantsForUser(ctx context.Context, userID string) ([]Grant, error) {
	list, err := s.repo.ListByUser(ctx, userID)
	if list == nil {
		list = []Grant{}
	}
	return list, err
}

func (s *Service) ListByContest(ctx context.Context, actor Actor, contestID string) ([]Assignment, error) {
	if err := s.ensureCanViewContestStaff(ctx, actor, contestID); err != nil {
		return nil, err
	}
	list, err := s.repo.ListByContest(ctx, contestID)
	if list == nil {
		list = []Assignment{}
	}
	return list, err
}

func (s *Service) Replace(ctx context.Context, actor Actor, contestID, userID string, permissions []string) error {
	perms, err := normalizePermissions(permissions)
	if err != nil {
		return err
	}
	if err := s.ensureCanGrant(ctx, actor, contestID, userID); err != nil {
		return err
	}
	if len(perms) == 0 {
		if err := s.repo.Clear(ctx, contestID, userID); err != nil {
			return err
		}
		if s.audit != nil {
			s.audit.Log(ctx, actor.UserID, "STAFF_PERMISSIONS_CLEARED", "user", userID,
				map[string]any{"contest_id": contestID})
		}
		return nil
	}
	if err := s.repo.Replace(ctx, contestID, userID, actor.UserID, perms); err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Log(ctx, actor.UserID, "STAFF_PERMISSIONS_SET", "user", userID,
			map[string]any{"contest_id": contestID, "permissions": perms})
	}
	return nil
}
