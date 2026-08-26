package controllers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/yeanur-ys/nextGENjournalism/apps/go-backend/internal/auth"
	"github.com/yeanur-ys/nextGENjournalism/apps/go-backend/internal/models"
)

type ClaimsController struct {
	DB    *pgxpool.Pool
	Redis *redis.Client
}

func NewClaimsController(db *pgxpool.Pool, rdb *redis.Client) *ClaimsController {
	return &ClaimsController{DB: db, Redis: rdb}
}

type createClaimRequest struct {
	Text string `json:"text"`
	Tag  string `json:"tag"`
}

// Create implements FR-3.
func (c *ClaimsController) Create(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	requester, ok := auth.FromContext(r.Context())
	if !ok || requester == nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	articleID := r.PathValue("articleId")

	var req createClaimRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Text == "" || req.Tag == "" {
		http.Error(w, "text and tag are required", http.StatusBadRequest)
		return
	}

	id, err := models.CreateClaim(r.Context(), c.DB, requester.UserID, articleID, req.Text, req.Tag)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrArticleNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		case errors.Is(err, models.ErrNotClaimArticleOwner):
			http.Error(w, err.Error(), http.StatusForbidden)
		case errors.Is(err, models.ErrClaimArticleRetracted):
			http.Error(w, err.Error(), http.StatusConflict)
		default:
			http.Error(w, "failed to create claim", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"id": id})
}

// Pending lists claims awaiting cross-tag consensus (FR-7), paginated via
// ?page= and ?pageSize= (default 20) so a large backlog doesn't hide newer
// claims outside the old hardcoded top-100.
func (c *ClaimsController) Pending(w http.ResponseWriter, r *http.Request) {
	page := models.ClampPendingClaimsPage(r.URL.Query().Get("page"))
	pageSize := models.ClampPendingClaimsPageSize(r.URL.Query().Get("pageSize"))

	result, err := models.PendingClaims(r.Context(), c.DB, page, pageSize)
	if err != nil {
		http.Error(w, "failed to load claims", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// Mine lists every claim tagged across the requesting journalist's own
// articles, so their dashboard can offer a self-correct action on any of them
// — not just a claim tagged earlier in the same publish session.
func (c *ClaimsController) Mine(w http.ResponseWriter, r *http.Request) {
	requester, _ := auth.FromContext(r.Context())

	result, err := models.ClaimsByJournalist(r.Context(), c.DB, requester.UserID)
	if err != nil {
		http.Error(w, "failed to load your claims", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

type selfCorrectRequest struct {
	Text string `json:"text"`
	Tag  string `json:"tag"`
}

// SelfCorrect lets a journalist mark their own claim self-corrected,
// optionally rewriting its text/tag in the same request -- a "correction" is
// fixing what the claim says, not just relabelling it. Both fields are
// optional: an empty body just flips the status, same as before.
func (c *ClaimsController) SelfCorrect(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	requester, _ := auth.FromContext(r.Context())
	claimID := r.PathValue("claimId")

	var req selfCorrectRequest
	_ = json.NewDecoder(r.Body).Decode(&req) // body is optional; a decode failure just means no edits

	err := models.SelfCorrectClaim(r.Context(), c.DB, c.Redis, requester.UserID, claimID, req.Text, req.Tag)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrClaimNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		case errors.Is(err, models.ErrNotClaimOwner):
			http.Error(w, err.Error(), http.StatusForbidden)
		case errors.Is(err, models.ErrClaimAlreadyResolved):
			http.Error(w, err.Error(), http.StatusConflict)
		default:
			http.Error(w, "failed to self-correct claim", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
