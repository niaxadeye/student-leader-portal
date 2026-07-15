package auth

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/eazytech/student-leader-cabinet/internal/platform/security"
)

// Auditor записывает события аудита (реализуется модулем audit).
type Auditor interface {
	Log(ctx context.Context, actorUserID, action, entityType, entityID string, meta map[string]any)
}

type Service struct {
	repo    *Repo
	jwt     *security.JWTManager
	audit   Auditor
	refTTL  time.Duration
	now     func() time.Time
}

func NewService(repo *Repo, jwt *security.JWTManager, audit Auditor, refreshTTL time.Duration) *Service {
	return &Service{repo: repo, jwt: jwt, audit: audit, refTTL: refreshTTL, now: time.Now}
}

// LoginInput — параметры входа с контекстом клиента для сессии/аудита.
type LoginInput struct {
	Login, Password, UserAgent, IP string
}

// Login проверяет пароль, статус и блокировку, создаёт сессию и выдаёт пару токенов.
func (s *Service) Login(ctx context.Context, in LoginInput) (*TokenPair, *User, error) {
	u, err := s.repo.UserByLogin(ctx, in.Login)
	if err != nil {
		return nil, nil, ErrInvalidCredentials // не раскрываем, что логина нет (SITE.md §49.1)
	}
	if u.Status == StatusBlocked {
		return nil, nil, ErrAccountBlocked
	}
	if u.LockedUntil != nil && u.LockedUntil.After(s.now()) {
		return nil, nil, ErrAccountLocked
	}
	if err := security.VerifyPassword(in.Password, u.PasswordHash); err != nil {
		_ = s.repo.RecordLoginFailure(ctx, u.ID, lockDuration, maxFailedLogins)
		s.audit.Log(ctx, u.ID, "AUTH_LOGIN_FAILED", "user", u.ID, nil)
		return nil, nil, ErrInvalidCredentials
	}

	role := s.primaryRole(ctx, u.ID)
	pair, err := s.issueSession(ctx, u.ID, role, in.UserAgent, in.IP)
	if err != nil {
		return nil, nil, err
	}
	_ = s.repo.RecordLoginSuccess(ctx, u.ID)
	s.audit.Log(ctx, u.ID, "AUTH_LOGIN_SUCCESS", "user", u.ID, map[string]any{"session_id": pair.SessionID})
	return pair, u, nil
}

// issueSession создаёт сессию с новым token family и первым refresh-токеном.
func (s *Service) issueSession(ctx context.Context, userID, role, ua, ip string) (*TokenPair, error) {
	familyID := uuid.NewString()
	jti := uuid.NewString()
	refresh, err := security.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}
	refreshExp := s.now().Add(s.refTTL)
	sess := &Session{UserID: userID, UserAgent: ua, IPHash: security.HashIP(ip), ExpiresAt: refreshExp}
	if err := s.repo.CreateSession(ctx, sess, familyID, jti, security.HashToken(refresh), refreshExp); err != nil {
		return nil, err
	}
	access, accessExp, err := s.jwt.Issue(userID, role, sess.ID, jti)
	if err != nil {
		return nil, err
	}
	return &TokenPair{
		AccessToken: access, AccessExp: accessExp,
		RefreshToken: refresh, RefreshExp: refreshExp, SessionID: sess.ID,
	}, nil
}

func (s *Service) primaryRole(ctx context.Context, userID string) string {
	roles, _ := s.repo.RolesByUser(ctx, userID)
	// Приоритет: SUPER_ADMIN > ADMIN > CONTESTANT.
	rank := map[string]int{"SUPER_ADMIN": 3, "ADMIN": 2, "CONTESTANT": 1}
	best, bestRank := "CONTESTANT", 0
	for _, r := range roles {
		if rank[r.Code] > bestRank {
			best, bestRank = r.Code, rank[r.Code]
		}
	}
	return best
}
