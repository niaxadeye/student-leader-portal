// Package middleware — сквозные HTTP-middleware (SITE.md §16).
package middleware

import "context"

// Principal — аутентифицированный субъект запроса.
type Principal struct {
	UserID    string
	Role      string
	SessionID string
}

type principalKey struct{}

func withPrincipal(ctx context.Context, p *Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

// PrincipalFrom извлекает принципала из контекста (nil, если не аутентифицирован).
func PrincipalFrom(ctx context.Context) *Principal {
	p, _ := ctx.Value(principalKey{}).(*Principal)
	return p
}
