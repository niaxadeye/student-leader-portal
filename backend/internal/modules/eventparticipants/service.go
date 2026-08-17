package eventparticipants

import (
	"context"
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
	CreateSession(ctx context.Context, contestID, participantID, tokenHash, userAgent, ipHash string, expiresAt time.Time) (string, error)
	AuthenticateSession(ctx context.Context, tokenHash string) (*Principal, error)
	RevokeSession(ctx context.Context, tokenHash, reason string) error
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

func (s *Service) List(ctx context.Context, actor Actor, contestID string, filter ListFilter) (*ListResult, error) {
	if err := s.ensureManage(ctx, actor, contestID); err != nil {
		return nil, err
	}
	filter.Search = strings.TrimSpace(filter.Search)
	filter.Status = strings.ToUpper(strings.TrimSpace(filter.Status))
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
	return &Participant{
		ID: participantID, ContestID: contestID, FullName: name,
		FullNameNormalized: normalized, BirthDate: input.BirthDate,
		UnionCardNumber: normalizeOptionalIdentifier(input.UnionCardNumber),
		SKSBarcode:      normalizeOptionalIdentifier(input.SKSBarcode),
	}, nil
}

func validStatus(status string) bool {
	return status == StatusActive || status == StatusBlocked || status == StatusArchived
}
