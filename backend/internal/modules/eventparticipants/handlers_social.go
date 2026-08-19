package eventparticipants

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/eazytech/student-leader-cabinet/internal/platform/httpserver"
	"github.com/eazytech/student-leader-cabinet/internal/platform/security"
)

const oauthStateCookie = "slc_oauth_state"

func (h *Handler) LoginOptions(w http.ResponseWriter, r *http.Request) {
	options, err := h.svc.LoginOptions(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, options, nil)
}

func (h *Handler) VKStart(w http.ResponseWriter, r *http.Request) {
	target, state, err := h.svc.VKStartURL(chi.URLParam(r, "eventSlug"), time.Now())
	if err != nil {
		writeError(w, r, err)
		return
	}
	h.setOAuthCookie(w, state)
	http.Redirect(w, r, target, http.StatusFound)
}

func (h *Handler) VKCallback(w http.ResponseWriter, r *http.Request) {
	client := requestClientInfo(r)
	if !h.allowSocialLogin(r, client) {
		h.redirectSocialError(w, r, "", ErrRateLimited)
		return
	}
	state := r.URL.Query().Get("state")
	if state == "" {
		state = h.oauthState(r)
	}
	slug, _ := h.svc.parseOAuthState(state, "vk", time.Now())
	result, err := h.svc.LoginByVKCallback(r.Context(), r.URL.Query().Get("code"), state, client)
	if err != nil {
		h.redirectSocialError(w, r, slug, err)
		return
	}
	h.finishSocialOutcome(w, r, result)
}

type telegramLoginRequest struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
	PhotoURL  string `json:"photo_url"`
	AuthDate  int64  `json:"auth_date"`
	Hash      string `json:"hash"`
	EventSlug string `json:"event_slug"`
}

func (h *Handler) LoginByTelegram(w http.ResponseWriter, r *http.Request) {
	client := requestClientInfo(r)
	if !h.allowLogin(r, client) {
		writeError(w, r, ErrRateLimited)
		return
	}
	var req telegramLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, ErrValidation)
		return
	}
	slug := strings.TrimSpace(req.EventSlug)
	if slug == "" {
		slug = chi.URLParam(r, "eventSlug")
	}
	result, err := h.svc.LoginByTelegramValues(r.Context(), slug, telegramWidgetValues(req), client)
	if err != nil {
		writeError(w, r, err)
		return
	}
	h.writeSocialResult(w, r, result)
}

// telegramWidgetValues повторяет набор полей, по которому виджет считал подпись.
func telegramWidgetValues(req telegramLoginRequest) url.Values {
	values := url.Values{}
	values.Set("id", strconv.FormatInt(req.ID, 10))
	values.Set("auth_date", strconv.FormatInt(req.AuthDate, 10))
	values.Set("hash", req.Hash)
	for key, value := range map[string]string{
		"first_name": req.FirstName,
		"last_name":  req.LastName,
		"username":   req.Username,
		"photo_url":  req.PhotoURL,
	} {
		if value != "" {
			values.Set(key, value)
		}
	}
	return values
}

type telegramWebAppRequest struct {
	InitData  string `json:"init_data"`
	EventSlug string `json:"event_slug"`
}

func (h *Handler) LoginByTelegramWebApp(w http.ResponseWriter, r *http.Request) {
	client := requestClientInfo(r)
	if !h.allowLogin(r, client) {
		writeError(w, r, ErrRateLimited)
		return
	}
	var req telegramWebAppRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, ErrValidation)
		return
	}
	slug := strings.TrimSpace(req.EventSlug)
	if slug == "" {
		slug = chi.URLParam(r, "eventSlug")
	}
	result, err := h.svc.LoginByTelegramWebApp(r.Context(), slug, req.InitData, client)
	if err != nil {
		writeError(w, r, err)
		return
	}
	h.writeSocialResult(w, r, result)
}

type vkTokenRequest struct {
	AccessToken string `json:"access_token"`
	EventSlug   string `json:"event_slug"`
}

func (h *Handler) LoginByVKToken(w http.ResponseWriter, r *http.Request) {
	client := requestClientInfo(r)
	if !h.allowLogin(r, client) {
		writeError(w, r, ErrRateLimited)
		return
	}
	var req vkTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, ErrValidation)
		return
	}
	slug := strings.TrimSpace(req.EventSlug)
	if slug == "" {
		slug = chi.URLParam(r, "eventSlug")
	}
	result, err := h.svc.LoginByVKAccessToken(r.Context(), slug, req.AccessToken, client)
	if err != nil {
		writeError(w, r, err)
		return
	}
	h.writeSocialResult(w, r, result)
}

