package eventtasks

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/eazytech/student-leader-cabinet/internal/middleware"
	"github.com/eazytech/student-leader-cabinet/internal/modules/eventparticipants"
	"github.com/eazytech/student-leader-cabinet/internal/platform/httpserver"
)

type Handler struct {
	svc            *Service
	store          FileStore
	maxUploadBytes int64
}

func NewHandler(service *Service, store FileStore, maxImageBytes int64) *Handler {
	if maxImageBytes <= 0 {
		maxImageBytes = 10 << 20
	}
	return &Handler{svc: service, store: store, maxUploadBytes: maxImageBytes*10 + (1 << 20)}
}

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
	task, err := h.svc.Get(r.Context(), actorFrom(r), chi.URLParam(r, "contestId"), chi.URLParam(r, "taskId"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, task, nil)
}

func (h *Handler) AdminCreate(w http.ResponseWriter, r *http.Request) {
	var input TaskInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, r, ErrValidation)
		return
	}
	task, err := h.svc.Create(r.Context(), actorFrom(r), chi.URLParam(r, "contestId"), input)
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusCreated, task, nil)
}

func (h *Handler) AdminUpdate(w http.ResponseWriter, r *http.Request) {
	var input TaskInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, r, ErrValidation)
		return
	}
	task, err := h.svc.Update(r.Context(), actorFrom(r), chi.URLParam(r, "contestId"), chi.URLParam(r, "taskId"), input)
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, task, nil)
}

func (h *Handler) AdminTransition(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		task, err := h.svc.Transition(r.Context(), actorFrom(r), chi.URLParam(r, "contestId"), chi.URLParam(r, "taskId"), action)
		if err != nil {
			writeError(w, r, err)
			return
		}
		httpserver.WriteJSON(w, r, http.StatusOK, task, nil)
	}
}

func (h *Handler) AdminDelete(w http.ResponseWriter, r *http.Request) {
	err := h.svc.Delete(r.Context(), actorFrom(r), chi.URLParam(r, "contestId"), chi.URLParam(r, "taskId"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, map[string]bool{"deleted": true}, nil)
}

func (h *Handler) AdminSetImage(w http.ResponseWriter, r *http.Request) {
	h.setTaskFile(w, r, true)
}

func (h *Handler) AdminDeleteImage(w http.ResponseWriter, r *http.Request) {
	task, err := h.svc.DeleteImage(r.Context(), actorFrom(r), chi.URLParam(r, "contestId"), chi.URLParam(r, "taskId"), h.store)
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, task, nil)
}

func (h *Handler) AdminSetIcon(w http.ResponseWriter, r *http.Request) {
	h.setTaskFile(w, r, false)
}

func (h *Handler) AdminDeleteIcon(w http.ResponseWriter, r *http.Request) {
	task, err := h.svc.DeleteIcon(r.Context(), actorFrom(r), chi.URLParam(r, "contestId"), chi.URLParam(r, "taskId"), h.store)
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, task, nil)
}

func (h *Handler) setTaskFile(w http.ResponseWriter, r *http.Request, cover bool) {
	r.Body = http.MaxBytesReader(w, r.Body, h.maxUploadBytes)
	if err := r.ParseMultipartForm(h.maxUploadBytes); err != nil {
		writeError(w, r, ErrValidation)
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	file, header, err := r.FormFile("image")
	if err != nil {
		writeError(w, r, ErrValidation)
		return
	}
	defer file.Close()
	upload := ImageUpload{
		OriginalName: header.Filename, ContentType: header.Header.Get("Content-Type"),
		Size: header.Size, Reader: file, KeySuffix: uuid.NewString(),
	}
	contestID, taskID := chi.URLParam(r, "contestId"), chi.URLParam(r, "taskId")
	var task *Task
	if cover {
		task, err = h.svc.SetImage(r.Context(), actorFrom(r), contestID, taskID, upload, h.store)
	} else {
		task, err = h.svc.SetIcon(r.Context(), actorFrom(r), contestID, taskID, upload, h.store)
	}
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, task, nil)
}

func (h *Handler) ParticipantList(w http.ResponseWriter, r *http.Request) {
	principal := eventparticipants.PrincipalFrom(r.Context())
	if principal == nil {
		writeError(w, r, ErrForbidden)
		return
	}
	list, err := h.svc.ParticipantList(r.Context(), principal.Event.ID, principal.Participant.ID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, list, map[string]any{"count": len(list)})
}

func (h *Handler) ParticipantGet(w http.ResponseWriter, r *http.Request) {
	principal := eventparticipants.PrincipalFrom(r.Context())
	if principal == nil {
		writeError(w, r, ErrForbidden)
		return
	}
	task, err := h.svc.ParticipantGet(r.Context(), principal.Event.ID, chi.URLParam(r, "taskId"), principal.Participant.ID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, task, nil)
}

