package useradmin

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/eazytech/student-leader-cabinet/internal/platform/httpserver"
)

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	res, err := h.svc.List(r.Context(), actorOf(r), ListFilter{
		Search: q.Get("search"), Role: q.Get("role"), Status: q.Get("status"),
		Limit: limit, Offset: offset,
	})
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, res.Users, map[string]any{
		"total": res.Total, "limit": res.Limit, "offset": res.Offset,
	})
}

func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	u, err := h.svc.Get(r.Context(), chi.URLParam(r, "userId"))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, u, nil)
}

type createUserReq struct {
	Login        string `json:"login"`
	FullName     string `json:"full_name"`
	Email        string `json:"email"`
	Organization string `json:"organization"`
	Role         string `json:"role"`
	ScopeType    string `json:"scope_type"`
	ScopeID      string `json:"scope_id"`
	AccessLevel  string `json:"access_level"`
	OrgName      string `json:"org_name"`
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", nil)
		return
	}
	res, err := h.svc.Create(r.Context(), actorOf(r), CreateInput{
		Login: req.Login, FullName: req.FullName, Email: req.Email, Organization: req.Organization,
		Role: req.Role, ScopeType: req.ScopeType, ScopeID: req.ScopeID, AccessLevel: req.AccessLevel,
		OrgName: req.OrgName,
	})
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusCreated, map[string]any{
		"user_id": res.UserID, "login": res.Login, "temp_password": res.TempPassword,
	}, nil)
}

type updateUserReq struct {
	FullName     string `json:"full_name"`
	Email        string `json:"email"`
	Organization string `json:"organization"`
}

func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	var req updateUserReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", nil)
		return
	}
	u, err := h.svc.Update(r.Context(), actorID(r), chi.URLParam(r, "userId"),
		req.FullName, req.Email, req.Organization)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, u, nil)
}
