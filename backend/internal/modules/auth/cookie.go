package auth

import (
	"net"
	"net/http"
	"strings"
	"time"
)

// CookieConfig — параметры refresh-cookie (SITE.md §16: HttpOnly, Secure, SameSite).
type CookieConfig struct {
	Name     string
	Domain   string
	Secure   bool
	SameSite http.SameSite
	Path     string
}

func (h *Handler) setRefreshCookie(w http.ResponseWriter, pair *TokenPair) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.cookie.Name,
		Value:    pair.RefreshToken,
		Path:     h.cookie.Path,
		Domain:   h.cookie.Domain,
		Expires:  pair.RefreshExp,
		HttpOnly: true,
		Secure:   h.cookie.Secure,
		SameSite: h.cookie.SameSite,
	})
}

func (h *Handler) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.cookie.Name,
		Value:    "",
		Path:     h.cookie.Path,
		Domain:   h.cookie.Domain,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cookie.Secure,
		SameSite: h.cookie.SameSite,
	})
}

func readRefreshCookie(r *http.Request, name string) string {
	c, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return c.Value
}

// clientIP извлекает исходный IP с учётом X-Forwarded-For (за nginx).
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i > 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
