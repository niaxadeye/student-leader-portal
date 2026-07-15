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

// List отдаёт страницу реестра с нормализованными лимитами.
func (s *Service) List(ctx context.Context, f ListFilter) (*ListResult, error) {
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
	users, total, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, err
	}
	if users == nil {
		users = []User{}
	}
	return &ListResult{Users: users, Total: total, Limit: f.Limit, Offset: f.Offset}, nil
}

// Get возвращает пользователя с ролями.
func (s *Service) Get(ctx context.Context, id string) (*User, error) {
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
}

// CreateResult — созданный пользователь и его временный пароль (показать один раз).
type CreateResult struct {
	UserID       string
	Login        string
	TempPassword string
}

// Create создаёт пользователя с временным паролем и опциональной ролью.
func (s *Service) Create(ctx context.Context, actorID string, in CreateInput) (*CreateResult, error) {
	login := strings.TrimSpace(in.Login)
	name := strings.TrimSpace(in.FullName)
	if login == "" || name == "" {
		return nil, ErrValidation
	}
	role, scopeType, scopeID := "", "", ""
	if strings.TrimSpace(in.Role) != "" {
		norm, ok := normScope(AssignRoleInput{Role: in.Role, ScopeType: in.ScopeType, ScopeID: in.ScopeID})
		if !ok {
			return nil, ErrValidation
		}
		role, scopeType, scopeID = norm.Role, norm.ScopeType, norm.ScopeID
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
	}, role, scopeType, scopeID)
	if err != nil {
		return nil, err
	}
	s.audit.Log(ctx, actorID, "USER_CREATED", "user", id, map[string]any{"login": login, "role": role})
	return &CreateResult{UserID: id, Login: login, TempPassword: temp}, nil
}

// Update меняет профиль пользователя.
func (s *Service) Update(ctx context.Context, actorID, id, fullName, email, org string) (*User, error) {
	name := strings.TrimSpace(fullName)
	if name == "" {
		return nil, ErrValidation
	}
	if err := s.repo.UpdateProfile(ctx, id, name, optStr(email), optStr(org)); err != nil {
		return nil, err
	}
	s.audit.Log(ctx, actorID, "USER_UPDATED", "user", id, nil)
	return s.repo.ByID(ctx, id)
}

func optStr(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}
