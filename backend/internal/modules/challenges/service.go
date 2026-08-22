package challenges

import (
	"context"
	"io"
	"strings"
	"time"
)

// Auditor пишет события аудита (реализуется модулем audit).
type Auditor interface {
	Log(ctx context.Context, actorUserID, action, entityType, entityID string, meta map[string]any)
}

// ContestAccess проверяет уровень доступа к конкурсу (реализуется contests.Repo).
// isMega — принципал MEGA_ADMIN (полный доступ, §3.1).
type ContestAccess interface {
	ContestViewable(ctx context.Context, userID, contestID string, isMega bool) (bool, error)
	ContestEditable(ctx context.Context, userID, contestID string, isMega bool) (bool, error)
}

type FileStore interface {
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	Remove(ctx context.Context, key string) error
}

type Service struct {
	repo       *Repo
	access     ContestAccess
	audit      Auditor
	store      FileStore
	presign    func(context.Context, string) (string, error)
	maxBytes   int64
	juryReview JuryReviewFn
}

// JuryReviewFn — проверка, что пользователь — заочное жюри испытания.
type JuryReviewFn func(ctx context.Context, userID, challengeID string, isMega bool) (bool, error)

func NewService(repo *Repo, access ContestAccess, audit Auditor) *Service {
	return &Service{repo: repo, access: access, audit: audit, maxBytes: 20 << 20}
}

func (s *Service) SetJuryReview(fn JuryReviewFn) {
	s.juryReview = fn
}

func (s *Service) SetFiles(store FileStore, presign func(context.Context, string) (string, error), maxBytes int64) {
	s.store = store
	s.presign = presign
	if maxBytes > 0 {
		s.maxBytes = maxBytes
	}
}

// Actor — субъект операции (из принципала запроса).
type Actor struct {
	UserID  string
	IsSuper bool
	IsMega  bool
}

// ensureView — доступ к конкурсу хотя бы на чтение (EDIT|VIEW).
func (s *Service) ensureView(ctx context.Context, a Actor, contestID string) error {
	ok, err := s.access.ContestViewable(ctx, a.UserID, contestID, a.IsMega)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	return nil
}

// ensureEdit — доступ к конкурсу на редактирование (владелец, EDIT-админ, мега).
func (s *Service) ensureEdit(ctx context.Context, a Actor, contestID string) error {
	ok, err := s.access.ContestEditable(ctx, a.UserID, contestID, a.IsMega)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	return nil
}

// AdminList — испытания конкурса для админа (все статусы). Доступ на чтение.
func (s *Service) AdminList(ctx context.Context, a Actor, contestID string) ([]Challenge, error) {
	if err := s.ensureView(ctx, a, contestID); err != nil {
		return nil, err
	}
	return s.repo.List(ctx, contestID, false)
}

// AdminGet — испытание для админа (проверка доступа на чтение к его конкурсу).
func (s *Service) AdminGet(ctx context.Context, a Actor, challengeID string) (*Challenge, error) {
	c, err := s.repo.ByID(ctx, challengeID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureView(ctx, a, c.ContestID); err != nil {
		return nil, err
	}
	return c, nil
}

// adminGetForEdit — как AdminGet, но требует доступ на редактирование (для мутаций).
func (s *Service) adminGetForEdit(ctx context.Context, a Actor, challengeID string) (*Challenge, error) {
	c, err := s.repo.ByID(ctx, challengeID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureEdit(ctx, a, c.ContestID); err != nil {
		return nil, err
	}
	return c, nil
}

// CreateInput — поля создания/редактирования испытания.
type CreateInput struct {
	Title              string
	Slug               string
	ShortDescription   *string
	FullDescription    *string
	Instructions       *string
	OpenAt             *time.Time
	DeadlineAt         *time.Time
	CloseAt            *time.Time
	HeldAt             *time.Time
	Venue              *string
	AcceptsSubmissions *bool
}

func trimOptional(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

func venueOrErr(s *string) (*string, error) {
	v := trimOptional(s)
	if v != nil && len([]rune(*v)) > 200 {
		return nil, ErrValidation
	}
	return v, nil
}

func acceptsOrDefault(v *bool, fallback bool) bool {
	if v == nil {
		return fallback
	}
	return *v
}

// Create создаёт испытание в статусе DRAFT (нужен доступ к конкурсу).
func (s *Service) Create(ctx context.Context, a Actor, contestID string, in CreateInput) (*Challenge, error) {
	if err := s.ensureEdit(ctx, a, contestID); err != nil {
		return nil, err
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return nil, ErrValidation
	}
	slug := strings.TrimSpace(in.Slug)
	if slug == "" {
		slug = slugify(title)
	}
	if slug == "" {
		return nil, ErrValidation
	}
	venue, err := venueOrErr(in.Venue)
	if err != nil {
		return nil, err
	}
	c := &Challenge{
		ContestID: contestID, Title: title, Slug: slug,
		ShortDescription: in.ShortDescription, FullDescription: in.FullDescription,
		Instructions: in.Instructions, OpenAt: in.OpenAt, DeadlineAt: in.DeadlineAt, CloseAt: in.CloseAt,
		HeldAt: in.HeldAt, Venue: venue, AcceptsSubmissions: acceptsOrDefault(in.AcceptsSubmissions, true),
	}
	id, err := s.repo.Create(ctx, c, a.UserID)
	if err != nil {
		return nil, err
	}
	s.audit.Log(ctx, a.UserID, "CHALLENGE_CREATED", "challenge", id, map[string]any{"contest_id": contestID, "title": title})
	return s.repo.ByID(ctx, id)
}

// Update редактирует мету испытания и, если оно опубликовано, версионирует схему.
func (s *Service) Update(ctx context.Context, a Actor, challengeID string, in CreateInput) (*Challenge, error) {
	cur, err := s.adminGetForEdit(ctx, a, challengeID)
	if err != nil {
		return nil, err
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return nil, ErrValidation
	}
	venue, err := venueOrErr(in.Venue)
	if err != nil {
		return nil, err
	}
	upd := &Challenge{
		Title: title, ShortDescription: in.ShortDescription, FullDescription: in.FullDescription,
		Instructions: in.Instructions, OpenAt: in.OpenAt, DeadlineAt: in.DeadlineAt, CloseAt: in.CloseAt,
		HeldAt: in.HeldAt, Venue: venue,
		AcceptsSubmissions: acceptsOrDefault(in.AcceptsSubmissions, cur.AcceptsSubmissions),
	}
	if err := s.repo.Update(ctx, challengeID, upd, a.UserID); err != nil {
		return nil, err
	}
	s.audit.Log(ctx, a.UserID, "CHALLENGE_UPDATED", "challenge", challengeID, nil)
	return s.repo.ByID(ctx, challengeID)
}
