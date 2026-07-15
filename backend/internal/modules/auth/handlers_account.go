package auth

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/eazytech/student-leader-cabinet/internal/middleware"
	"github.com/eazytech/student-leader-cabinet/internal/platform/httpserver"
)

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	p := middleware.PrincipalFrom(r.Context())
	u, roles, err := h.svc.Me(r.Context(), p.UserID)
	if err != nil {
		writeAuthError(w, r, err)
		return
	}
	codes := make([]string, 0, len(roles))
	for _, role := range roles {
		codes = append(codes, role.Code)
	}
	httpserver.WriteJSON(w, r, http.StatusOK, map[string]any{
		"id":                   u.ID,
		"login":                u.Login,
		"full_name":            u.FullName,
		"roles":                codes,
		"must_change_password": u.MustChangePassword,
	}, nil)
}

type changePasswordReq struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	p := middleware.PrincipalFrom(r.Context())
	var req changePasswordReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", nil)
		return
	}
	if err := h.svc.ChangePassword(r.Context(), p.UserID, req.OldPassword, req.NewPassword); err != nil {
		writeAuthError(w, r, err)
		return
	}
	// Сессии отозваны — сбрасываем cookie, клиент должен перелогиниться.
	h.clearRefreshCookie(w)
	httpserver.WriteJSON(w, r, http.StatusOK, map[string]string{"status": "ok"}, nil)
}

func (h *Handler) Sessions(w http.ResponseWriter, r *http.Request) {
	p := middleware.PrincipalFrom(r.Context())
	list, err := h.svc.Sessions(r.Context(), p.UserID, p.SessionID)
	if err != nil {
		writeAuthError(w, r, err)
		return
	}
	out := make([]map[string]any, 0, len(list))
	for _, s := range list {
		out = append(out, map[string]any{
			"id":           s.ID,
			"user_agent":   s.UserAgent,
			"last_used_at": s.LastUsedAt,
			"created_at":   s.CreatedAt,
			"current":      s.Current,
		})
	}
	httpserver.WriteJSON(w, r, http.StatusOK, out, nil)
}

func (h *Handler) RevokeSession(w http.ResponseWriter, r *http.Request) {
	p := middleware.PrincipalFrom(r.Context())
	sessionID := chi.URLParam(r, "sessionId")
	if err := h.svc.RevokeSession(r.Context(), p.UserID, sessionID); err != nil {
		writeAuthError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, map[string]string{"status": "ok"}, nil)
}
