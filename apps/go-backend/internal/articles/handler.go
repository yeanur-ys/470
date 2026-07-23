package articles

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/yeanur-ys/nextGENjournalism/apps/go-backend/internal/auth"
)

type Handler struct {
	DB    *pgxpool.Pool
	Redis *redis.Client
}

func NewHandler(db *pgxpool.Pool, rdb *redis.Client) *Handler {
	return &Handler{DB: db, Redis: rdb}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(context.Background(), `
		SELECT id, journalist_id, parent_article_id, title, body, signature,
		       readership_volume, verified_claims, self_corrected_claims, false_claims,
		       is_retracted, created_at
		FROM articles
		ORDER BY created_at DESC
		LIMIT 100
	`)
	if err != nil {
		http.Error(w, "failed to load articles", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[Article])
	if err != nil {
		http.Error(w, "failed to read articles", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// Mine implements the journalist dashboard's article list (FR-1: each
// journalist manages their own graph of articles).
func (h *Handler) Mine(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.FromContext(r.Context())

	rows, err := h.DB.Query(context.Background(), `
		SELECT id, journalist_id, parent_article_id, title, body, signature,
		       readership_volume, verified_claims, self_corrected_claims, false_claims,
		       is_retracted, created_at
		FROM articles
		WHERE journalist_id = $1
		ORDER BY created_at DESC
	`, claims.UserID)
	if err != nil {
		http.Error(w, "failed to load articles", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[Article])
	if err != nil {
		http.Error(w, "failed to read articles", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

type createArticleRequest struct {
	Title           string  `json:"title"`
	Body            string  `json:"body"`
	Signature       string  `json:"signature"`
	ParentArticleID *string `json:"parentArticleId,omitempty"`
}

// Create implements FR-3 (claim tagging happens client-side via #Claim, persisted
// separately through the claims endpoints) and FR-4 (Sequence Stitching via ParentArticleID).
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.FromContext(r.Context())

	var req createArticleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Title == "" || req.Body == "" || req.Signature == "" {
		http.Error(w, "title, body and signature are required", http.StatusBadRequest)
		return
	}

	var id string
	err := h.DB.QueryRow(context.Background(), `
		INSERT INTO articles (journalist_id, parent_article_id, title, body, signature)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, claims.UserID, req.ParentArticleID, req.Title, req.Body, req.Signature).Scan(&id)
	if err != nil {
		http.Error(w, "failed to create article", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"id": id})
}

type createAppealRequest struct {
	ArticleID     string  `json:"articleId"`
	StakedPercent float64 `json:"stakedPercent"`
	EvidenceText  string  `json:"evidenceText"`
	EvidenceURL   string  `json:"evidenceUrl"`
}

// CreateAppeal implements FR-5/F-14, the Proof of Evidence Appeals Protocol:
// a journalist disputes a ruling on their own article by submitting new
// primary evidence and staking a percentage of their rank score. FR-9's
// pulsing amber UI state is driven by appeals.status == 'active'.
//
// The stake is deducted immediately rather than merely recorded — an appeal
// that costs nothing to file isn't a stake, and F-14's whole premise is that
// the journalist has skin in the game.
func (h *Handler) CreateAppeal(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	claims, ok := auth.FromContext(r.Context())
	if !ok || claims == nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	var req createAppealRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ArticleID == "" || req.StakedPercent <= 0 {
		http.Error(w, "articleId and a positive stakedPercent are required", http.StatusBadRequest)
		return
	}
	if req.StakedPercent > 100 {
		http.Error(w, "stakedPercent must be between 0 and 100", http.StatusBadRequest)
		return
	}
	// F-14: "Proof of Evidence" — an appeal with no new evidence is just a
	// complaint. One of the two evidence fields must carry something.
	if strings.TrimSpace(req.EvidenceText) == "" && strings.TrimSpace(req.EvidenceURL) == "" {
		http.Error(w, "an appeal must include new evidence (evidenceText or evidenceUrl)", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		http.Error(w, "failed to create appeal", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(ctx)

	// Ownership check: without it any journalist could file appeals against
	// any article in the system, including a competitor's, and pin their node
	// into the "under dispute" state (FR-9) indefinitely.
	var ownerID string
	if err := tx.QueryRow(ctx, `SELECT journalist_id FROM articles WHERE id = $1`, req.ArticleID).Scan(&ownerID); err != nil {
		http.Error(w, "article not found", http.StatusNotFound)
		return
	}
	if ownerID != claims.UserID {
		http.Error(w, "you can only appeal rulings on your own articles", http.StatusForbidden)
		return
	}

	var rankScore float64
	if err := tx.QueryRow(ctx, `SELECT rank_score FROM users WHERE id = $1 FOR UPDATE`, claims.UserID).Scan(&rankScore); err != nil {
		http.Error(w, "failed to load your rank score", http.StatusInternalServerError)
		return
	}
	staked := rankScore * (req.StakedPercent / 100)

	// uniq_active_appeal_per_article turns a second concurrent appeal into a
	// unique-violation here rather than a duplicate amber node.
	if _, err := tx.Exec(ctx, `
		INSERT INTO appeals (article_id, journalist_id, staked_percent, evidence_text, evidence_url)
		VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''))
	`, req.ArticleID, claims.UserID, req.StakedPercent, req.EvidenceText, req.EvidenceURL); err != nil {
		http.Error(w, "you already have an active appeal on this article", http.StatusConflict)
		return
	}

	if _, err := tx.Exec(ctx,
		`UPDATE users SET rank_score = GREATEST(rank_score - $2, 0) WHERE id = $1`,
		claims.UserID, staked,
	); err != nil {
		http.Error(w, "failed to stake rank score", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		http.Error(w, "failed to create appeal", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]float64{"stakedRankScore": staked})
}
