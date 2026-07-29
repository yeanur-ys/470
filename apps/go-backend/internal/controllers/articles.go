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

type ArticlesController struct {
	DB    *pgxpool.Pool
	Redis *redis.Client
}

func NewArticlesController(db *pgxpool.Pool, rdb *redis.Client) *ArticlesController {
	return &ArticlesController{DB: db, Redis: rdb}
}

// List is the public reader listing: latest 100 stories.
func (c *ArticlesController) List(w http.ResponseWriter, r *http.Request) {
	result, err := models.ListArticles(r.Context(), c.DB)
	if err != nil {
		http.Error(w, "failed to load articles", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// Mine implements the journalist dashboard's article list (FR-1).
func (c *ArticlesController) Mine(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.FromContext(r.Context())

	result, err := models.ArticlesByJournalist(r.Context(), c.DB, claims.UserID)
	if err != nil {
		http.Error(w, "failed to load articles", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// Get is the public reading endpoint: no account required.
func (c *ArticlesController) Get(w http.ResponseWriter, r *http.Request) {
	articleID := r.PathValue("articleId")

	detail, err := models.GetArticleDetail(r.Context(), c.DB, articleID)
	if err != nil {
		if errors.Is(err, models.ErrArticleNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, "failed to load article", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(detail)
}

type createArticleRequest struct {
	Title           string  `json:"title"`
	Body            string  `json:"body"`
	Signature       string  `json:"signature"`
	ParentArticleID *string `json:"parentArticleId,omitempty"`
}

// Create implements FR-3/FR-4.
func (c *ArticlesController) Create(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.FromContext(r.Context())

	var req createArticleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Title == "" || req.Body == "" || req.Signature == "" {
		http.Error(w, "title, body and signature are required", http.StatusBadRequest)
		return
	}

	id, err := models.CreateArticle(r.Context(), c.DB, claims.UserID, models.NewArticleInput{
		Title: req.Title, Body: req.Body, Signature: req.Signature, ParentArticleID: req.ParentArticleID,
	})
	if err != nil {
		http.Error(w, "failed to create article", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"id": id})
}

// RecordRead implements the readership-volume side of FR-12.
func (c *ArticlesController) RecordRead(w http.ResponseWriter, r *http.Request) {
	articleID := r.PathValue("articleId")

	if err := models.RecordArticleRead(r.Context(), c.DB, c.Redis, articleID); err != nil {
		http.Error(w, "failed to record read", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type createAppealRequest struct {
	ArticleID     string  `json:"articleId"`
	StakedPercent float64 `json:"stakedPercent"`
	EvidenceText  string  `json:"evidenceText"`
	EvidenceURL   string  `json:"evidenceUrl"`
}

// CreateAppeal implements FR-5/F-14.
func (c *ArticlesController) CreateAppeal(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	claims, ok := auth.FromContext(r.Context())
	if !ok || claims == nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	var req createAppealRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	staked, err := models.CreateAppeal(r.Context(), c.DB, claims.UserID, models.NewAppealInput{
		ArticleID: req.ArticleID, StakedPercent: req.StakedPercent,
		EvidenceText: req.EvidenceText, EvidenceURL: req.EvidenceURL,
	})
	if err != nil {
		switch {
		case errors.Is(err, models.ErrMissingAppealInputs), errors.Is(err, models.ErrStakePercentRange), errors.Is(err, models.ErrMissingEvidence):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, models.ErrArticleNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		case errors.Is(err, models.ErrNotArticleOwner):
			http.Error(w, err.Error(), http.StatusForbidden)
		case errors.Is(err, models.ErrAppealAlreadyActive):
			http.Error(w, err.Error(), http.StatusConflict)
		default:
			http.Error(w, "failed to create appeal", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]float64{"stakedRankScore": staked})
}
