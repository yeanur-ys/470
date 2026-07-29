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

type ConsensusController struct {
	DB    *pgxpool.Pool
	Redis *redis.Client
}

func NewConsensusController(db *pgxpool.Pool, rdb *redis.Client) *ConsensusController {
	return &ConsensusController{DB: db, Redis: rdb}
}

type voteRequest struct {
	Stake   float64 `json:"stake"`
	Verdict bool    `json:"verdict"`
}

// Vote implements FR-6/F-12.
func (c *ConsensusController) Vote(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	claims, ok := auth.FromContext(r.Context())
	if !ok || claims == nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	claimID := r.PathValue("claimId")

	var req voteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "a positive stake is required", http.StatusBadRequest)
		return
	}

	resolved, verdict, err := models.CastVote(r.Context(), c.DB, c.Redis, claims.UserID, claimID, req.Stake, req.Verdict)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrInvalidStake):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, models.ErrAuditorNotVerified):
			http.Error(w, err.Error(), http.StatusForbidden)
		case errors.Is(err, models.ErrClaimNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		case errors.Is(err, models.ErrClaimAlreadyResolved):
			http.Error(w, err.Error(), http.StatusConflict)
		case errors.Is(err, models.ErrInsufficientStake):
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		default:
			http.Error(w, "failed to record vote", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if !resolved {
		w.WriteHeader(http.StatusAccepted) // vote recorded, consensus pending
		_ = json.NewEncoder(w).Encode(map[string]any{"resolved": false})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"resolved": true, "verdict": verdict})
}
