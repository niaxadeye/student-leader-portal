package contests

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/eazytech/student-leader-cabinet/internal/platform/httpserver"
)

func (h *Handler) ListParticipants(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.Participants(r.Context(), actorOf(r), chi.URLParam(r, "contestId"))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	out := make([]map[string]any, 0, len(list))
	for _, p := range list {
		out = append(out, h.participantJSON(r.Context(), p))
	}
	httpserver.WriteJSON(w, r, http.StatusOK, out, map[string]any{"count": len(out)})
}

type addContestantReq struct {
	Login        string `json:"login"`
	FullName     string `json:"full_name"`
	Organization string `json:"organization"`
}

func (h *Handler) AddContestant(w http.ResponseWriter, r *http.Request) {
	var req addContestantReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", nil)
		return
	}
	res, err := h.svc.AddContestant(r.Context(), actorOf(r), chi.URLParam(r, "contestId"), AddContestantInput{
		Login: req.Login, FullName: req.FullName, Organization: req.Organization,
	})
	if err != nil {
		writeErr(w, r, err)
		return
	}
	status := http.StatusCreated
	body := map[string]any{"user_id": res.UserID, "login": res.Login, "created": res.Created}
	if res.Created {
		body["temp_password"] = res.TempPassword
	} else {
		status = http.StatusOK
	}
	httpserver.WriteJSON(w, r, status, body, nil)
}

func (h *Handler) RemoveContestant(w http.ResponseWriter, r *http.Request) {
	err := h.svc.RemoveContestant(r.Context(), actorOf(r),
		chi.URLParam(r, "contestId"), chi.URLParam(r, "userId"))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, map[string]string{"status": "ok"}, nil)
}

func (h *Handler) participantJSON(ctx context.Context, p Participant) map[string]any {
	return map[string]any{
		"id": p.ID, "user_id": p.UserID, "type": p.ParticipantType,
		"login": p.Login, "full_name": p.FullName, "organization": p.Organization,
		"user_status": p.UserStatus, "joined_at": p.JoinedAt,
		"avatar_url": h.avatarURL(ctx, p.AvatarKey),
	}
}

func (h *Handler) avatarURL(ctx context.Context, key *string) any {
	if key == nil || *key == "" || h.store == nil {
		return nil
	}
	u, err := h.store.PresignGet(ctx, *key)
	if err != nil {
		return nil
	}
	return u
}

func (h *Handler) SetContestantAvatar(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, h.maxUploadBytes)
	if err := r.ParseMultipartForm(h.maxUploadBytes); err != nil {
		writeErr(w, r, ErrValidation)
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	file, header, err := r.FormFile("image")
	if err != nil {
		writeErr(w, r, ErrValidation)
		return
	}
	defer file.Close()
	key, err := h.svc.SetAvatar(r.Context(), actorOf(r),
		chi.URLParam(r, "contestId"), chi.URLParam(r, "userId"),
		ImageUpload{
			OriginalName: header.Filename, ContentType: header.Header.Get("Content-Type"),
			Size: header.Size, Reader: file, KeySuffix: uuid.NewString(),
		}, h.store)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, map[string]any{"avatar_url": h.avatarURL(r.Context(), &key)}, nil)
}

func (h *Handler) DeleteContestantAvatar(w http.ResponseWriter, r *http.Request) {
	err := h.svc.DeleteAvatar(r.Context(), actorOf(r),
		chi.URLParam(r, "contestId"), chi.URLParam(r, "userId"), h.store)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, map[string]any{"avatar_url": nil}, nil)
}
