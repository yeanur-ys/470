package articles

import (
	"context"
	"encoding/json"
	"net/http"

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
	ArticleID      string  `json:"articleId"`
	StakedPercent  float64 `json:"stakedPercent"`
}

// CreateAppeal implements FR-5: journalists dispute a ruling by staking a
// percentage of their own rank score; FR-9 (pulsing amber UI state) is a
// frontend concern driven by appeals.status == 'active'.
func (h *Handler) CreateAppeal(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.FromContext(r.Context())

	var req createAppealRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ArticleID == "" || req.StakedPercent <= 0 {
		http.Error(w, "articleId and a positive stakedPercent are required", http.StatusBadRequest)
		return
	}

	_, err := h.DB.Exec(context.Background(), `
		INSERT INTO appeals (article_id, journalist_id, staked_percent)
		VALUES ($1, $2, $3)
	`, req.ArticleID, claims.UserID, req.StakedPercent)
	if err != nil {
		http.Error(w, "failed to create appeal", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
