package server

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yeanur-ys/nextGENjournalism/apps/go-backend/internal/articles"
	"github.com/yeanur-ys/nextGENjournalism/apps/go-backend/internal/auth"
	"github.com/yeanur-ys/nextGENjournalism/apps/go-backend/internal/compliance"
	"github.com/yeanur-ys/nextGENjournalism/apps/go-backend/internal/consensus"
)

func NewRouter(db *pgxpool.Pool, tokens *auth.TokenService) http.Handler {
	mux := http.NewServeMux()

	authHandler := auth.NewHandler(db, tokens)
	articlesHandler := articles.NewHandler(db)
	consensusHandler := consensus.NewHandler(db)
	complianceHandler := compliance.NewHandler(db)

	// --- Public routes ---
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("POST /auth/login", authHandler.Login)
	mux.HandleFunc("GET /articles", articlesHandler.List) // readers: public lineage browsing

	// --- Journalist routes (FR-3, FR-4, FR-5) ---
	mux.Handle("POST /articles", RequireRole("journalist")(http.HandlerFunc(articlesHandler.Create)))
	mux.Handle("POST /appeals", RequireRole("journalist")(http.HandlerFunc(articlesHandler.CreateAppeal)))

	// --- Auditor routes (FR-6, FR-7, FR-8) ---
	mux.Handle("POST /claims/{claimId}/votes", RequireRole("auditor")(http.HandlerFunc(consensusHandler.Vote)))

	// --- Admin routes (FR-13, FR-14, FR-15) ---
	mux.Handle("POST /admin/articles/{articleId}/retract", RequireRole("admin")(http.HandlerFunc(complianceHandler.Retract)))

	// Authenticate wraps everything so protected handlers above can read claims
	// from context; it does not itself reject unauthenticated requests, so the
	// public routes above stay open to everyone.
	return Authenticate(tokens)(mux)
}
