package auth

import (
	"context"

	"github.com/google/uuid"

	"github.com/eazytech/student-leader-cabinet/internal/platform/security"
)

// Refresh выполняет ротацию refresh-токена с детекцией повторного использования.
// Если предъявлен уже использованный/отозванный токен — отзывается всё семейство.
func (s *Service) Refresh(ctx context.Context, refreshToken, ua, ip string) (*TokenPair, error) {
	row, err := s.repo.FindRefresh(ctx, security.HashToken(refreshToken))
	if err != nil {
		return nil, err
	}

	// Reuse detection: токен уже использован или отозван → компрометация семейства.
	if row.UsedAt != nil || row.RevokedAt != nil {
		_ = s.repo.RevokeFamily(ctx, row.FamilyID, "refresh_reuse_detected")
		s.audit.Log(ctx, row.UserID, "AUTH_REFRESH_REUSED", "session", row.SessionID, nil)
		return nil, ErrRefreshReused
	}
	// Сессия отозвана или истекла.
	if row.SessionRevoke != nil || row.SessionExp.Before(s.now()) || row.ExpiresAt.Before(s.now()) {
		return nil, ErrSessionExpired
	}

	role := s.primaryRole(ctx, row.UserID)
	jti := uuid.NewString()
	newRefresh, err := security.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}
	newExp := s.now().Add(s.refTTL)
	if err := s.repo.RotateRefresh(ctx, row.ID, row.SessionID, jti, security.HashToken(newRefresh), newExp); err != nil {
		return nil, err
	}
	access, accessExp, err := s.jwt.Issue(row.UserID, role, row.SessionID, jti)
	if err != nil {
		return nil, err
	}
	return &TokenPair{
		AccessToken: access, AccessExp: accessExp,
		RefreshToken: newRefresh, RefreshExp: newExp, SessionID: row.SessionID,
	}, nil
}