type continueSocialRequest struct {
	ContinueToken string `json:"continue_token"`
	EventSlug     string `json:"event_slug"`
}

func (h *Handler) ContinueSocialLogin(w http.ResponseWriter, r *http.Request) {
	client := requestClientInfo(r)
	if !h.allowLogin(r, client) {
		writeError(w, r, ErrRateLimited)
		return
	}
	var req continueSocialRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, ErrValidation)
		return
	}
	result, err := h.svc.ContinueSocialLogin(r.Context(), req.ContinueToken, req.EventSlug, client)
	if err != nil {
		writeError(w, r, err)
		return
	}
	h.writeSocialResult(w, r, result)
}

func (h *Handler) writeSocialResult(w http.ResponseWriter, r *http.Request, result *SocialAuthResult) {
	if result == nil {
		writeError(w, r, ErrInvalidCredentials)
		return
	}
	if result.Session != nil {
		h.writeLoginResult(w, r, result.Session)
		return
	}
	events := result.Events
	if events == nil {
		events = []PublicEvent{}
	}
	httpserver.WriteJSON(w, r, http.StatusOK, map[string]any{
		"status":         "choose_event",
		"events":         events,
		"continue_token": result.ContinueToken,
	}, nil)
}

func (h *Handler) finishSocialOutcome(w http.ResponseWriter, r *http.Request, result *SocialAuthResult) {
	if result != nil && result.Session != nil {
		h.finishSocialRedirect(w, r, result.Session)
		return
	}
	h.clearOAuthCookie(w)
	query := url.Values{}
	query.Set("as", "participant")
	if result != nil && result.ContinueToken != "" {
		query.Set("continue", result.ContinueToken)
	}
	http.Redirect(w, r, h.publicURL("/login?"+query.Encode()), http.StatusFound)
}

func (h *Handler) finishSocialRedirect(w http.ResponseWriter, r *http.Request, result *SessionResult) {
	h.clearOAuthCookie(w)
	http.SetCookie(w, &http.Cookie{
		Name: h.cookie.Name, Value: result.Token, Path: h.cookie.Path, Domain: h.cookie.Domain,
		Expires: result.ExpiresAt, HttpOnly: true, Secure: h.cookie.Secure, SameSite: h.cookie.SameSite,
	})
	http.Redirect(w, r, h.publicURL("/event/"+url.PathEscape(result.Event.Slug)+"/me"), http.StatusFound)
}

func (h *Handler) redirectSocialError(w http.ResponseWriter, r *http.Request, eventSlug string, err error) {
	h.clearOAuthCookie(w)
	query := url.Values{}
	query.Set("as", "participant")
	query.Set("error", socialErrorCode(err))
	if eventSlug != "" {
		query.Set("event", eventSlug)
	}
	http.Redirect(w, r, h.publicURL("/login?"+query.Encode()), http.StatusFound)
}

func socialErrorCode(err error) string {
	switch {
	case errors.Is(err, ErrRateLimited):
		return "rate"
	case errors.Is(err, ErrSocialUnavailable):
		return "unavailable"
	case errors.Is(err, ErrSocialNotLinked):
		return "unlinked"
	default:
		return "social"
	}
}

func (h *Handler) publicURL(path string) string {
	return strings.TrimRight(h.svc.social.PublicBaseURL, "/") + path
}

func (h *Handler) setOAuthCookie(w http.ResponseWriter, state string) {
	http.SetCookie(w, &http.Cookie{
		Name: oauthStateCookie, Value: state, Path: "/api/v1/participant-auth",
		Domain: h.cookie.Domain, MaxAge: 600, HttpOnly: true,
		Secure: h.cookie.Secure, SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) clearOAuthCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: oauthStateCookie, Value: "", Path: "/api/v1/participant-auth",
		Domain: h.cookie.Domain, MaxAge: -1, HttpOnly: true,
		Secure: h.cookie.Secure, SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) oauthState(r *http.Request) string {
	return readCookie(r, oauthStateCookie)
}

func (h *Handler) allowSocialLogin(r *http.Request, client ClientInfo) bool {
	if h.limiter == nil {
		return true
	}
	return h.limiter.Allow(security.HashToken("social|" + client.IP + "|" + client.UserAgent))
}
