package server

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/redis/go-redis/v9"

	"github.com/yeanur-ys/nextGENjournalism/apps/go-backend/internal/articles"
	"github.com/yeanur-ys/nextGENjournalism/apps/go-backend/internal/auditors"
	"github.com/yeanur-ys/nextGENjournalism/apps/go-backend/internal/auth"
	"github.com/yeanur-ys/nextGENjournalism/apps/go-backend/internal/claims"
	"github.com/yeanur-ys/nextGENjournalism/apps/go-backend/internal/compliance"
	"github.com/yeanur-ys/nextGENjournalism/apps/go-backend/internal/consensus"
	"github.com/yeanur-ys/nextGENjournalism/apps/go-backend/internal/graph"
	"github.com/yeanur-ys/nextGENjournalism/apps/go-backend/internal/leaderboard"
)

type Deps struct {
	DB               *pgxpool.Pool
	Redis            *redis.Client
	Neo4j            neo4j.DriverWithContext
	Tokens           *auth.TokenService
	CORSAllowOrigins string
}

func NewRouter(deps Deps) http.Handler {
	mux := http.NewServeMux()

	authHandler := auth.NewHandler(deps.DB, deps.Tokens)
	articlesHandler := articles.NewHandler(deps.DB, deps.Redis)
	claimsHandler := claims.NewHandler(deps.DB, deps.Redis)
	consensusHandler := consensus.NewHandler(deps.DB, deps.Redis)
	complianceHandler := compliance.NewHandler(deps.DB)
	leaderboardHandler := leaderboard.NewHandler(deps.Redis, deps.DB)
	graphHandler := graph.NewHandler(deps.Neo4j, deps.DB)
	auditorsHandler := auditors.NewHandler(deps.DB)

	// --- Public routes ---
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("POST /auth/login", authHandler.Login)
	mux.HandleFunc("POST /auth/signup", authHandler.Signup)
	mux.HandleFunc("GET /articles", articlesHandler.List) // readers: public lineage browsing
	mux.HandleFunc("GET /articles/{articleId}", articlesHandler.Get)
	mux.HandleFunc("POST /articles/{articleId}/read", articlesHandler.RecordRead)
	mux.HandleFunc("GET /journalists/{journalistId}/graph", graphHandler.JournalistGraph)
	mux.HandleFunc("GET /leaderboard", leaderboardHandler.Top)

	// --- Journalist routes (FR-3, FR-4, FR-5) ---
	mux.Handle("POST /articles", RequireRole("journalist")(http.HandlerFunc(articlesHandler.Create)))
	mux.Handle("GET /articles/mine", RequireRole("journalist")(http.HandlerFunc(articlesHandler.Mine)))
	mux.Handle("POST /articles/{articleId}/claims", RequireRole("journalist")(http.HandlerFunc(claimsHandler.Create)))
	mux.Handle("POST /claims/{claimId}/self-correct", RequireRole("journalist")(http.HandlerFunc(claimsHandler.SelfCorrect)))
	mux.Handle("POST /appeals", RequireRole("journalist")(http.HandlerFunc(articlesHandler.CreateAppeal)))

	// --- Auditor routes (FR-6, FR-7, FR-8; NFR-6 gates voting behind credential_verified) ---
	mux.Handle("GET /claims/pending", RequireRole("auditor")(http.HandlerFunc(claimsHandler.Pending)))
	mux.Handle("POST /claims/{claimId}/votes", RequireRole("auditor")(http.HandlerFunc(consensusHandler.Vote)))

	// --- Admin routes (FR-13, FR-14, FR-15, NFR-6) ---
	mux.Handle("POST /admin/articles/{articleId}/retract", RequireRole("admin")(http.HandlerFunc(complianceHandler.Retract)))
	mux.Handle("GET /admin/auditors/pending", RequireRole("admin")(http.HandlerFunc(auditorsHandler.Pending)))
	mux.Handle("POST /admin/auditors/{auditorId}/verify", RequireRole("admin")(http.HandlerFunc(auditorsHandler.Verify)))

	// CORS must wrap everything, outermost: a browser's preflight OPTIONS
	// request needs an answer before Authenticate or the mux ever see it.
	// Authenticate wraps the mux so protected handlers can read claims from
	// context; it does not itself reject unauthenticated requests, so the
	// public routes above stay open to everyone.
	return CORS(deps.CORSAllowOrigins)(Authenticate(deps.Tokens)(mux))
}
