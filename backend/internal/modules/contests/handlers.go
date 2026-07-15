package contests

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/eazytech/student-leader-cabinet/internal/middleware"
	"github.com/eazytech/student-leader-cabinet/internal/platform/httpserver"
)

// ImageStore — часть storage, нужная модулю (presigned + запись/удаление обложек).
type ImageStore interface {
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	Remove(ctx context.Context, key string) error
	PresignGet(ctx context.Context, key string) (string, error)
}

type Handler struct {
	svc   *Service
	store ImageStore
}

func NewHandler(svc *Service, store ImageStore) *Handler {
	return &Handler{svc: svc, store: store}
}

func actorOf(r *http.Request) Actor {
	p := middleware.PrincipalFrom(r.Context())
	if p == nil {
		return Actor{}
	}
	return Actor{UserID: p.UserID, IsSuper: p.Role == "SUPER_ADMIN"}
}

// contestJSON сериализует конкурс. image_url — presigned-ссылка (или null),
// генерируется best-effort: ошибка подписи не должна ронять ответ.
func (h *Handler) contestJSON(ctx context.Context, c *Contest) map[string]any {
	var imageURL any
	if c.ImageKey != nil && *c.ImageKey != "" && h.store != nil {
		if u, err := h.store.PresignGet(ctx, *c.ImageKey); err == nil {
			imageURL = u
		}
	}
	return map[string]any{
		"id": c.ID, "name": c.Name, "slug": c.Slug, "description": c.Description,
		"status": c.Status, "start_at": c.StartAt, "end_at": c.EndAt,
		"timezone": c.Timezone, "participants_count": c.ParticipantsCount,
		"challenges_count": c.ChallengesCount,
		"image_url":  imageURL,
		"created_at": c.CreatedAt, "updated_at": c.UpdatedAt, "archived_at": c.ArchivedAt,
	}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	list, err := h.svc.List(r.Context(), actorOf(r), status)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	out := make([]map[string]any, 0, len(list))
	for i := range list {
		out = append(out, h.contestJSON(r.Context(), &list[i]))
	}
	httpserver.WriteJSON(w, r, http.StatusOK, out, map[string]any{"count": len(out)})
}

// MyContests — GET /my/contests: конкурсы, где пользователь участник (кабинет конкурсанта).
func (h *Handler) MyContests(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.MyContests(r.Context(), actorOf(r))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	out := make([]map[string]any, 0, len(list))
	for i := range list {
		out = append(out, h.contestJSON(r.Context(), &list[i]))
	}
	httpserver.WriteJSON(w, r, http.StatusOK, out, map[string]any{"count": len(out)})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	c, err := h.svc.Get(r.Context(), actorOf(r), chi.URLParam(r, "contestId"))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, h.contestJSON(r.Context(), c), nil)
}

type contestReq struct {
	Name        string     `json:"name"`
	Slug        string     `json:"slug"`
	Description *string    `json:"description"`
	StartAt     *time.Time `json:"start_at"`
	EndAt       *time.Time `json:"end_at"`
	Timezone    string     `json:"timezone"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req contestReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", nil)
		return
	}
	c, err := h.svc.Create(r.Context(), actorOf(r), CreateInput{
		Name: req.Name, Slug: req.Slug, Desc: req.Description,
		StartAt: req.StartAt, EndAt: req.EndAt, Timezone: req.Timezone,
	})
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusCreated, h.contestJSON(r.Context(), c), nil)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var req contestReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", nil)
		return
	}
	c, err := h.svc.Update(r.Context(), actorOf(r), chi.URLParam(r, "contestId"), CreateInput{
		Name: req.Name, Desc: req.Description, StartAt: req.StartAt, EndAt: req.EndAt, Timezone: req.Timezone,
	})
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, h.contestJSON(r.Context(), c), nil)
}

func (h *Handler) transition(target string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := h.svc.Transition(r.Context(), actorOf(r), chi.URLParam(r, "contestId"), target)
		if err != nil {
			writeErr(w, r, err)
			return
		}
		httpserver.WriteJSON(w, r, http.StatusOK, h.contestJSON(r.Context(), c), nil)
	}
}

func (h *Handler) Publish() http.HandlerFunc { return h.transition(StatusActive) }
func (h *Handler) Finish() http.HandlerFunc  { return h.transition(StatusFinished) }
func (h *Handler) Archive() http.HandlerFunc { return h.transition(StatusArchived) }

// writeErr маппит доменные ошибки на envelope-ответы (SITE.md §50).
func writeErr(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpserver.WriteError(w, r, http.StatusNotFound, "NOT_FOUND", "Конкурс не найден", nil)
	case errors.Is(err, ErrForbidden):
		httpserver.WriteError(w, r, http.StatusForbidden, "FORBIDDEN", "Нет доступа к конкурсу", nil)
	case errors.Is(err, ErrSlugTaken):
		httpserver.WriteError(w, r, http.StatusConflict, "SLUG_TAKEN", "Слаг уже занят", nil)
	case errors.Is(err, ErrBadStatus):
		httpserver.WriteError(w, r, http.StatusConflict, "INVALID_TRANSITION", "Недопустимый переход статуса", nil)
	case errors.Is(err, ErrValidation):
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Проверьте заполнение полей", nil)
	default:
		httpserver.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка", nil)
	}
}
