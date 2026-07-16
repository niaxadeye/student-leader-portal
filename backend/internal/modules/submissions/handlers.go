package submissions

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/eazytech/student-leader-cabinet/internal/middleware"
	"github.com/eazytech/student-leader-cabinet/internal/platform/httpserver"
)

type Handler struct {
	svc            *Service
	store          FileStore
	maxUploadBytes int64
}

// NewHandler создаёт хендлер модуля. maxFileSizeMB ограничивает тело multipart-запроса
// загрузки файла (SITE.md §29, DEFAULT_MAX_FILE_SIZE_MB); поле-лимит проверяется отдельно в Service.
func NewHandler(svc *Service, store FileStore, maxFileSizeMB int) *Handler {
	return &Handler{svc: svc, store: store, maxUploadBytes: int64(maxFileSizeMB) << 20}
}

func actorOf(r *http.Request) Actor {
	p := middleware.PrincipalFrom(r.Context())
	if p == nil {
		return Actor{}
	}
	return Actor{UserID: p.UserID, IsSuper: p.Role == "SUPER_ADMIN", IsMega: p.Role == "MEGA_ADMIN"}
}

func writeErr(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpserver.WriteError(w, r, http.StatusNotFound, "NOT_FOUND", "Работа не найдена", nil)
	case errors.Is(err, ErrForbidden):
		httpserver.WriteError(w, r, http.StatusForbidden, "FORBIDDEN", "Нет доступа", nil)
	case errors.Is(err, ErrLocked):
		httpserver.WriteError(w, r, http.StatusConflict, "SUBMISSION_LOCKED", "Работа заблокирована", nil)
	case errors.Is(err, ErrClosed):
		httpserver.WriteError(w, r, http.StatusConflict, "SUBMISSION_CLOSED", "Приём ответов закрыт", nil)
	case errors.Is(err, ErrDeadline):
		httpserver.WriteError(w, r, http.StatusConflict, "DEADLINE_PASSED", "Дедлайн истёк", nil)
	case errors.Is(err, ErrValidation):
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
	default:
		httpserver.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка", nil)
	}
}

// submissionJSON — представление работы для конкурсанта.
func submissionJSON(s *Submission) map[string]any {
	files := make([]map[string]any, 0, len(s.Files))
	for _, f := range s.Files {
		files = append(files, fileJSON(f))
	}
	return map[string]any{
		"id": s.ID, "challenge_id": s.ChallengeID, "status": s.Status,
		"answers": s.Answers, "schema_version": s.SchemaVersion, "version": s.Version,
		"current_revision_number": s.CurrentRevisionNumber,
		"first_opened_at":         s.FirstOpenedAt, "last_saved_at": s.LastSavedAt,
		"submitted_at": s.SubmittedAt, "last_resubmitted_at": s.LastResubmittedAt,
		"locked": s.LockedAt != nil, "lock_reason": s.LockReason,
		"files": files,
	}
}

func fileJSON(f SubmissionFile) map[string]any {
	return map[string]any{
		"file_id": f.FileID, "field_id": f.FieldID, "field_key": f.FieldKey,
		"original_name": f.OriginalName, "size_bytes": f.SizeBytes,
		"mime_type": f.MimeType, "download_url": f.DownloadURL,
	}
}

// GetOrCreate — GET /contestant/challenges/{challengeId}/submission.
func (h *Handler) GetOrCreate(w http.ResponseWriter, r *http.Request) {
	sub, err := h.svc.GetOrCreateDraft(r.Context(), actorOf(r), chi.URLParam(r, "challengeId"))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, submissionJSON(sub), nil)
}

type answersReq struct {
	Answers map[string]any `json:"answers"`
}

// SaveDraft — PUT .../submission/draft.
func (h *Handler) SaveDraft(w http.ResponseWriter, r *http.Request) {
	var req answersReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", nil)
		return
	}
	sub, err := h.svc.SaveDraft(r.Context(), actorOf(r), chi.URLParam(r, "challengeId"), req.Answers)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, submissionJSON(sub), nil)
}

// Submit — POST .../submission/submit.
func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	var req answersReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", nil)
		return
	}
	sub, err := h.svc.Submit(r.Context(), actorOf(r), chi.URLParam(r, "challengeId"), req.Answers)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, submissionJSON(sub), nil)
}
