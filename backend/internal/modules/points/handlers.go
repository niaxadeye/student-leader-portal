package points

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/eazytech/student-leader-cabinet/internal/middleware"
	"github.com/eazytech/student-leader-cabinet/internal/modules/eventparticipants"
	"github.com/eazytech/student-leader-cabinet/internal/platform/httpserver"
)

type Handler struct {
	svc *Service
}

func NewHandler(service *Service) *Handler { return &Handler{svc: service} }

func staffActor(r *http.Request) Actor {
	principal := middleware.PrincipalFrom(r.Context())
	if principal == nil {
		return Actor{}
	}
	return Actor{UserID: principal.UserID, IsMega: principal.Role == "MEGA_ADMIN"}
}

func pagination(r *http.Request) (int, int) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	return limit, offset
}

func writeOverview(w http.ResponseWriter, r *http.Request, overview *Overview) {
	httpserver.WriteJSON(w, r, http.StatusOK, map[string]any{
		"balance": overview.Balance,
		"entries": overview.Entries,
	}, map[string]any{
		"total": overview.Total, "limit": overview.Limit, "offset": overview.Offset,
	})
}

func (h *Handler) ParticipantOverview(w http.ResponseWriter, r *http.Request) {
	principal := eventparticipants.PrincipalFrom(r.Context())
	if principal == nil {
		writeError(w, r, ErrNotFound)
		return
	}
	limit, offset := pagination(r)
	overview, err := h.svc.ParticipantOverview(
		r.Context(), principal.Event.ID, principal.Participant.ID, limit, offset,
	)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeOverview(w, r, overview)
}

func (h *Handler) AdminOverview(w http.ResponseWriter, r *http.Request) {
	limit, offset := pagination(r)
	overview, err := h.svc.AdminOverview(
		r.Context(), staffActor(r), chi.URLParam(r, "contestId"),
		chi.URLParam(r, "participantId"), limit, offset,
	)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeOverview(w, r, overview)
}

type adjustmentRequest struct {
	Amount         int64  `json:"amount"`
	Reason         string `json:"reason"`
	IdempotencyKey string `json:"idempotency_key"`
}

func (h *Handler) AdminAdjustment(w http.ResponseWriter, r *http.Request) {
	var request adjustmentRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, r, ErrValidation)
		return
	}
	if header := strings.TrimSpace(r.Header.Get("Idempotency-Key")); header != "" {
		request.IdempotencyKey = header
	}
	result, err := h.svc.Adjust(
		r.Context(), staffActor(r), chi.URLParam(r, "contestId"),
		chi.URLParam(r, "participantId"), AdjustmentInput{
			Amount: request.Amount, Reason: request.Reason, IdempotencyKey: request.IdempotencyKey,
		},
	)
	if err != nil {
		writeError(w, r, err)
		return
	}
	status := http.StatusCreated
	if result.Replayed {
		status = http.StatusOK
	}
	httpserver.WriteJSON(w, r, status, result, nil)
}

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpserver.WriteError(w, r, http.StatusNotFound, "NOT_FOUND", "Участник не найден", nil)
	case errors.Is(err, ErrForbidden):
		httpserver.WriteError(w, r, http.StatusForbidden, "FORBIDDEN", "Недостаточно прав", nil)
	case errors.Is(err, ErrValidation):
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR",
			"Укажите ненулевую целую сумму, причину и idempotency key", nil)
	case errors.Is(err, ErrIdempotencyConflict):
		httpserver.WriteError(w, r, http.StatusConflict, "IDEMPOTENCY_CONFLICT",
			"Idempotency key уже использован для другой операции", nil)
	default:
		httpserver.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка", nil)
	}
}
