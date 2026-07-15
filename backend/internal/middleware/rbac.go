package middleware

import (
	"net/http"

	"github.com/eazytech/student-leader-cabinet/internal/platform/httpserver"
)

// RequireRole пропускает только принципалов с одной из перечисленных ролей (SITE.md §6: п.4).
// Проверка scope конкурса (п.5) добавляется на Этапе 2 вместе с contest-модулем.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p := PrincipalFrom(r.Context())
			if p == nil {
				httpserver.WriteError(w, r, http.StatusUnauthorized, "AUTH_SESSION_EXPIRED", "Требуется авторизация", nil)
				return
			}
			// SUPER_ADMIN имеет полный доступ (SITE.md §5.1).
			if p.Role == "SUPER_ADMIN" || allowed[p.Role] {
				next.ServeHTTP(w, r)
				return
			}
			httpserver.WriteError(w, r, http.StatusForbidden, "FORBIDDEN", "Недостаточно прав", nil)
		})
	}
}
