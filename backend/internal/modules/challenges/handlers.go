package challenges

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/eazytech/student-leader-cabinet/internal/middleware"
	"github.com/eazytech/student-leader-cabinet/internal/platform/httpserver"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func actorOf(r *http.Request) Actor {
	p := middleware.PrincipalFrom(r.Context())
	if p == nil {
		return Actor{}
	}
	return Actor{UserID: p.UserID, IsSuper: p.Role == "SUPER_ADMIN"}
}

func challengeJSON(c *Challenge) map[string]any {
	return map[string]any{
		"id": c.ID, "contest_id": c.ContestID, "title": c.Title, "slug": c.Slug,
		"short_description": c.ShortDescription, "full_description": c.FullDescription,
		"instructions": c.Instructions, "status": c.Status, "sort_order": c.SortOrder,
		"open_at": c.OpenAt, "deadline_at": c.DeadlineAt, "close_at": c.CloseAt,
		"current_schema_version": c.CurrentSchemaVersion, "fields_count": c.FieldsCount,
		"my_submission_status": c.MySubmissionStatus,
		"created_at":           c.CreatedAt, "updated_at": c.UpdatedAt,
		"published_at": c.PublishedAt, "archived_at": c.ArchivedAt,
	}
}

type challengeReq struct {
	Title            string     `json:"title"`
	Slug             string     `json:"slug"`
	ShortDescription *string    `json:"short_description"`
	FullDescription  *string    `json:"full_description"`
	Instructions     *string    `json:"instructions"`
	OpenAt           *time.Time `json:"open_at"`
	DeadlineAt       *time.Time `json:"deadline_at"`
	CloseAt          *time.Time `json:"close_at"`
}

func (req challengeReq) toInput() CreateInput {
	return CreateInput{
		Title: req.Title, Slug: req.Slug, ShortDescription: req.ShortDescription,
		FullDescription: req.FullDescription, Instructions: req.Instructions,
		OpenAt: req.OpenAt, DeadlineAt: req.DeadlineAt, CloseAt: req.CloseAt,
	}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.AdminList(r.Context(), actorOf(r), chi.URLParam(r, "contestId"))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	writeChallengeList(w, r, list)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req challengeReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", nil)
		return
	}
	c, err := h.svc.Create(r.Context(), actorOf(r), chi.URLParam(r, "contestId"), req.toInput())
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusCreated, challengeJSON(c), nil)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	c, err := h.svc.AdminGet(r.Context(), actorOf(r), chi.URLParam(r, "challengeId"))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, challengeJSON(c), nil)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var req challengeReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", nil)
		return
	}
	c, err := h.svc.Update(r.Context(), actorOf(r), chi.URLParam(r, "challengeId"), req.toInput())
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, challengeJSON(c), nil)
}

func (h *Handler) Duplicate(w http.ResponseWriter, r *http.Request) {
	c, err := h.svc.Duplicate(r.Context(), actorOf(r), chi.URLParam(r, "challengeId"))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusCreated, challengeJSON(c), nil)
}

func (h *Handler) transition(target string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := h.svc.Transition(r.Context(), actorOf(r), chi.URLParam(r, "challengeId"), target)
		if err != nil {
			writeErr(w, r, err)
			return
		}
		httpserver.WriteJSON(w, r, http.StatusOK, challengeJSON(c), nil)
	}
}

func (h *Handler) Publish() http.HandlerFunc { return h.transition(StatusPublished) }
func (h *Handler) Close() http.HandlerFunc   { return h.transition(StatusClosed) }
func (h *Handler) Archive() http.HandlerFunc { return h.transition(StatusArchived) }

func writeChallengeList(w http.ResponseWriter, r *http.Request, list []Challenge) {
	out := make([]map[string]any, 0, len(list))
	for i := range list {
		out = append(out, challengeJSON(&list[i]))
	}
	httpserver.WriteJSON(w, r, http.StatusOK, out, map[string]any{"count": len(out)})
}

// writeErr маппит доменные ошибки на envelope-ответы (SITE.md §50).
func writeErr(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpserver.WriteError(w, r, http.StatusNotFound, "NOT_FOUND", "Испытание не найдено", nil)
	case errors.Is(err, ErrForbidden):
		httpserver.WriteError(w, r, http.StatusForbidden, "FORBIDDEN", "Нет доступа к испытанию", nil)
	case errors.Is(err, ErrSlugTaken):
		httpserver.WriteError(w, r, http.StatusConflict, "SLUG_TAKEN", "Слаг уже занят", nil)
	case errors.Is(err, ErrFieldKey):
		httpserver.WriteError(w, r, http.StatusConflict, "FIELD_KEY_TAKEN", "Ключ поля уже используется", nil)
	case errors.Is(err, ErrBadStatus):
		httpserver.WriteError(w, r, http.StatusConflict, "INVALID_TRANSITION", "Недопустимый переход статуса", nil)
	case errors.Is(err, ErrValidation):
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Проверьте заполнение полей", nil)
	default:
		httpserver.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка", nil)
	}
}
