package consensus

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

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

type voteRequest struct {
	Stake   float64 `json:"stake"`
	Verdict bool    `json:"verdict"`
}

// Vote implements FR-6: an auditor stakes reputation before voting on a claim.
func (h *Handler) Vote(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.FromContext(r.Context())
	claimID := r.PathValue("claimId")

	var req voteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Stake <= 0 {
		http.Error(w, "a positive stake is required", http.StatusBadRequest)
		return
	}

	var credentialVerified bool
	if err := h.DB.QueryRow(context.Background(),
		`SELECT credential_verified FROM users WHERE id = $1`, claims.UserID,
	).Scan(&credentialVerified); err != nil {
		http.Error(w, "could not verify auditor status", http.StatusInternalServerError)
		return
	}
	if !credentialVerified {
		http.Error(w, "your credentials are still pending admin verification", http.StatusForbidden)
		return
	}

	_, err := h.DB.Exec(context.Background(), `
		INSERT INTO votes (claim_id, auditor_id, stake, verdict)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (claim_id, auditor_id) DO UPDATE SET stake = $3, verdict = $4
	`, claimID, claims.UserID, req.Stake, req.Verdict)
	if err != nil {
		http.Error(w, "failed to record vote", http.StatusInternalServerError)
		return
	}

	verdict, err := TryResolve(context.Background(), h.DB, h.Redis, claimID)
	switch {
	case errors.Is(err, ErrNoConsensusYet):
		w.WriteHeader(http.StatusAccepted) // vote recorded, consensus pending
		return
	case err != nil:
		http.Error(w, "failed to evaluate consensus", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"resolved": true, "verdict": verdict})
}
