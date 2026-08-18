package eventpermissions

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/eazytech/student-leader-cabinet/internal/middleware"
	"github.com/eazytech/student-leader-cabinet/internal/platform/httpserver"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func actorOf(r *http.Request) Actor {
	if p := middleware.PrincipalFrom(r.Context()); p != nil {
		return Actor{UserID: p.UserID, Role: p.Role}
	}
	return Actor{}
}

func (h *Handler) ListForUser(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListByUser(r.Context(), actorOf(r), chi.URLParam(r, "userId"))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, list, nil)
}

func (h *Handler) ListForContest(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListByContest(r.Context(), actorOf(r), chi.URLParam(r, "contestId"))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, list, nil)
}

type replaceReq struct {
	ContestID   string   `json:"contest_id"`
	Permissions []string `json:"permissions"`
}

func (h *Handler) ReplaceForUser(w http.ResponseWriter, r *http.Request) {
	var req replaceReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", nil)
		return
	}
	if err := h.svc.Replace(r.Context(), actorOf(r), req.ContestID, chi.URLParam(r, "userId"), req.Permissions); err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, map[string]string{"status": "ok"}, nil)
}

func (h *Handler) ClearForUser(w http.ResponseWriter, r *http.Request) {
	contestID := r.URL.Query().Get("contest_id")
	if err := h.svc.Replace(r.Context(), actorOf(r), contestID, chi.URLParam(r, "userId"), nil); err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, map[string]string{"status": "ok"}, nil)
}

func (h *Handler) ReplaceForContest(w http.ResponseWriter, r *http.Request) {
	var req replaceReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", nil)
		return
	}
	if err := h.svc.Replace(r.Context(), actorOf(r), chi.URLParam(r, "contestId"), chi.URLParam(r, "userId"), req.Permissions); err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, map[string]string{"status": "ok"}, nil)
}

func (h *Handler) ClearForContest(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Replace(r.Context(), actorOf(r), chi.URLParam(r, "contestId"), chi.URLParam(r, "userId"), nil); err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, map[string]string{"status": "ok"}, nil)
}

func writeErr(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpserver.WriteError(w, r, http.StatusNotFound, "NOT_FOUND", "Не найдено", nil)
	case errors.Is(err, ErrValidation):
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Проверьте заполнение полей", nil)
	case errors.Is(err, ErrForbidden):
		httpserver.WriteError(w, r, http.StatusForbidden, "FORBIDDEN", "Недостаточно прав", nil)
	default:
		httpserver.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка", nil)
	}
}
