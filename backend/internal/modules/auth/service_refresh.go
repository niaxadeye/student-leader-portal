package auth

import (
	"context"

	"github.com/eazytech/student-leader-cabinet/internal/platform/security"
)

// Refresh выполняет ротацию refresh-токена с детекцией повторного использования.
// Если предъявлен уже использованный/отозванный токен — отзывается всё семейство.
func (s *Service) Refresh(ctx context.Context, refreshToken, ua, ip string) (*TokenPair, error) {
	row, err := s.repo.FindRefresh(ctx, security.HashToken(refreshToken))
	if err != nil {
		return nil, err
	}
	if err := s.validateRefresh(ctx, row); err != nil {
		return nil, err
	}

	role := s.primaryRole(ctx, row.UserID)
	jti, newRefresh, newExp, err := s.newRefreshCredentials()
	if err != nil {
		return nil, err
	}
	if err := s.repo.RotateRefresh(ctx, row.ID, row.SessionID, jti, security.HashToken(newRefresh), newExp); err != nil {
		return nil, err
	}
	return s.mintTokenPair(row.UserID, role, row.SessionID, jti, newRefresh, newExp)
}

// validateRefresh проверяет предъявленное звено: повторное использование → отзыв
// всего семейства (SITE.md §16), отозванная/истёкшая сессия → отказ.
func (s *Service) validateRefresh(ctx context.Context, row *refreshRow) error {
	// Reuse detection: токен уже использован или отозван → компрометация семейства.
	if row.UsedAt != nil || row.RevokedAt != nil {
		_ = s.repo.RevokeFamily(ctx, row.FamilyID, "refresh_reuse_detected")
		s.audit.Log(ctx, row.UserID, "AUTH_REFRESH_REUSED", "session", row.SessionID, nil)
		return ErrRefreshReused
	}
	// Сессия отозвана или истекла.
	if row.SessionRevoke != nil || row.SessionExp.Before(s.now()) || row.ExpiresAt.Before(s.now()) {
		return ErrSessionExpired
	}
	return nil
}
