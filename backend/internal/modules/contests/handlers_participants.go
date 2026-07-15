package contests

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

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
		out = append(out, map[string]any{
			"id": p.ID, "user_id": p.UserID, "type": p.ParticipantType,
			"login": p.Login, "full_name": p.FullName, "organization": p.Organization,
			"user_status": p.UserStatus, "joined_at": p.JoinedAt,
		})
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
	// Временный пароль отдаётся один раз — админ передаёт его конкурсанту.
	httpserver.WriteJSON(w, r, http.StatusCreated, map[string]any{
		"user_id": res.UserID, "login": res.Login, "temp_password": res.TempPassword,
	}, nil)
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
