package middleware

import (
	"net/http"
	"strings"

	"github.com/eazytech/student-leader-cabinet/internal/platform/httpserver"
)

// SecurityHeaders добавляет базовые защитные заголовки (SITE.md §16).
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

// CSRFOrigin проверяет заголовок Origin для небезопасных методов (SITE.md §16: CSRF).
// Cookie-эндпоинты (login/refresh/logout) защищены SameSite + проверкой Origin.
func CSRFOrigin(allowedOrigins ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[strings.TrimRight(o, "/")] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
				next.ServeHTTP(w, r)
				return
			}
			origin := strings.TrimRight(r.Header.Get("Origin"), "/")
			// Пустой Origin допустим для не-браузерных клиентов без cookie (тесты, CLI).
			if origin != "" && !allowed[origin] {
				httpserver.WriteError(w, r, http.StatusForbidden, "FORBIDDEN", "Недопустимый источник запроса", nil)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
