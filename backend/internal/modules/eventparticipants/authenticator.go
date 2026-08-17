package eventparticipants

import (
	"context"
	"net/http"
)

type participantPrincipalKey struct{}

func withPrincipal(ctx context.Context, principal *Principal) context.Context {
	return context.WithValue(ctx, participantPrincipalKey{}, principal)
}

func PrincipalFrom(ctx context.Context) *Principal {
	principal, _ := ctx.Value(participantPrincipalKey{}).(*Principal)
	return principal
}

type Authenticator struct {
	svc        *Service
	cookieName string
}

func NewAuthenticator(svc *Service, cookieName string) *Authenticator {
	return &Authenticator{svc: svc, cookieName: cookieName}
}

func (a *Authenticator) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, err := a.svc.Authenticate(r.Context(), readCookie(r, a.cookieName))
		if err != nil {
			writeError(w, r, ErrSessionExpired)
			return
		}
		next.ServeHTTP(w, r.WithContext(withPrincipal(r.Context(), principal)))
	})
}
