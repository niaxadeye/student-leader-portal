package challenges

import (
	"context"
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

type Service struct {
	repo   *Repo
	access ContestAccess
	audit  Auditor
}

func NewService(repo *Repo, access ContestAccess, audit Auditor) *Service {
	return &Service{repo: repo, access: access, audit: audit}
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
	Title            string
	Slug             string
	ShortDescription *string
	FullDescription  *string
	Instructions     *string
	OpenAt           *time.Time
	DeadlineAt       *time.Time
	CloseAt          *time.Time
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
	c := &Challenge{
		ContestID: contestID, Title: title, Slug: slug,
		ShortDescription: in.ShortDescription, FullDescription: in.FullDescription,
		Instructions: in.Instructions, OpenAt: in.OpenAt, DeadlineAt: in.DeadlineAt, CloseAt: in.CloseAt,
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
	if _, err := s.adminGetForEdit(ctx, a, challengeID); err != nil {
		return nil, err
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return nil, ErrValidation
	}
	upd := &Challenge{
		Title: title, ShortDescription: in.ShortDescription, FullDescription: in.FullDescription,
		Instructions: in.Instructions, OpenAt: in.OpenAt, DeadlineAt: in.DeadlineAt, CloseAt: in.CloseAt,
	}
	if err := s.repo.Update(ctx, challengeID, upd, a.UserID); err != nil {
		return nil, err
	}
	s.audit.Log(ctx, a.UserID, "CHALLENGE_UPDATED", "challenge", challengeID, nil)
	return s.repo.ByID(ctx, challengeID)
}
