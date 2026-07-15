package contests

import (
	"context"
	"strings"
	"time"
)

// Auditor пишет события аудита (реализуется модулем audit).
type Auditor interface {
	Log(ctx context.Context, actorUserID, action, entityType, entityID string, meta map[string]any)
}

type Service struct {
	repo  *Repo
	audit Auditor
}

func NewService(repo *Repo, audit Auditor) *Service {
	return &Service{repo: repo, audit: audit}
}

// Actor — субъект операции (из принципала запроса).
type Actor struct {
	UserID  string
	IsSuper bool
}

// List возвращает конкурсы в области видимости актора, опционально фильтруя по статусу.
func (s *Service) List(ctx context.Context, a Actor, status string) ([]Contest, error) {
	return s.repo.ListForPrincipal(ctx, a.UserID, a.IsSuper, status)
}

// MyContests — конкурсы, где текущий пользователь активный участник (кабинет конкурсанта).
func (s *Service) MyContests(ctx context.Context, a Actor) ([]Contest, error) {
	return s.repo.ListForParticipant(ctx, a.UserID)
}

// Get проверяет доступ и возвращает конкурс.
func (s *Service) Get(ctx context.Context, a Actor, id string) (*Contest, error) {
	if err := s.ensureAccess(ctx, a, id); err != nil {
		return nil, err
	}
	return s.repo.ByID(ctx, id)
}

// CreateInput — поля для создания конкурса.
type CreateInput struct {
	Name     string
	Slug     string
	Desc     *string
	StartAt  *time.Time
	EndAt    *time.Time
	Timezone string
}

// Create — только SUPER_ADMIN (создание конкурсов, SITE.md §5.1).
func (s *Service) Create(ctx context.Context, a Actor, in CreateInput) (*Contest, error) {
	if !a.IsSuper {
		return nil, ErrForbidden
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, ErrValidation
	}
	slug := strings.TrimSpace(in.Slug)
	if slug == "" {
		slug = slugify(name)
	}
	if slug == "" {
		return nil, ErrValidation
	}
	tz := in.Timezone
	if tz == "" {
		tz = "Europe/Moscow"
	}
	c := &Contest{Name: name, Slug: slug, Description: in.Desc, StartAt: in.StartAt, EndAt: in.EndAt, Timezone: tz}
	id, err := s.repo.Create(ctx, c, a.UserID)
	if err != nil {
		return nil, err
	}
	s.audit.Log(ctx, a.UserID, "CONTEST_CREATED", "contest", id, map[string]any{"name": name})
	return s.repo.ByID(ctx, id)
}

func (s *Service) ensureAccess(ctx context.Context, a Actor, contestID string) error {
	ok, err := s.repo.HasContestAccess(ctx, a.UserID, contestID, a.IsSuper)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	return nil
}
