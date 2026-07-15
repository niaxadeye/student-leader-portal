package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/eazytech/student-leader-cabinet/internal/platform/httpserver"
	"github.com/eazytech/student-leader-cabinet/internal/platform/security"
)

// SessionChecker проверяет, что сессия активна (не отозвана, не истекла).
type SessionChecker interface {
	SessionActive(ctx context.Context, sessionID string) (bool, error)
}

// Authenticator валидирует access-токен и активность сессии, кладёт Principal в контекст.
type Authenticator struct {
	jwt      *security.JWTManager
	sessions SessionChecker
}

func NewAuthenticator(jwt *security.JWTManager, sessions SessionChecker) *Authenticator {
	return &Authenticator{jwt: jwt, sessions: sessions}
}

// Require — middleware, требующий валидный access-токен (SITE.md §6: п.1–3).
func (a *Authenticator) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearer(r)
		if token == "" {
			httpserver.WriteError(w, r, http.StatusUnauthorized, "AUTH_SESSION_EXPIRED", "Требуется авторизация", nil)
			return
		}
		claims, err := a.jwt.Parse(token)
		if err != nil {
			httpserver.WriteError(w, r, http.StatusUnauthorized, "AUTH_SESSION_EXPIRED", "Сессия недействительна", nil)
			return
		}
		// Проверка активности сессии в БД — access отзывается вместе с сессией.
		active, err := a.sessions.SessionActive(r.Context(), claims.SessionID)
		if err != nil || !active {
			httpserver.WriteError(w, r, http.StatusUnauthorized, "AUTH_SESSION_EXPIRED", "Сессия завершена", nil)
			return
		}
		p := &Principal{UserID: claims.Subject, Role: claims.Role, SessionID: claims.SessionID}
		next.ServeHTTP(w, r.WithContext(withPrincipal(r.Context(), p)))
	})
}

func bearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return ""
}
