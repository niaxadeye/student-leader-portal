package challenges

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/eazytech/student-leader-cabinet/internal/platform/httpserver"
)

func briefingFileJSON(f BriefingFile) map[string]any {
	return map[string]any{
		"file_id": f.FileID, "original_name": f.OriginalName,
		"size_bytes": f.SizeBytes, "mime_type": f.MimeType,
		"download_url": f.DownloadURL,
	}
}

func briefingFilesJSON(files []BriefingFile) []map[string]any {
	out := make([]map[string]any, 0, len(files))
	for _, f := range files {
		out = append(out, briefingFileJSON(f))
	}
	return out
}

func resolvedBriefingJSON(r *ResolvedBriefing) map[string]any {
	if r == nil {
		return nil
	}
	return map[string]any{
		"visible": r.Visible, "scheduled": r.Scheduled, "hidden": r.Hidden,
		"personalized": r.Personalized, "publish_at": r.PublishAt,
		"body_text": r.BodyText, "files": briefingFilesJSON(r.Files),
	}
}

func overrideJSON(o *BriefingOverride) map[string]any {
	if o == nil {
		return nil
	}
	return map[string]any{
		"custom_text": o.CustomText, "body_text": o.BodyText,
		"custom_publish": o.CustomPublish, "publish_at": o.PublishAt,
		"hidden": o.Hidden, "replace_files": o.ReplaceFiles,
		"files": briefingFilesJSON(o.Files),
	}
}

func adminBriefingJSON(def *Briefing, people []BriefingContestant) map[string]any {
	contestants := make([]map[string]any, 0, len(people))
	for _, p := range people {
		contestants = append(contestants, map[string]any{
			"user_id": p.UserID, "login": p.Login, "full_name": p.FullName,
			"organization": p.Organization, "visible": p.Visible,
			"publish_at": p.PublishAt, "personalized": p.Personalized,
			"override": overrideJSON(p.Override),
		})
	}
	return map[string]any{
		"body_text": def.BodyText, "publish_at": def.PublishAt,
		"updated_at": def.UpdatedAt, "files": briefingFilesJSON(def.Files),
		"contestants": contestants,
	}
}

func writeAdminBriefing(w http.ResponseWriter, r *http.Request, def *Briefing, people []BriefingContestant, err error) {
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, adminBriefingJSON(def, people), nil)
}

func (h *Handler) AdminGetBriefing(w http.ResponseWriter, r *http.Request) {
	def, people, err := h.svc.AdminGetBriefing(r.Context(), actorOf(r), chi.URLParam(r, "challengeId"))
	writeAdminBriefing(w, r, def, people, err)
}

type briefingReq struct {
	BodyText  string     `json:"body_text"`
	PublishAt *time.Time `json:"publish_at"`
}

func (h *Handler) AdminSaveBriefing(w http.ResponseWriter, r *http.Request) {
	var req briefingReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", nil)
		return
	}
	def, people, err := h.svc.AdminSaveBriefing(r.Context(), actorOf(r), chi.URLParam(r, "challengeId"), BriefingInput{
		BodyText: req.BodyText, PublishAt: req.PublishAt,
	})
	writeAdminBriefing(w, r, def, people, err)
}

type overrideReq struct {
	CustomText    bool       `json:"custom_text"`
	BodyText      string     `json:"body_text"`
	CustomPublish bool       `json:"custom_publish"`
	PublishAt     *time.Time `json:"publish_at"`
	Hidden        bool       `json:"hidden"`
	ReplaceFiles  bool       `json:"replace_files"`
}

func (h *Handler) AdminSaveOverride(w http.ResponseWriter, r *http.Request) {
	var req overrideReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", nil)
		return
	}
	def, people, err := h.svc.AdminSaveOverride(
		r.Context(), actorOf(r),
		chi.URLParam(r, "challengeId"), chi.URLParam(r, "userId"),
		OverrideInput{
			CustomText: req.CustomText, BodyText: req.BodyText,
			CustomPublish: req.CustomPublish, PublishAt: req.PublishAt,
			Hidden: req.Hidden, ReplaceFiles: req.ReplaceFiles,
		},
	)
	writeAdminBriefing(w, r, def, people, err)
}

func (h *Handler) AdminClearOverride(w http.ResponseWriter, r *http.Request) {
	def, people, err := h.svc.AdminClearOverride(
		r.Context(), actorOf(r),
		chi.URLParam(r, "challengeId"), chi.URLParam(r, "userId"),
	)
	writeAdminBriefing(w, r, def, people, err)
}

func (h *Handler) AdminUploadBriefingFile(w http.ResponseWriter, r *http.Request) {
	h.uploadBriefingFile(w, r, "")
}

func (h *Handler) AdminUploadOverrideFile(w http.ResponseWriter, r *http.Request) {
	h.uploadBriefingFile(w, r, chi.URLParam(r, "userId"))
}

func (h *Handler) uploadBriefingFile(w http.ResponseWriter, r *http.Request, contestantID string) {
	r.Body = http.MaxBytesReader(w, r.Body, h.maxUploadBytes)
	if err := r.ParseMultipartForm(h.maxUploadBytes); err != nil {
		httpserver.WriteError(w, r, http.StatusRequestEntityTooLarge, "VALIDATION_ERROR", "Файл превышает допустимый размер", nil)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Файл не передан", nil)
		return
	}
	defer file.Close()
	suffix := httpserver.RequestIDFrom(r.Context())
	if suffix == "" {
		suffix = uuid.NewString()
	} else {
		suffix += "-" + strconv.FormatInt(header.Size, 10)
	}
	def, people, err := h.svc.UploadBriefingFile(r.Context(), actorOf(r), chi.URLParam(r, "challengeId"), contestantID, BriefingUpload{
		OriginalName: header.Filename,
		ContentType:  header.Header.Get("Content-Type"),
		Size:         header.Size,
		Reader:       file,
		KeySuffix:    suffix,
	})
	writeAdminBriefing(w, r, def, people, err)
}

func (h *Handler) AdminDeleteBriefingFile(w http.ResponseWriter, r *http.Request) {
	def, people, err := h.svc.DeleteBriefingFile(
		r.Context(), actorOf(r),
		chi.URLParam(r, "challengeId"), chi.URLParam(r, "fileId"),
	)
	writeAdminBriefing(w, r, def, people, err)
}

func (h *Handler) ContestantGetBriefing(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.ContestantBriefing(r.Context(), actorOf(r), chi.URLParam(r, "challengeId"))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, resolvedBriefingJSON(&res), nil)
}

func (h *Handler) JuryGetBriefing(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.JuryBriefing(r.Context(), actorOf(r), chi.URLParam(r, "challengeId"))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, resolvedBriefingJSON(res), nil)
}
