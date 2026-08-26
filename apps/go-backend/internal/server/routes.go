package server

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/redis/go-redis/v9"

	"github.com/yeanur-ys/nextGENjournalism/apps/go-backend/internal/auth"
	"github.com/yeanur-ys/nextGENjournalism/apps/go-backend/internal/controllers"
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

	authController := controllers.NewAuthController(deps.DB, deps.Tokens, deps.Redis)
	articlesController := controllers.NewArticlesController(deps.DB, deps.Redis)
	claimsController := controllers.NewClaimsController(deps.DB, deps.Redis)
	consensusController := controllers.NewConsensusController(deps.DB, deps.Redis)
	complianceController := controllers.NewComplianceController(deps.DB)
	leaderboardController := controllers.NewLeaderboardController(deps.Redis, deps.DB)
	graphController := controllers.NewGraphController(deps.Neo4j, deps.DB)
	auditorsController := controllers.NewAuditorsController(deps.DB)

	// --- Public routes ---
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("POST /auth/login", authController.Login)
	mux.HandleFunc("POST /auth/signup", authController.Signup)
	mux.HandleFunc("GET /articles", articlesController.List) // readers: public lineage browsing
	mux.HandleFunc("GET /articles/{articleId}", articlesController.Get)
	mux.HandleFunc("POST /articles/{articleId}/read", articlesController.RecordRead)
	mux.HandleFunc("GET /journalists/{journalistId}/graph", graphController.JournalistGraph)
	mux.HandleFunc("GET /graph", graphController.GlobalGraph) // platform-wide epistemic graph (NFR-11)
	mux.HandleFunc("GET /leaderboard", leaderboardController.Top)

	// --- Journalist routes (FR-3, FR-4, FR-5) ---
	mux.Handle("POST /articles", RequireRole("journalist")(http.HandlerFunc(articlesController.Create)))
	mux.Handle("GET /articles/mine", RequireRole("journalist")(http.HandlerFunc(articlesController.Mine)))
	mux.Handle("POST /articles/{articleId}/claims", RequireRole("journalist")(http.HandlerFunc(claimsController.Create)))
	mux.Handle("GET /claims/mine", RequireRole("journalist")(http.HandlerFunc(claimsController.Mine)))
	mux.Handle("POST /claims/{claimId}/self-correct", RequireRole("journalist")(http.HandlerFunc(claimsController.SelfCorrect)))
	mux.Handle("POST /appeals", RequireRole("journalist")(http.HandlerFunc(articlesController.CreateAppeal)))

	// --- Auditor routes (FR-6, FR-7, FR-8; NFR-6 gates voting behind credential_verified) ---
	mux.Handle("GET /auditor/me", RequireRole("auditor")(http.HandlerFunc(auditorsController.Me)))
	mux.Handle("GET /claims/pending", RequireRole("auditor")(http.HandlerFunc(claimsController.Pending)))
	mux.Handle("POST /claims/{claimId}/votes", RequireRole("auditor")(http.HandlerFunc(consensusController.Vote)))

	// --- Admin routes (FR-13, FR-14, FR-15, NFR-6) ---
	mux.Handle("POST /admin/articles/{articleId}/retract", RequireRole("admin")(http.HandlerFunc(complianceController.Retract)))
	mux.Handle("GET /admin/auditors/pending", RequireRole("admin")(http.HandlerFunc(auditorsController.Pending)))
	mux.Handle("POST /admin/auditors/{auditorId}/verify", RequireRole("admin")(http.HandlerFunc(auditorsController.Verify)))

	// CORS must wrap everything, outermost: a browser's preflight OPTIONS
	// request needs an answer before Authenticate or the mux ever see it.
	// Authenticate wraps the mux so protected handlers can read claims from
	// context; it does not itself reject unauthenticated requests, so the
	// public routes above stay open to everyone.
	return CORS(deps.CORSAllowOrigins)(Authenticate(deps.Tokens)(mux))
}
