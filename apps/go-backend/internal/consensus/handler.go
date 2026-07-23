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

// Vote implements FR-6/F-12: an auditor stakes reputation before voting on a
// claim, and the stake is actually locked against their balance — F-12's
// acceptance criterion is "the system deducts or locks an auditor's staked
// reputation weight before allowing a vote", which the previous version did
// not do (it wrote votes.stake and never touched the auditor's balance, so
// the same reputation could be staked on unlimited claims simultaneously).
func (h *Handler) Vote(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	claims, ok := auth.FromContext(r.Context())
	if !ok || claims == nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	claimID := r.PathValue("claimId")

	var req voteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Stake <= 0 {
		http.Error(w, "a positive stake is required", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		http.Error(w, "could not record vote", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(ctx)

	// Lock the auditor row for the balance check so two concurrent votes
	// can't both pass an available-reputation check against the same balance.
	var credentialVerified bool
	var rankScore, lockedStake float64
	if err := tx.QueryRow(ctx,
		`SELECT credential_verified, rank_score, locked_stake FROM users WHERE id = $1 FOR UPDATE`,
		claims.UserID,
	).Scan(&credentialVerified, &rankScore, &lockedStake); err != nil {
		http.Error(w, "could not verify auditor status", http.StatusInternalServerError)
		return
	}

	// NFR-6: Sybil resistance — an unverified credential cannot vote.
	if !credentialVerified {
		http.Error(w, "your credentials are still pending admin verification", http.StatusForbidden)
		return
	}

	// The claim must still be open. Voting on an already-resolved claim would
	// lock reputation that no resolution will ever settle, stranding it.
	var status string
	if err := tx.QueryRow(ctx, `SELECT status FROM claims WHERE id = $1`, claimID).Scan(&status); err != nil {
		http.Error(w, "claim not found", http.StatusNotFound)
		return
	}
	if status != "pending" {
		http.Error(w, "this claim has already been resolved", http.StatusConflict)
		return
	}

	// An auditor may revise an open vote; only the delta needs to be covered.
	var previousStake float64
	_ = tx.QueryRow(ctx,
		`SELECT stake FROM votes WHERE claim_id = $1 AND auditor_id = $2`, claimID, claims.UserID,
	).Scan(&previousStake)

	if available := rankScore - lockedStake + previousStake; req.Stake > available {
		http.Error(w, "stake exceeds your available reputation", http.StatusUnprocessableEntity)
		return
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO votes (claim_id, auditor_id, stake, verdict)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (claim_id, auditor_id) DO UPDATE SET stake = $3, verdict = $4
	`, claimID, claims.UserID, req.Stake, req.Verdict); err != nil {
		http.Error(w, "failed to record vote", http.StatusInternalServerError)
		return
	}

	if _, err := tx.Exec(ctx,
		`UPDATE users SET locked_stake = locked_stake - $2 + $3 WHERE id = $1`,
		claims.UserID, previousStake, req.Stake,
	); err != nil {
		http.Error(w, "failed to lock stake", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		http.Error(w, "failed to record vote", http.StatusInternalServerError)
		return
	}

	// Consensus evaluation runs in its own transaction, after the vote is
	// durably recorded — so a vote is never lost just because the resolution
	// that followed it hit a conflict.
	verdict, err := TryResolve(ctx, h.DB, h.Redis, claimID)
	switch {
	case errors.Is(err, ErrNoConsensusYet):
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted) // vote recorded, consensus pending
		_ = json.NewEncoder(w).Encode(map[string]any{"resolved": false})
		return
	case err != nil:
		http.Error(w, "failed to evaluate consensus", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"resolved": true, "verdict": verdict})
}
