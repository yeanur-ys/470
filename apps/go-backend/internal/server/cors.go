package server

import (
	"net/http"
	"strings"
)

// CORS makes the API callable from a browser on a different origin (the
// frontend on :3010, the API on :8080 — two different origins as far as a
// browser is concerned). Without this, every fetch() from the frontend fails
// before it ever reaches a handler: a JSON POST triggers a preflight OPTIONS
// request first, and with no OPTIONS route registered the mux returned 405;
// even a request that did get through had no Access-Control-Allow-Origin
// header, so the browser would refuse to hand the response to JS at all.
// curl doesn't enforce any of this, which is why testing with curl alone
// didn't catch it.
//
// allowedOrigins is a comma-separated list (CORS_ALLOW_ORIGINS), or "*" to
// allow any origin. The request's Origin is only ever reflected back when it
// matches the allowlist.
func CORS(allowedOrigins string) func(http.Handler) http.Handler {
	allowAll := allowedOrigins == "*"
	origins := map[string]bool{}
	if !allowAll {
		for _, o := range strings.Split(allowedOrigins, ",") {
			if o = strings.TrimSpace(o); o != "" {
				origins[o] = true
			}
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && (allowAll || origins[origin]) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "600")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
