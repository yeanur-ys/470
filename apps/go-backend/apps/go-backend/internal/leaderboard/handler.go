package leaderboard

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/redis/go-redis/v9"

	"github.com/yeanur-ys/nextGENjournalism/apps/go-backend/internal/redisstore"
)

type Handler struct {
	Redis *redis.Client
}

func NewHandler(rdb *redis.Client) *Handler {
	return &Handler{Redis: rdb}
}

type entry struct {
	JournalistID string  `json:"journalistId"`
	RankScore    float64 `json:"rankScore"`
}

func (h *Handler) Top(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	results, err := h.Redis.ZRevRangeWithScores(ctx, redisstore.LeaderboardKey, 0, 49).Result()
	if err != nil {
		http.Error(w, "failed to load leaderboard", http.StatusInternalServerError)
		return
	}

	entries := make([]entry, 0, len(results))
	for _, z := range results {
		member, ok := z.Member.(string)
		if !ok {
			continue
		}
		entries = append(entries, entry{JournalistID: member, RankScore: z.Score})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(entries)
}
