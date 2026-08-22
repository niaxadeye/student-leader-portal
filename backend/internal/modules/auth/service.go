package auth

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/eazytech/student-leader-cabinet/internal/modules/eventpermissions"
	"github.com/eazytech/student-leader-cabinet/internal/platform/security"
)

// Auditor записывает события аудита (реализуется модулем audit).
type Auditor interface {
	Log(ctx context.Context, actorUserID, action, entityType, entityID string, meta map[string]any)
}

type staffDirectory interface {
	GrantsForUser(ctx context.Context, userID string) ([]eventpermissions.Grant, error)
}

type juryAudienceCheck func(ctx context.Context, userID string) (bool, error)

type Service struct {
	repo      *Repo
	jwt       *security.JWTManager
	audit     Auditor
	staff     staffDirectory
	refTTL    time.Duration
	now       func() time.Time
	juryExtra juryAudienceCheck
}

func NewService(repo *Repo, jwt *security.JWTManager, audit Auditor, refreshTTL time.Duration) *Service {
	return &Service{repo: repo, jwt: jwt, audit: audit, refTTL: refreshTTL, now: time.Now}
}

func (s *Service) SetStaffDirectory(d staffDirectory) { s.staff = d }

func (s *Service) SetJuryAudienceCheck(fn juryAudienceCheck) { s.juryExtra = fn }

// LoginInput — параметры входа с контекстом клиента для сессии/аудита.
type LoginInput struct {
	Login, Password, UserAgent, IP, Audience string
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

	roles, _ := s.repo.RolesByUser(ctx, u.ID)
	if !s.audienceAllowed(ctx, in.Audience, u.ID, roles) {
		return nil, nil, ErrInvalidCredentials
	}
	role := s.tokenRole(ctx, in.Audience, u.ID, roles)
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
	jti, refresh, refreshExp, err := s.newRefreshCredentials()
	if err != nil {
		return nil, err
	}
	sess := &Session{UserID: userID, UserAgent: ua, IPHash: security.HashIP(ip), ExpiresAt: refreshExp}
	if err := s.repo.CreateSession(ctx, sess, familyID, jti, security.HashToken(refresh), refreshExp); err != nil {
		return nil, err
	}
	return s.mintTokenPair(userID, role, sess.ID, jti, refresh, refreshExp)
}

// newRefreshCredentials генерирует jti, сырой refresh-токен и срок его действия
// для нового звена цепочки. Персистит его вызывающий (CreateSession/RotateRefresh).
func (s *Service) newRefreshCredentials() (jti, refresh string, exp time.Time, err error) {
	refresh, err = security.GenerateRefreshToken()
	if err != nil {
		return "", "", time.Time{}, err
	}
	return uuid.NewString(), refresh, s.now().Add(s.refTTL), nil
}

// mintTokenPair выпускает access-JWT для уже сохранённого refresh-звена и собирает пару.
func (s *Service) mintTokenPair(userID, role, sessionID, jti, refresh string, refreshExp time.Time) (*TokenPair, error) {
	access, accessExp, err := s.jwt.Issue(userID, role, sessionID, jti)
	if err != nil {
		return nil, err
	}
	return &TokenPair{
		AccessToken: access, AccessExp: accessExp,
		RefreshToken: refresh, RefreshExp: refreshExp, SessionID: sessionID,
	}, nil
}

func (s *Service) primaryRole(ctx context.Context, userID string) string {
	roles, _ := s.repo.RolesByUser(ctx, userID)
	if s.hasRemoteJury(ctx, userID) {
		roles = append(roles, Role{Code: "JURY"})
	}
	return primaryRole(roles)
}

func (s *Service) hasRemoteJury(ctx context.Context, userID string) bool {
	if s.juryExtra == nil {
		return false
	}
	ok, err := s.juryExtra(ctx, userID)
	return err == nil && ok
}

func (s *Service) audienceAllowed(ctx context.Context, audience, userID string, roles []Role) bool {
	if roleAllowedForAudience(audience, roles) {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(audience), "jury") && s.hasRemoteJury(ctx, userID) {
		return true
	}
	return false
}

func (s *Service) tokenRole(_ context.Context, audience, _ string, roles []Role) string {
	if strings.EqualFold(strings.TrimSpace(audience), "jury") {
		return "JURY"
	}
	return primaryRole(roles)
}

func primaryRole(roles []Role) string {
	rank := map[string]int{"MEGA_ADMIN": 5, "SUPER_ADMIN": 4, "ADMIN": 3, "STAFF": 2, "JURY": 2, "REMOTE_JURY": 2, "CONTESTANT": 1}
	best, bestRank := "CONTESTANT", 0
	for _, r := range roles {
		if rank[r.Code] > bestRank {
			best, bestRank = r.Code, rank[r.Code]
		}
	}
	if best == "REMOTE_JURY" {
		return "JURY"
	}
	return best
}

func roleAllowedForAudience(audience string, roles []Role) bool {
	audience = strings.TrimSpace(strings.ToLower(audience))
	if audience == "" {
		return true
	}
	has := func(code string) bool {
		for _, role := range roles {
			if role.Code == code {
				return true
			}
		}
		return false
	}
	switch audience {
	case "admin", "staff":
		return has("MEGA_ADMIN") || has("SUPER_ADMIN") || has("ADMIN") || has("STAFF")
	case "contestant":
		return has("CONTESTANT")
	case "jury":
		return has("JURY") || has("REMOTE_JURY")
	default:
		return false
	}
}
