package auth

import (
	"context"

	"github.com/eazytech/student-leader-cabinet/internal/platform/security"
)

// Refresh выполняет ротацию refresh-токена с детекцией повторного использования.
// Если предъявлен уже использованный/отозванный токен — отзывается всё семейство.
func (s *Service) Refresh(ctx context.Context, refreshToken, ua, ip string) (*TokenPair, error) {
	jti, newRefresh, newExp, err := s.newRefreshCredentials()
	if err != nil {
		return nil, err
	}
	row, err := s.repo.RotateRefreshAtomically(
		ctx,
		security.HashToken(refreshToken),
		jti,
		security.HashToken(newRefresh),
		newExp,
		s.now(),
	)
	if err != nil {
		if err == ErrRefreshReused && row != nil {
			s.audit.Log(ctx, row.UserID, "AUTH_REFRESH_REUSED", "session", row.SessionID, nil)
		}
		return nil, err
	}
	role := s.primaryRole(ctx, row.UserID)
	return s.mintTokenPair(row.UserID, role, row.SessionID, jti, newRefresh, newExp)
}
