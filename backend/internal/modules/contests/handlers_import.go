package contests

import (
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/eazytech/student-leader-cabinet/internal/platform/httpserver"
)

const maxImportBytes = 1 << 20 // 1 MiB — защита от гигантских тел (скелет).

// ImportContestants принимает CSV в теле (text/csv или text/plain).
func (h *Handler) ImportContestants(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxImportBytes))
	if err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Не удалось прочитать тело", nil)
		return
	}
	res, err := h.svc.ImportContestants(r.Context(), actorOf(r), chi.URLParam(r, "contestId"), string(body))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, res, map[string]any{
		"created": res.Created, "failed": res.Failed,
	})
}

// ExportContestants отдаёт CSV как файл-вложение.
func (h *Handler) ExportContestants(w http.ResponseWriter, r *http.Request) {
	csv, err := h.svc.ExportContestants(r.Context(), actorOf(r), chi.URLParam(r, "contestId"))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="contestants.csv"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(csv))
}
