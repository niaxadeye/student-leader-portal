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
//   IsSuper — роль SUPER_ADMIN (организатор-создатель).
//   IsMega  — роль MEGA_ADMIN (полный кросс-арендный доступ, §3.1).
type Actor struct {
	UserID  string
	IsSuper bool
	IsMega  bool
}

// List возвращает конкурсы в области видимости актора, опционально фильтруя по статусу.
func (s *Service) List(ctx context.Context, a Actor, status string) ([]Contest, error) {
	return s.repo.ListForPrincipal(ctx, a.UserID, a.IsMega, status)
}

// MyContests — конкурсы, где текущий пользователь активный участник (кабинет конкурсанта).
func (s *Service) MyContests(ctx context.Context, a Actor) ([]Contest, error) {
	return s.repo.ListForParticipant(ctx, a.UserID)
}

// Get проверяет доступ (чтение), возвращает конкурс с проставленным уровнем доступа актора.
func (s *Service) Get(ctx context.Context, a Actor, id string) (*Contest, error) {
	if err := s.ensureView(ctx, a, id); err != nil {
		return nil, err
	}
	c, err := s.repo.ByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if lvl, err := s.AccessFor(ctx, a, id); err == nil {
		c.AccessLevel = lvl
	}
	return c, nil
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

// Create — SUPER_ADMIN (владелец) или MEGA_ADMIN (§1.3). Создатель становится owner_user_id.
func (s *Service) Create(ctx context.Context, a Actor, in CreateInput) (*Contest, error) {
	if !a.IsSuper && !a.IsMega {
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

// ensureView — доступ хотя бы на чтение (EDIT или VIEW).
func (s *Service) ensureView(ctx context.Context, a Actor, contestID string) error {
	lvl, err := s.repo.AccessLevel(ctx, a.UserID, contestID, a.IsMega)
	if err != nil {
		return err
	}
	if !lvl.CanView() {
		return ErrForbidden
	}
	return nil
}

// ensureEdit — доступ на редактирование (владелец, назначенный EDIT-админ или мега).
func (s *Service) ensureEdit(ctx context.Context, a Actor, contestID string) error {
	lvl, err := s.repo.AccessLevel(ctx, a.UserID, contestID, a.IsMega)
	if err != nil {
		return err
	}
	if !lvl.CanEdit() {
		return ErrForbidden
	}
	return nil
}

// AccessFor возвращает строковый уровень доступа актора к конкурсу для UI (§4):
// OWNER (владелец или мега) | EDIT | VIEW. Пустая строка, если доступа нет.
func (s *Service) AccessFor(ctx context.Context, a Actor, contestID string) (string, error) {
	if a.IsMega {
		return "OWNER", nil
	}
	owner, err := s.repo.IsOwner(ctx, a.UserID, contestID)
	if err != nil {
		return "", err
	}
	if owner {
		return "OWNER", nil
	}
	lvl, err := s.repo.AccessLevel(ctx, a.UserID, contestID, false)
	if err != nil {
		return "", err
	}
	switch lvl {
	case AccessEdit:
		return "EDIT", nil
	case AccessView:
		return "VIEW", nil
	default:
		return "", nil
	}
}

// ensureOwnerOrMega — операции, доступные только владельцу конкурса или мега (§3.6):
// управление составом участников. EDIT-админ сюда НЕ допускается.
func (s *Service) ensureOwnerOrMega(ctx context.Context, a Actor, contestID string) error {
	if a.IsMega {
		return nil
	}
	owner, err := s.repo.IsOwner(ctx, a.UserID, contestID)
	if err != nil {
		return err
	}
	if !owner {
		return ErrForbidden
	}
	return nil
}
