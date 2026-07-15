package useradmin

import "time"

// Валидные роли и scope для назначения (SITE.md §5, §21.2–21.3).
var ValidRoles = map[string]bool{"MEGA_ADMIN": true, "SUPER_ADMIN": true, "ADMIN": true, "CONTESTANT": true}

// Уровни доступа ADMIN к конкурсу (user_roles.access_level, §1.2).
const (
	AccessEdit = "EDIT"
	AccessView = "VIEW"
)

var ValidAccessLevels = map[string]bool{AccessEdit: true, AccessView: true}

const (
	ScopeGlobal  = "GLOBAL"
	ScopeContest = "CONTEST"
	nilUUID      = "00000000-0000-0000-0000-000000000000"
)

// RoleAssignment — роль пользователя со scope и уровнем доступа (для ADMIN+CONTEST).
type RoleAssignment struct {
	Code        string  `json:"code"`
	ScopeType   string  `json:"scope_type"`
	ScopeID     string  `json:"scope_id"`
	AccessLevel *string `json:"access_level"`
}

// User — запись реестра пользователей.
type User struct {
	ID           string           `json:"id"`
	Login        string           `json:"login"`
	FullName     string           `json:"full_name"`
	Email        *string          `json:"email"`
	Organization *string          `json:"organization"`
	OrgName      *string          `json:"org_name"`
	Status       string           `json:"status"`
	MustChange   bool             `json:"must_change_password"`
	LastLoginAt  *time.Time       `json:"last_login_at"`
	CreatedAt    time.Time        `json:"created_at"`
	Roles        []RoleAssignment `json:"roles"`
}

// ListFilter — параметры серверной пагинации/поиска/фильтров (SITE.md §43).
// OwnerID/AllOwners задают изоляцию по владению (§3.3): AllOwners=true (мега) — весь реестр;
// иначе только пользователи с created_by = OwnerID.
type ListFilter struct {
	Search    string
	Role      string
	Status    string
	Limit     int
	Offset    int
	OwnerID   string
	AllOwners bool
}

// Actor — субъект операции useradmin (из принципала запроса).
type Actor struct {
	UserID string
	Role   string
}

// IsMega сообщает, что актор — MEGA_ADMIN (полный доступ, §3.1).
func (a Actor) IsMega() bool { return a.Role == "MEGA_ADMIN" }

// IsSuper сообщает, что актор — SUPER_ADMIN (организатор).
func (a Actor) IsSuper() bool { return a.Role == "SUPER_ADMIN" }
