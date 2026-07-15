package auth

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/eazytech/student-leader-cabinet/internal/middleware"
	"github.com/eazytech/student-leader-cabinet/internal/platform/httpserver"
)

type Handler struct {
	svc    *Service
	cookie CookieConfig
}

func NewHandler(svc *Service, cookie CookieConfig) *Handler {
	return &Handler{svc: svc, cookie: cookie}
}

type loginReq struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", nil)
		return
	}
	pair, user, err := h.svc.Login(r.Context(), LoginInput{
		Login: req.Login, Password: req.Password,
		UserAgent: r.UserAgent(), IP: clientIP(r),
	})
	if err != nil {
		writeAuthError(w, r, err)
		return
	}
	h.setRefreshCookie(w, pair)
	httpserver.WriteJSON(w, r, http.StatusOK, map[string]any{
		"access_token":         pair.AccessToken,
		"expires_at":           pair.AccessExp,
		"must_change_password": user.MustChangePassword,
	}, nil)
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	rt := readRefreshCookie(r, h.cookie.Name)
	if rt == "" {
		httpserver.WriteError(w, r, http.StatusUnauthorized, "AUTH_SESSION_EXPIRED", "Нет refresh-токена", nil)
		return
	}
	pair, err := h.svc.Refresh(r.Context(), rt, r.UserAgent(), clientIP(r))
	if err != nil {
		h.clearRefreshCookie(w)
		writeAuthError(w, r, err)
		return
	}
	h.setRefreshCookie(w, pair)
	httpserver.WriteJSON(w, r, http.StatusOK, map[string]any{
		"access_token": pair.AccessToken,
		"expires_at":   pair.AccessExp,
	}, nil)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	p := middleware.PrincipalFrom(r.Context())
	_ = h.svc.Logout(r.Context(), p.UserID, p.SessionID)
	h.clearRefreshCookie(w)
	httpserver.WriteJSON(w, r, http.StatusOK, map[string]string{"status": "ok"}, nil)
}

func (h *Handler) LogoutAll(w http.ResponseWriter, r *http.Request) {
	p := middleware.PrincipalFrom(r.Context())
	_ = h.svc.LogoutAll(r.Context(), p.UserID)
	h.clearRefreshCookie(w)
	httpserver.WriteJSON(w, r, http.StatusOK, map[string]string{"status": "ok"}, nil)
}

func writeAuthError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrInvalidCredentials):
		httpserver.WriteError(w, r, http.StatusUnauthorized, "AUTH_INVALID_CREDENTIALS", "Неверный логин или пароль", nil)
	case errors.Is(err, ErrAccountBlocked):
		httpserver.WriteError(w, r, http.StatusForbidden, "AUTH_ACCOUNT_BLOCKED", "Учётная запись заблокирована", nil)
	case errors.Is(err, ErrAccountLocked):
		httpserver.WriteError(w, r, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "Слишком много попыток, попробуйте позже", nil)
	case errors.Is(err, ErrRefreshReused):
		httpserver.WriteError(w, r, http.StatusUnauthorized, "AUTH_REFRESH_REUSED", "Сессия скомпрометирована, войдите заново", nil)
	case errors.Is(err, ErrSessionExpired):
		httpserver.WriteError(w, r, http.StatusUnauthorized, "AUTH_SESSION_EXPIRED", "Сессия истекла", nil)
	case errors.Is(err, ErrWrongOldPassword):
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Неверный текущий пароль", nil)
	case errors.Is(err, ErrPasswordTooShort):
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Пароль слишком короткий (мин. 10)", nil)
	default:
		httpserver.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка", nil)
	}
}
