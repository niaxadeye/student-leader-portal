package useradmin

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/eazytech/student-leader-cabinet/internal/platform/httpserver"
)

type roleReq struct {
	Role        string `json:"role"`
	ScopeType   string `json:"scope_type"`
	ScopeID     string `json:"scope_id"`
	AccessLevel string `json:"access_level"`
}

func (h *Handler) AssignRole(w http.ResponseWriter, r *http.Request) {
	var req roleReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", nil)
		return
	}
	err := h.svc.AssignRole(r.Context(), actorOf(r), chi.URLParam(r, "userId"),
		AssignRoleInput{Role: req.Role, ScopeType: req.ScopeType, ScopeID: req.ScopeID, AccessLevel: req.AccessLevel})
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, map[string]string{"status": "ok"}, nil)
}

// RemoveRole принимает role/scope из query (?role=ADMIN&scope_type=CONTEST&scope_id=...).
func (h *Handler) RemoveRole(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	err := h.svc.RemoveRole(r.Context(), actorOf(r), chi.URLParam(r, "userId"),
		AssignRoleInput{Role: q.Get("role"), ScopeType: q.Get("scope_type"), ScopeID: q.Get("scope_id")})
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, map[string]string{"status": "ok"}, nil)
}
