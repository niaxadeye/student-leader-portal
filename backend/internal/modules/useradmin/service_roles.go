package useradmin

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
)

// AssignRoleInput — назначение роли пользователю.
// AccessLevel (EDIT|VIEW) обязателен для роли ADMIN на конкретный конкурс, иначе пуст (§3.4).
type AssignRoleInput struct {
	Role        string
	ScopeType   string
	ScopeID     string
	AccessLevel string
}

func normScope(in AssignRoleInput) (AssignRoleInput, bool) {
	in.Role = strings.ToUpper(strings.TrimSpace(in.Role))
	if !ValidRoles[in.Role] {
		return in, false
	}
	in.ScopeType = strings.ToUpper(strings.TrimSpace(in.ScopeType))
	if in.ScopeType == "" {
		in.ScopeType = ScopeGlobal
	}
	in.AccessLevel = strings.ToUpper(strings.TrimSpace(in.AccessLevel))
	switch in.ScopeType {
	case ScopeGlobal:
		in.ScopeID = nilUUID
		in.AccessLevel = "" // уровень доступа только для ADMIN+CONTEST
	case ScopeContest:
		if strings.TrimSpace(in.ScopeID) == "" {
			return in, false
		}
		// Для ADMIN на конкурс уровень обязателен и валиден; для прочих ролей — не задаём.
		if in.Role == "ADMIN" {
			if !ValidAccessLevels[in.AccessLevel] {
				return in, false
			}
		} else {
			in.AccessLevel = ""
		}
	default:
		return in, false
	}
	return in, true
}

// ensureCanGrant проверяет право актора назначать/снимать данную роль и scope (§3.3–3.4):
//   - роль SUPER_ADMIN/MEGA_ADMIN — только мега (как и создание);
//   - доступ ADMIN к конкретному конкурсу — только владелец конкурса или мега.
func (s *Service) ensureCanGrant(ctx context.Context, a Actor, norm AssignRoleInput) error {
	if a.IsMega() {
		return nil
	}
	if !canCreateRole(a, norm.Role) {
		return ErrForbidden
	}
	// Назначение доступа к конкурсу может делать только его владелец.
	if norm.ScopeType == ScopeContest {
		var owner bool
		err := s.pool.QueryRow(ctx,
			`SELECT owner_user_id = $1 FROM contests WHERE id = $2`, a.UserID, norm.ScopeID).Scan(&owner)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrValidation
		}
		if err != nil {
			return err
		}
		if !owner {
			return ErrForbidden
		}
	}
	return nil
}

// AssignRole назначает роль (валидирует код и scope). Идемпотентно.
func (s *Service) AssignRole(ctx context.Context, a Actor, userID string, in AssignRoleInput) error {
	norm, ok := normScope(in)
	if !ok {
		return ErrValidation
	}
	if err := s.ensureCanGrant(ctx, a, norm); err != nil {
		return err
	}
	if _, err := s.repo.ByID(ctx, userID); err != nil {
		return err
	}
	if err := s.repo.AssignRole(ctx, userID, norm.Role, norm.ScopeType, norm.ScopeID, norm.AccessLevel); err != nil {
		return err
	}
	s.audit.Log(ctx, a.UserID, "ROLE_ASSIGNED", "user", userID,
		map[string]any{"role": norm.Role, "scope_type": norm.ScopeType, "scope_id": norm.ScopeID, "access_level": norm.AccessLevel})
	return nil
}

// RemoveRole снимает роль со scope.
func (s *Service) RemoveRole(ctx context.Context, a Actor, userID string, in AssignRoleInput) error {
	norm, ok := normScope(in)
	if !ok {
		return ErrValidation
	}
	if err := s.ensureCanGrant(ctx, a, norm); err != nil {
		return err
	}
	if _, err := s.repo.ByID(ctx, userID); err != nil {
		return err
	}
	if err := s.repo.RemoveRole(ctx, userID, norm.Role, norm.ScopeType, norm.ScopeID); err != nil {
		return err
	}
	s.audit.Log(ctx, a.UserID, "ROLE_REMOVED", "user", userID,
		map[string]any{"role": norm.Role, "scope_type": norm.ScopeType, "scope_id": norm.ScopeID})
	return nil
}
