package useradmin

import (
	"context"
	"strings"
)

// AssignRoleInput — назначение роли пользователю.
type AssignRoleInput struct {
	Role      string
	ScopeType string
	ScopeID   string
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
	switch in.ScopeType {
	case ScopeGlobal:
		in.ScopeID = nilUUID
	case ScopeContest:
		if strings.TrimSpace(in.ScopeID) == "" {
			return in, false
		}
	default:
		return in, false
	}
	return in, true
}

// AssignRole назначает роль (валидирует код и scope). Идемпотентно.
func (s *Service) AssignRole(ctx context.Context, actorID, userID string, in AssignRoleInput) error {
	norm, ok := normScope(in)
	if !ok {
		return ErrValidation
	}
	if _, err := s.repo.ByID(ctx, userID); err != nil {
		return err
	}
	if err := s.repo.AssignRole(ctx, userID, norm.Role, norm.ScopeType, norm.ScopeID); err != nil {
		return err
	}
	s.audit.Log(ctx, actorID, "ROLE_ASSIGNED", "user", userID,
		map[string]any{"role": norm.Role, "scope_type": norm.ScopeType, "scope_id": norm.ScopeID})
	return nil
}

// RemoveRole снимает роль со scope.
func (s *Service) RemoveRole(ctx context.Context, actorID, userID string, in AssignRoleInput) error {
	norm, ok := normScope(in)
	if !ok {
		return ErrValidation
	}
	if _, err := s.repo.ByID(ctx, userID); err != nil {
		return err
	}
	if err := s.repo.RemoveRole(ctx, userID, norm.Role, norm.ScopeType, norm.ScopeID); err != nil {
		return err
	}
	s.audit.Log(ctx, actorID, "ROLE_REMOVED", "user", userID,
		map[string]any{"role": norm.Role, "scope_type": norm.ScopeType, "scope_id": norm.ScopeID})
	return nil
}