func (h *Handler) ParticipantSubmit(w http.ResponseWriter, r *http.Request) {
	principal := eventparticipants.PrincipalFrom(r.Context())
	if principal == nil {
		writeError(w, r, ErrForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, h.maxUploadBytes)
	if err := r.ParseMultipartForm(h.maxUploadBytes); err != nil {
		writeError(w, r, ErrValidation)
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	if r.MultipartForm == nil {
		writeError(w, r, ErrValidation)
		return
	}
	links := append([]string{}, r.MultipartForm.Value["links"]...)
	links = append(links, r.MultipartForm.Value["link"]...)
	images := make([]ImageUpload, 0)
	for _, header := range r.MultipartForm.File["images"] {
		file, err := header.Open()
		if err != nil {
			writeError(w, r, ErrValidation)
			return
		}
		defer file.Close()
		images = append(images, ImageUpload{
			OriginalName: header.Filename, ContentType: header.Header.Get("Content-Type"),
			Size: header.Size, Reader: file, KeySuffix: uuid.NewString(),
		})
	}
	submission, err := h.svc.Submit(r.Context(), principal.Event.ID, principal.Participant.ID,
		chi.URLParam(r, "taskId"), SubmitInput{
			ParticipantComment: r.FormValue("participant_comment"), Links: links, Images: images,
		}, h.store)
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusCreated, submission, nil)
}

func (h *Handler) ParticipantAsset(w http.ResponseWriter, r *http.Request) {
	principal := eventparticipants.PrincipalFrom(r.Context())
	if principal == nil {
		writeError(w, r, ErrForbidden)
		return
	}
	value, err := h.svc.ParticipantAssetURL(r.Context(), principal.Participant.ID, chi.URLParam(r, "assetId"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, map[string]string{"download_url": value}, nil)
}

func (h *Handler) ModerationList(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ModerationList(r.Context(), actorFrom(r), chi.URLParam(r, "contestId"), r.URL.Query().Get("status"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, list, map[string]any{"count": len(list)})
}

func (h *Handler) ModerationGet(w http.ResponseWriter, r *http.Request) {
	submission, err := h.svc.ModerationGet(r.Context(), actorFrom(r), chi.URLParam(r, "contestId"), chi.URLParam(r, "submissionId"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, submission, nil)
}

func (h *Handler) moderate(approve bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input ModerationInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, r, ErrValidation)
			return
		}
		var result *ModerationResult
		var err error
		if approve {
			result, err = h.svc.Approve(r.Context(), actorFrom(r), chi.URLParam(r, "contestId"), chi.URLParam(r, "submissionId"), input)
		} else {
			result, err = h.svc.Reject(r.Context(), actorFrom(r), chi.URLParam(r, "contestId"), chi.URLParam(r, "submissionId"), input)
		}
		if err != nil {
			writeError(w, r, err)
			return
		}
		status := http.StatusOK
		if !result.Replayed {
			status = http.StatusCreated
		}
		httpserver.WriteJSON(w, r, status, result, nil)
	}
}

func (h *Handler) Approve() http.HandlerFunc { return h.moderate(true) }
func (h *Handler) Reject() http.HandlerFunc  { return h.moderate(false) }

func (h *Handler) AdminAsset(w http.ResponseWriter, r *http.Request) {
	value, err := h.svc.AdminAssetURL(r.Context(), actorFrom(r), chi.URLParam(r, "contestId"),
		chi.URLParam(r, "submissionId"), chi.URLParam(r, "assetId"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, map[string]string{"download_url": value}, nil)
}

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpserver.WriteError(w, r, http.StatusNotFound, "TASK_NOT_FOUND", "Задание или отправка не найдены", nil)
	case errors.Is(err, ErrForbidden):
		httpserver.WriteError(w, r, http.StatusForbidden, "FORBIDDEN", "Недостаточно прав", nil)
	case errors.Is(err, ErrValidation):
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Проверьте поля, ссылки и изображения", nil)
	case errors.Is(err, ErrInvalidTransition):
		httpserver.WriteError(w, r, http.StatusConflict, "INVALID_TASK_TRANSITION", "Операция недоступна в текущем статусе", nil)
	case errors.Is(err, ErrSubmissionClosed):
		httpserver.WriteError(w, r, http.StatusConflict, "TASK_SUBMISSION_CLOSED", "Приём подтверждений сейчас закрыт", nil)
	case errors.Is(err, ErrStorageDisabled):
		httpserver.WriteError(w, r, http.StatusServiceUnavailable, "STORAGE_UNAVAILABLE", "Хранилище изображений временно недоступно", nil)
	case errors.Is(err, http.ErrMissingFile):
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Изображение не передано", nil)
	default:
		httpserver.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка", nil)
	}
}
