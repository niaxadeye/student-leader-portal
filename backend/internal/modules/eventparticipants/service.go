package eventparticipants

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/eazytech/student-leader-cabinet/internal/platform/security"
)

const (
	defaultLimit = 50
	maxLimit     = 200
)

type Repository interface {
	CanManage(ctx context.Context, userID, contestID string) (bool, error)
	CanAccessDirections(ctx context.Context, userID, contestID string) (bool, error)
	List(ctx context.Context, contestID string, filter ListFilter) ([]Participant, int, error)
	All(ctx context.Context, contestID string) ([]Participant, error)
	ByID(ctx context.Context, contestID, participantID string) (*Participant, error)
	Create(ctx context.Context, participant *Participant) (string, error)
	Update(ctx context.Context, participant *Participant) error
	SetStatus(ctx context.Context, contestID, participantID, status string) error
	EventBySlug(ctx context.Context, slug string) (*EventRef, error)
	FindByNameBirth(ctx context.Context, contestID, normalizedName string, birthDate time.Time) ([]Participant, error)
	FindByUnionCard(ctx context.Context, contestID, number string) (*Participant, error)
	FindBySKSBarcode(ctx context.Context, contestID, barcode string) (*Participant, error)
	ListActiveEvents(ctx context.Context) ([]EventRef, error)
	FindByTelegramUserID(ctx context.Context, contestID string, userID int64) (*Participant, error)
	FindByVKUserID(ctx context.Context, contestID string, userID int64) (*Participant, error)
	FindByTelegramUsername(ctx context.Context, contestID, username string) (*Participant, error)
	FindByVKIdentity(ctx context.Context, contestID string, userID int64, screenName string) (*Participant, error)
	ListActiveByTelegramUserID(ctx context.Context, userID int64) ([]ParticipantEventMatch, error)
	ListActiveByTelegramUsername(ctx context.Context, username string) ([]ParticipantEventMatch, error)
	ListActiveByVKUserID(ctx context.Context, userID int64) ([]ParticipantEventMatch, error)
	ListActiveByVKIdentity(ctx context.Context, userID int64, screenName string) ([]ParticipantEventMatch, error)
	BindTelegram(ctx context.Context, contestID, participantID string, userID int64, profileURL *string) error
	BindVK(ctx context.Context, contestID, participantID string, userID int64, profileURL *string) error
	CreateSession(ctx context.Context, contestID, participantID, tokenHash, userAgent, ipHash string, expiresAt time.Time) (string, error)
	AuthenticateSession(ctx context.Context, tokenHash string) (*Principal, error)
	RevokeSession(ctx context.Context, tokenHash, reason string) error
	ListDirections(ctx context.Context, contestID string) ([]Direction, error)
	CreateDirection(ctx context.Context, contestID, name string) (*Direction, error)
	UpdateDirection(ctx context.Context, contestID, directionID, name string) (*Direction, error)
	DeleteDirection(ctx context.Context, contestID, directionID string) error
	EnsureDirection(ctx context.Context, contestID, name string) (*Direction, error)
	DirectionInContest(ctx context.Context, contestID, directionID string) (bool, error)
}

type Auditor interface {
	Log(ctx context.Context, actorUserID, action, entityType, entityID string, meta map[string]any)
	LogParticipant(ctx context.Context, participantID, contestID, action, entityType, entityID string, meta map[string]any)
}

type Service struct {
	repo       Repository
	audit      Auditor
	sessionTTL time.Duration
	now        func() time.Time
	newToken   func() (string, error)
	social     SocialAuth
	http       httpDoer
}

type SocialAuth struct {
	TelegramBotToken    string
	TelegramBotUsername string
	VKClientID          string
	VKClientSecret      string
	VKServiceToken      string
	VKRedirectURL       string
	PublicBaseURL       string
	StateSecret         string
}

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

func (c SocialAuth) TelegramEnabled() bool {
	return strings.TrimSpace(c.TelegramBotToken) != "" && telegramBotID(c.TelegramBotToken) != ""
}

func (c SocialAuth) VKEnabled() bool {
	return strings.TrimSpace(c.VKClientID) != "" && strings.TrimSpace(c.VKRedirectURL) != ""
}

func NewService(repo Repository, audit Auditor, sessionTTL time.Duration) *Service {
	if sessionTTL <= 0 {
		sessionTTL = 12 * time.Hour
	}
	return &Service{
		repo: repo, audit: audit, sessionTTL: sessionTTL,
		now: time.Now, newToken: security.GenerateRefreshToken,
	}
}

func (s *Service) SetSocialAuth(cfg SocialAuth, client httpDoer) {
	s.social = cfg
	s.http = client
}

func (s *Service) ensureManage(ctx context.Context, actor Actor, contestID string) error {
	if actor.IsMega {
		return nil
	}
	allowed, err := s.repo.CanManage(ctx, actor.UserID, contestID)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrForbidden
	}
	return nil
}

func (s *Service) ensureDirectionRead(ctx context.Context, actor Actor, contestID string) error {
	if actor.IsMega {
		return nil
	}
	allowed, err := s.repo.CanAccessDirections(ctx, actor.UserID, contestID)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrForbidden
	}
	return nil
}

