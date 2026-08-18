package lectures

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/eazytech/student-leader-cabinet/internal/middleware"
	"github.com/eazytech/student-leader-cabinet/internal/modules/eventparticipants"
	"github.com/eazytech/student-leader-cabinet/internal/platform/httpserver"
)

type Handler struct {
	svc *Service
}

func NewHandler(service *Service) *Handler { return &Handler{svc: service} }

func actorFrom(r *http.Request) Actor {
	principal := middleware.PrincipalFrom(r.Context())
	if principal == nil {
		return Actor{}
	}
	return Actor{UserID: principal.UserID, IsMega: principal.Role == "MEGA_ADMIN"}
}

func (h *Handler) AdminList(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.List(r.Context(), actorFrom(r), chi.URLParam(r, "contestId"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, list, map[string]any{"count": len(list)})
}

func (h *Handler) AdminGet(w http.ResponseWriter, r *http.Request) {
	lecture, err := h.svc.Get(r.Context(), actorFrom(r), chi.URLParam(r, "contestId"), chi.URLParam(r, "lectureId"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, lecture, nil)
}

func (h *Handler) AdminCreate(w http.ResponseWriter, r *http.Request) {
	var input LectureInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, r, ErrValidation)
		return
	}
	lecture, err := h.svc.Create(r.Context(), actorFrom(r), chi.URLParam(r, "contestId"), input)
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusCreated, lecture, nil)
}

func (h *Handler) AdminUpdate(w http.ResponseWriter, r *http.Request) {
	var input LectureInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, r, ErrValidation)
		return
	}
	lecture, err := h.svc.Update(r.Context(), actorFrom(r), chi.URLParam(r, "contestId"), chi.URLParam(r, "lectureId"), input)
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, lecture, nil)
}

func (h *Handler) transition(activate bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var lecture *Lecture
		var err error
		if activate {
			lecture, err = h.svc.Activate(r.Context(), actorFrom(r), chi.URLParam(r, "contestId"), chi.URLParam(r, "lectureId"))
		} else {
			lecture, err = h.svc.Finish(r.Context(), actorFrom(r), chi.URLParam(r, "contestId"), chi.URLParam(r, "lectureId"))
		}
		if err != nil {
			writeError(w, r, err)
			return
		}
		httpserver.WriteJSON(w, r, http.StatusOK, lecture, nil)
	}
}

func (h *Handler) Activate() http.HandlerFunc { return h.transition(true) }
func (h *Handler) Finish() http.HandlerFunc   { return h.transition(false) }

func (h *Handler) AdminDelete(w http.ResponseWriter, r *http.Request) {
	err := h.svc.Delete(r.Context(), actorFrom(r), chi.URLParam(r, "contestId"), chi.URLParam(r, "lectureId"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, map[string]bool{"deleted": true}, nil)
}

func (h *Handler) Scan(w http.ResponseWriter, r *http.Request) {
	var input ScanInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, r, ErrValidation)
		return
	}
	result, err := h.svc.Scan(r.Context(), actorFrom(r), chi.URLParam(r, "contestId"), chi.URLParam(r, "lectureId"), input)
	if err != nil {
		writeError(w, r, err)
		return
	}
	status := http.StatusCreated
	if result.AlreadyChecked {
		status = http.StatusOK
	}
	httpserver.WriteJSON(w, r, status, result, nil)
}

func (h *Handler) AdminAttendance(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListAttendance(r.Context(), actorFrom(r), chi.URLParam(r, "contestId"), chi.URLParam(r, "lectureId"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, list, map[string]any{"count": len(list)})
}

func (h *Handler) ParticipantCode(w http.ResponseWriter, r *http.Request) {
	principal := eventparticipants.PrincipalFrom(r.Context())
	if principal == nil {
		writeError(w, r, ErrParticipantInactive)
		return
	}
	code, err := h.svc.IssueCode(r.Context(), principal.Event.ID, principal.Participant.ID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusCreated, code, nil)
}

func (h *Handler) ParticipantLectures(w http.ResponseWriter, r *http.Request) {
	principal := eventparticipants.PrincipalFrom(r.Context())
	if principal == nil {
		writeError(w, r, ErrParticipantInactive)
		return
	}
	list, err := h.svc.ParticipantLectures(r.Context(), principal.Event.ID, principal.Participant.ID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, list, map[string]any{"count": len(list)})
}

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpserver.WriteError(w, r, http.StatusNotFound, "LECTURE_NOT_FOUND", "Лекция не найдена", nil)
	case errors.Is(err, ErrForbidden):
		httpserver.WriteError(w, r, http.StatusForbidden, "FORBIDDEN", "Недостаточно прав", nil)
	case errors.Is(err, ErrValidation):
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Проверьте данные лекции или сканирования", nil)
	case errors.Is(err, ErrInvalidTransition):
		httpserver.WriteError(w, r, http.StatusConflict, "INVALID_LECTURE_TRANSITION", "Недопустимый статус лекции", nil)
	case errors.Is(err, ErrLectureHasAttendance):
		httpserver.WriteError(w, r, http.StatusConflict, "LECTURE_HAS_ATTENDANCE", "Лекцию с посещениями удалить нельзя", nil)
	case errors.Is(err, ErrAttendanceClosed):
		httpserver.WriteError(w, r, http.StatusConflict, "ATTENDANCE_CLOSED", "Регистрация посещения сейчас закрыта", nil)
	case errors.Is(err, ErrInvalidCode):
		httpserver.WriteError(w, r, http.StatusBadRequest, "QR_INVALID", "QR-код недействителен", nil)
	case errors.Is(err, ErrExpiredCode):
		httpserver.WriteError(w, r, http.StatusGone, "QR_EXPIRED", "QR-код истёк — обновите его", nil)
	case errors.Is(err, ErrReplayedCode):
		httpserver.WriteError(w, r, http.StatusConflict, "QR_REPLAYED", "QR-код уже был использован", nil)
	case errors.Is(err, ErrParticipantInactive):
		httpserver.WriteError(w, r, http.StatusConflict, "PARTICIPANT_UNAVAILABLE", "Участник или мероприятие недоступны", nil)
	case errors.Is(err, ErrWrongDirection):
		httpserver.WriteError(w, r, http.StatusConflict, "LECTURE_WRONG_DIRECTION",
			"Эта лекция не для направления участника", nil)
	default:
		httpserver.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка", nil)
	}
}
