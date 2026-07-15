package submissions

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/eazytech/student-leader-cabinet/internal/platform/httpserver"
)

// maxUploadBytes — верхняя граница тела multipart (защита памяти); поле-лимит проверяется отдельно.
const maxUploadBytes = 64 << 20 // 64 MiB

// UploadFile — POST .../submission/files (multipart: field_id, file). Проксирует в MinIO.
func (h *Handler) UploadFile(w http.ResponseWriter, r *http.Request) {
	if h.store == nil {
		httpserver.WriteError(w, r, http.StatusServiceUnavailable, "STORAGE_UNAVAILABLE", "Хранилище файлов недоступно", nil)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Не удалось прочитать файл", nil)
		return
	}
	fieldID := r.FormValue("field_id")
	file, header, err := r.FormFile("file")
	if err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Файл не передан", nil)
		return
	}
	defer file.Close()

	// KeySuffix берём из request-id (уникален на запрос) — без Date/rand в модуле.
	suffix := httpserver.RequestIDFrom(r.Context())
	if suffix == "" {
		suffix = strconv.FormatInt(header.Size, 10)
	}
	out, err := h.svc.UploadFile(r.Context(), actorOf(r), UploadInput{
		ChallengeID:  chi.URLParam(r, "challengeId"),
		FieldID:      fieldID,
		OriginalName: header.Filename,
		ContentType:  header.Header.Get("Content-Type"),
		Size:         header.Size,
		Reader:       file,
		KeySuffix:    suffix,
	}, h.store)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusCreated, fileJSON(*out), nil)
}

// DeleteFile — DELETE .../submission/files/{fileId}.
func (h *Handler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	err := h.svc.DeleteFile(r.Context(), actorOf(r),
		chi.URLParam(r, "challengeId"), chi.URLParam(r, "fileId"), h.store)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, map[string]string{"status": "ok"}, nil)
}
