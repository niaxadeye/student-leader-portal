package useradmin

import "time"

// Валидные роли и scope для назначения (SITE.md §5, §21.2–21.3).
var ValidRoles = map[string]bool{"SUPER_ADMIN": true, "ADMIN": true, "CONTESTANT": true}

const (
	ScopeGlobal  = "GLOBAL"
	ScopeContest = "CONTEST"
	nilUUID      = "00000000-0000-0000-0000-000000000000"
)

// RoleAssignment — роль пользователя со scope.
type RoleAssignment struct {
	Code      string `json:"code"`
	ScopeType string `json:"scope_type"`
	ScopeID   string `json:"scope_id"`
}

// User — запись реестра пользователей.
type User struct {
	ID           string           `json:"id"`
	Login        string           `json:"login"`
	FullName     string           `json:"full_name"`
	Email        *string          `json:"email"`
	Organization *string          `json:"organization"`
	Status       string           `json:"status"`
	MustChange   bool             `json:"must_change_password"`
	LastLoginAt  *time.Time       `json:"last_login_at"`
	CreatedAt    time.Time        `json:"created_at"`
	Roles        []RoleAssignment `json:"roles"`
}

// ListFilter — параметры серверной пагинации/поиска/фильтров (SITE.md §43).
type ListFilter struct {
	Search string
	Role   string
	Status string
	Limit  int
	Offset int
}
