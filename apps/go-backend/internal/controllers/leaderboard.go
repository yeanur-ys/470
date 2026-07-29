package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/yeanur-ys/nextGENjournalism/apps/go-backend/internal/models"
)

type LeaderboardController struct {
	Redis *redis.Client
	DB    *pgxpool.Pool
}

func NewLeaderboardController(rdb *redis.Client, db *pgxpool.Pool) *LeaderboardController {
	return &LeaderboardController{Redis: rdb, DB: db}
}

// Top implements F-19/NFR-3: the top 50 journalists by rank score.
func (c *LeaderboardController) Top(w http.ResponseWriter, r *http.Request) {
	entries, err := models.Top(r.Context(), c.Redis, c.DB)
	if err != nil {
		http.Error(w, "failed to load leaderboard", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(entries)
}
