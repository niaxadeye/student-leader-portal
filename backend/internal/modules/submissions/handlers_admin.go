package submissions

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/eazytech/student-leader-cabinet/internal/platform/httpserver"
)

// adminRowJSON — строка таблицы дирекции (SITE.md §7.6).
func adminRowJSON(a AdminRow) map[string]any {
	s := a.Submission
	return map[string]any{
		"id": s.ID, "contestant_user_id": s.ContestantUserID,
		"full_name": s.FullName, "login": s.Login, "organization": s.Organization,
		"status": s.Status, "version": s.Version,
		"current_revision_number": s.CurrentRevisionNumber,
		"last_saved_at":           s.LastSavedAt, "submitted_at": s.SubmittedAt,
		"last_resubmitted_at": s.LastResubmittedAt,
		"locked":              s.LockedAt != nil, "file_count": a.FileCount,
	}
}

// AdminList — GET /admin/challenges/{challengeId}/submissions?status=&limit=&offset=.
func (h *Handler) AdminList(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	rows, total, err := h.svc.AdminList(r.Context(), actorOf(r),
		chi.URLParam(r, "challengeId"), r.URL.Query().Get("status"), limit, offset)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, adminRowJSON(row))
	}
	httpserver.WriteJSON(w, r, http.StatusOK, out, map[string]any{"total": total})
}

// AdminGet — GET /admin/submissions/{submissionId}: ответы, файлы, история ревизий.
func (h *Handler) AdminGet(w http.ResponseWriter, r *http.Request) {
	sub, revs, err := h.svc.AdminGet(r.Context(), actorOf(r), chi.URLParam(r, "submissionId"))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	files := make([]map[string]any, 0, len(sub.Files))
	for _, f := range sub.Files {
		files = append(files, fileJSON(f))
	}
	revisions := make([]map[string]any, 0, len(revs))
	for _, rev := range revs {
		revisions = append(revisions, map[string]any{
			"id": rev.ID, "revision_number": rev.RevisionNumber, "action_type": rev.ActionType,
			"schema_version": rev.SchemaVersion, "checksum": rev.Checksum,
			"created_at": rev.CreatedAt, "answers": rev.AnswersSnapshot, "files": rev.FilesSnapshot,
		})
	}
	httpserver.WriteJSON(w, r, http.StatusOK, map[string]any{
		"id": sub.ID, "challenge_id": sub.ChallengeID,
		"contestant": map[string]any{
			"user_id": sub.ContestantUserID, "full_name": sub.FullName,
			"login": sub.Login, "organization": sub.Organization,
		},
		"status": sub.Status, "answers": sub.Answers,
		"schema_version": sub.SchemaVersion, "version": sub.Version,
		"current_revision_number": sub.CurrentRevisionNumber,
		"submitted_at":            sub.SubmittedAt, "last_resubmitted_at": sub.LastResubmittedAt,
		"last_saved_at": sub.LastSavedAt, "locked": sub.LockedAt != nil, "lock_reason": sub.LockReason,
		"files": files, "revisions": revisions,
	}, nil)
}

// DownloadFile — GET /admin/submissions/{submissionId}/files/{fileId}.
// Возвращает presigned-URL как JSON (эндпоинт за Bearer-авторизацией; сам presigned-URL
// авторизуется подписью и открывается браузером напрямую — SITE.md §7.6).
func (h *Handler) DownloadFile(w http.ResponseWriter, r *http.Request) {
	url, err := h.svc.PresignFile(r.Context(), actorOf(r),
		chi.URLParam(r, "submissionId"), chi.URLParam(r, "fileId"))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, map[string]string{"download_url": url}, nil)
}
