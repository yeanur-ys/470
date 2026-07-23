package claims

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/yeanur-ys/nextGENjournalism/apps/go-backend/internal/auth"
	"github.com/yeanur-ys/nextGENjournalism/apps/go-backend/internal/ranking"
	"github.com/yeanur-ys/nextGENjournalism/apps/go-backend/internal/redisstore"
)

type Handler struct {
	DB    *pgxpool.Pool
	Redis *redis.Client
}

func NewHandler(db *pgxpool.Pool, rdb *redis.Client) *Handler {
	return &Handler{DB: db, Redis: rdb}
}

type createClaimRequest struct {
	Text string `json:"text"`
	Tag  string `json:"tag"`
}

// Create implements FR-3: journalists encapsulate specific statements in
// #Claim tags at publish time (or afterwards) so auditors can vote on them.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
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

	ctx := context.Background()

	// Ownership check. The route is journalist-only, but "is a journalist" is
	// not "is THIS article's journalist" — without this, any signed-in
	// journalist could attach claims to a rival's article and drive their
	// Corruption Factor (FR-10) and Rank Score (FR-16) down by having those
	// claims voted false.
	var ownerID string
	var isRetracted bool
	if err := h.DB.QueryRow(ctx,
		`SELECT journalist_id, is_retracted FROM articles WHERE id = $1`, articleID,
	).Scan(&ownerID, &isRetracted); err != nil {
		http.Error(w, "article not found", http.StatusNotFound)
		return
	}
	if ownerID != requester.UserID {
		http.Error(w, "you can only tag claims on your own articles", http.StatusForbidden)
		return
	}
	if isRetracted {
		http.Error(w, "this article has been retracted", http.StatusConflict)
		return
	}

	var id string
	err := h.DB.QueryRow(ctx, `
		INSERT INTO claims (article_id, text, tag) VALUES ($1, $2, $3) RETURNING id
	`, articleID, req.Text, req.Tag).Scan(&id)
	if err != nil {
		http.Error(w, "failed to create claim", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"id": id})
}

// Pending lists claims awaiting cross-tag consensus (FR-7), for auditors to
// pick up and vote on.
func (h *Handler) Pending(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(context.Background(), `
		SELECT c.id, c.article_id, a.title AS article_title, c.text, c.tag
		FROM claims c
		JOIN articles a ON a.id = c.article_id
		WHERE c.status = 'pending'
		ORDER BY c.created_at ASC
		LIMIT 100
	`)
	if err != nil {
		http.Error(w, "failed to load claims", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[PendingClaim])
	if err != nil {
		http.Error(w, "failed to read claims", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// SelfCorrect lets a journalist mark their own claim as self-corrected before
// auditors resolve it. SRS formula (1) weighs self-correction (w2) higher
// than baseline verification (w1) to reward getting ahead of a mistake.
func (h *Handler) SelfCorrect(w http.ResponseWriter, r *http.Request) {
	requester, _ := auth.FromContext(r.Context())
	claimID := r.PathValue("claimId")
	ctx := context.Background()

	var journalistID, status string
	err := h.DB.QueryRow(ctx, `
		SELECT a.journalist_id, c.status
		FROM claims c
		JOIN articles a ON a.id = c.article_id
		WHERE c.id = $1
	`, claimID).Scan(&journalistID, &status)
	if err != nil {
		http.Error(w, "claim not found", http.StatusNotFound)
		return
	}
	if journalistID != requester.UserID {
		http.Error(w, "you can only self-correct your own claims", http.StatusForbidden)
		return
	}
	if status != "pending" {
		http.Error(w, "this claim has already been resolved", http.StatusConflict)
		return
	}

	if _, err := h.DB.Exec(ctx, `UPDATE claims SET status = 'self_corrected' WHERE id = $1`, claimID); err != nil {
		http.Error(w, "failed to self-correct claim", http.StatusInternalServerError)
		return
	}

	authorID, rankScore, err := ranking.BumpArticleCounterAndRecalculate(ctx, h.DB, claimID, "self_corrected_claims")
	if err != nil {
		http.Error(w, "failed to update rank score", http.StatusInternalServerError)
		return
	}

	if h.Redis != nil {
		if err := h.Redis.ZAdd(ctx, redisstore.LeaderboardKey, redis.Z{Score: rankScore, Member: authorID}).Err(); err != nil {
			http.Error(w, "failed to update leaderboard", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}