func (s *Service) List(ctx context.Context, actor Actor, contestID string, filter ListFilter) (*ListResult, error) {
	if err := s.ensureManage(ctx, actor, contestID); err != nil {
		return nil, err
	}
	filter.Search = strings.TrimSpace(filter.Search)
	filter.Status = strings.ToUpper(strings.TrimSpace(filter.Status))
	filter.DirectionID = strings.TrimSpace(filter.DirectionID)
	if filter.Status != "" && !validStatus(filter.Status) {
		return nil, ErrValidation
	}
	if filter.Limit <= 0 {
		filter.Limit = defaultLimit
	}
	if filter.Limit > maxLimit {
		filter.Limit = maxLimit
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	participants, total, err := s.repo.List(ctx, contestID, filter)
	if err != nil {
		return nil, err
	}
	if participants == nil {
		participants = []Participant{}
	}
	return &ListResult{Participants: participants, Total: total, Limit: filter.Limit, Offset: filter.Offset}, nil
}

func (s *Service) Get(ctx context.Context, actor Actor, contestID, participantID string) (*Participant, error) {
	if err := s.ensureManage(ctx, actor, contestID); err != nil {
		return nil, err
	}
	return s.repo.ByID(ctx, contestID, participantID)
}

func (s *Service) Create(ctx context.Context, actor Actor, contestID string, input CreateInput) (*Participant, error) {
	if err := s.ensureManage(ctx, actor, contestID); err != nil {
		return nil, err
	}
	p, err := normalizeParticipantInput(contestID, "", input, s.now())
	if err != nil {
		return nil, err
	}
	p.DirectionID, err = s.resolveDirectionID(ctx, contestID, input.DirectionID)
	if err != nil {
		return nil, err
	}
	s.fillVKUserID(ctx, p, nil, nil)
	id, err := s.repo.Create(ctx, p)
	if err != nil {
		return nil, err
	}
	s.audit.Log(ctx, actor.UserID, "EVENT_PARTICIPANT_CREATED", "event_participant", id,
		map[string]any{"contest_id": contestID})
	return s.repo.ByID(ctx, contestID, id)
}

func (s *Service) Update(ctx context.Context, actor Actor, contestID, participantID string, input UpdateInput) (*Participant, error) {
	if err := s.ensureManage(ctx, actor, contestID); err != nil {
		return nil, err
	}
	p, err := normalizeParticipantInput(contestID, participantID, input, s.now())
	if err != nil {
		return nil, err
	}
	p.DirectionID, err = s.resolveDirectionID(ctx, contestID, input.DirectionID)
	if err != nil {
		return nil, err
	}
	previous, err := s.repo.ByID(ctx, contestID, participantID)
	if err != nil {
		return nil, err
	}
	s.fillVKUserID(ctx, p, previous, nil)
	if err := s.repo.Update(ctx, p); err != nil {
		return nil, err
	}
	s.audit.Log(ctx, actor.UserID, "EVENT_PARTICIPANT_UPDATED", "event_participant", participantID,
		map[string]any{"contest_id": contestID})
	return s.repo.ByID(ctx, contestID, participantID)
}

func (s *Service) SetStatus(ctx context.Context, actor Actor, contestID, participantID, target string) (*Participant, error) {
	if err := s.ensureManage(ctx, actor, contestID); err != nil {
		return nil, err
	}
	target = strings.ToUpper(strings.TrimSpace(target))
	if !validStatus(target) {
		return nil, ErrValidation
	}
	current, err := s.repo.ByID(ctx, contestID, participantID)
	if err != nil {
		return nil, err
	}
	if current.Status == StatusArchived && target != StatusArchived {
		return nil, ErrValidation
	}
	if current.Status != target {
		if err := s.repo.SetStatus(ctx, contestID, participantID, target); err != nil {
			return nil, err
		}
		s.audit.Log(ctx, actor.UserID, "EVENT_PARTICIPANT_STATUS_CHANGED", "event_participant", participantID,
			map[string]any{"contest_id": contestID, "from": current.Status, "to": target})
	}
	return s.repo.ByID(ctx, contestID, participantID)
}

func normalizeParticipantInput(contestID, participantID string, input CreateInput, now time.Time) (*Participant, error) {
	name := cleanFullName(input.FullName)
	normalized := NormalizeFullName(name)
	if name == "" || normalized == "" || input.BirthDate.IsZero() || input.BirthDate.After(now) {
		return nil, ErrValidation
	}
	vkURL, err := normalizeOptionalSocialURL(socialVK, input.VKURL)
	if err != nil {
		return nil, err
	}
	telegramURL, err := normalizeOptionalSocialURL(socialTelegram, input.TelegramURL)
	if err != nil {
		return nil, err
	}
	return &Participant{
		ID: participantID, ContestID: contestID, FullName: name,
		FullNameNormalized: normalized, BirthDate: input.BirthDate,
		UnionCardNumber: normalizeOptionalIdentifier(input.UnionCardNumber),
		SKSBarcode:      normalizeOptionalIdentifier(input.SKSBarcode),
		VKURL:           vkURL,
		TelegramURL:     telegramURL,
	}, nil
}

func (s *Service) resolveDirectionID(ctx context.Context, contestID string, id *string) (*string, error) {
	if id == nil {
		return nil, nil
	}
	value := strings.TrimSpace(*id)
	if value == "" {
		return nil, nil
	}
	ok, err := s.repo.DirectionInContest(ctx, contestID, value)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrValidation
	}
	return &value, nil
}

func validStatus(status string) bool {
	return status == StatusActive || status == StatusBlocked || status == StatusArchived
}
