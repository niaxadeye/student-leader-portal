package useradmin

import (
	"context"
	"strings"

	"github.com/eazytech/student-leader-cabinet/internal/platform/security"
)

const (
	defaultLimit = 20
	maxLimit     = 100
)

// ListResult — страница пользователей с метаданными пагинации.
type ListResult struct {
	Users  []User
	Total  int
	Limit  int
	Offset int
}

// List отдаёт страницу реестра с нормализованными лимитами. Реестр изолирован по
// владению (§3.3): мега видит всех, SUPER_ADMIN — только созданных им (created_by).
func (s *Service) List(ctx context.Context, a Actor, f ListFilter) (*ListResult, error) {
	if f.Limit <= 0 {
		f.Limit = defaultLimit
	}
	if f.Limit > maxLimit {
		f.Limit = maxLimit
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	f.Search = strings.TrimSpace(f.Search)
	f.Role = strings.ToUpper(strings.TrimSpace(f.Role))
	f.Status = strings.ToUpper(strings.TrimSpace(f.Status))
	f.AllOwners = a.IsMega()
	f.OwnerID = a.UserID
	users, total, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, err
	}
	if users == nil {
		users = []User{}
	}
	return &ListResult{Users: users, Total: total, Limit: f.Limit, Offset: f.Offset}, nil
}

// Get возвращает пользователя с ролями, только если актор вправе его видеть.
func (s *Service) Get(ctx context.Context, a Actor, id string) (*User, error) {
	if err := s.ensureManage(ctx, a, id, ActionView); err != nil {
		return nil, err
	}
	return s.repo.ByID(ctx, id)
}

// CreateInput — поля нового пользователя (+ опциональная стартовая роль).
type CreateInput struct {
	Login        string
	FullName     string
	Email        string
	Organization string
	Role         string
	ScopeType    string
	ScopeID      string
	AccessLevel  string // EDIT|VIEW для роли ADMIN на конкурс
	OrgName      string // организация-арендатор (§2.3); задаёт мега для SUPER_ADMIN, иначе наследуется
}

// CreateResult — созданный пользователь и его временный пароль (показать один раз).
type CreateResult struct {
	UserID       string
	Login        string
	TempPassword string
}

// canCreateRole проверяет, вправе ли актор создать пользователя с данной ролью (§3.3):
// MEGA_ADMIN — любую; SUPER_ADMIN — ADMIN/STAFF/JURY/CONTESTANT; прочие — никакую.
func canCreateRole(a Actor, role string) bool {
	switch {
	case a.IsMega():
		return true
	case a.IsSuper():
		return role == "" || role == "ADMIN" || role == "STAFF" || role == "JURY" || role == "REMOTE_JURY" || role == "CONTESTANT"
	default:
		return false
	}
}

// Create создаёт пользователя с временным паролем и опциональной ролью.
// Проставляет created_by = актор; для роли ADMIN наследует org_name создателя (§3.3).
func (s *Service) Create(ctx context.Context, a Actor, in CreateInput) (*CreateResult, error) {
	login := strings.TrimSpace(in.Login)
	name := strings.TrimSpace(in.FullName)
	if login == "" || name == "" {
		return nil, ErrValidation
	}
	role, scopeType, scopeID, accessLevel := "", "", "", ""
	if strings.TrimSpace(in.Role) != "" {
		norm, ok := normScope(AssignRoleInput{Role: in.Role, ScopeType: in.ScopeType, ScopeID: in.ScopeID, AccessLevel: in.AccessLevel})
		if !ok {
			return nil, ErrValidation
		}
		role, scopeType, scopeID, accessLevel = norm.Role, norm.ScopeType, norm.ScopeID, norm.AccessLevel
	}
	if !canCreateRole(a, role) {
		return nil, ErrForbidden
	}
	if role != "" {
		if err := s.ensureCanGrant(ctx, a, AssignRoleInput{
			Role: role, ScopeType: scopeType, ScopeID: scopeID, AccessLevel: accessLevel,
		}); err != nil {
			return nil, err
		}
	}
	temp, err := security.GenerateTempPassword()
	if err != nil {
		return nil, err
	}
	hash, err := security.HashPassword(temp)
	if err != nil {
		return nil, err
	}
	id, err := s.repo.Create(ctx, NewUser{
		Login: login, FullName: name, PasswordHash: hash,
		Email: optStr(in.Email), Organization: optStr(in.Organization),
		CreatedBy: a.UserID, OrgName: optStr(in.OrgName),
	}, role, scopeType, scopeID, accessLevel)
	if err != nil {
		return nil, err
	}
	s.log(ctx, a.UserID, "USER_CREATED", id, map[string]any{"login": login, "role": role})
	return &CreateResult{UserID: id, Login: login, TempPassword: temp}, nil
}

// Update меняет профиль пользователя.
func (s *Service) Update(ctx context.Context, a Actor, id, fullName, email, org string) (*User, error) {
	name := strings.TrimSpace(fullName)
	if name == "" {
		return nil, ErrValidation
	}
	if err := s.ensureManage(ctx, a, id, ActionUpdate); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateProfile(ctx, id, name, optStr(email), optStr(org)); err != nil {
		return nil, err
	}
	s.log(ctx, a.UserID, "USER_UPDATED", id, nil)
	return s.repo.ByID(ctx, id)
}

func optStr(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}
