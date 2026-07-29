package server

import (
	"net/http"
	"strings"

	"github.com/yeanur-ys/nextGENjournalism/apps/go-backend/internal/auth"
)

// Authenticate parses the bearer token (if present) and attaches the claims to
// the request context. It never rejects a request on its own — that's what
// RequireRole is for — so public routes can share the same middleware chain.
func Authenticate(tokens *auth.TokenService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			tokenString := strings.TrimPrefix(header, "Bearer ")
			if tokenString == "" {
				next.ServeHTTP(w, r)
				return
			}

			claims, err := tokens.Parse(tokenString)
			if err != nil {
				http.Error(w, "invalid or expired token", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r.WithContext(auth.WithClaims(r.Context(), claims)))
		})
	}
}

// RequireRole rejects the request unless the authenticated user holds one of
// the allowed roles. Unlike a single global guard, this is applied per route
// group so journalist/auditor/admin routes don't collide.
func RequireRole(allowed ...string) func(http.Handler) http.Handler {
	allowedSet := make(map[string]bool, len(allowed))
	for _, role := range allowed {
		allowedSet[role] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := auth.FromContext(r.Context())
			if !ok || claims == nil {
				http.Error(w, "authentication required", http.StatusUnauthorized)
				return
			}
			if !allowedSet[claims.Role] {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
