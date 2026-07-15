package useradmin

import (
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

func actorID(r *http.Request) string {
	if p := middleware.PrincipalFrom(r.Context()); p != nil {
		return p.UserID
	}
	return ""
}

func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	temp, err := h.svc.ResetPassword(r.Context(), actorID(r), chi.URLParam(r, "userId"))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, map[string]any{"temp_password": temp}, nil)
}

func (h *Handler) Block(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.SetStatus(r.Context(), actorID(r), chi.URLParam(r, "userId"), "BLOCKED"); err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, map[string]string{"status": "BLOCKED"}, nil)
}

func (h *Handler) Unblock(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.SetStatus(r.Context(), actorID(r), chi.URLParam(r, "userId"), "ACTIVE"); err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, map[string]string{"status": "ACTIVE"}, nil)
}

func writeErr(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrUserNotFound):
		httpserver.WriteError(w, r, http.StatusNotFound, "NOT_FOUND", "Пользователь не найден", nil)
	case errors.Is(err, ErrRoleNotFound):
		httpserver.WriteError(w, r, http.StatusNotFound, "ROLE_NOT_FOUND", "Роль не найдена", nil)
	case errors.Is(err, ErrLoginTaken):
		httpserver.WriteError(w, r, http.StatusConflict, "LOGIN_TAKEN", "Логин уже занят", nil)
	case errors.Is(err, ErrValidation):
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Проверьте заполнение полей", nil)
	default:
		httpserver.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка", nil)
	}
}
