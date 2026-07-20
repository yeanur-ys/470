package leaderboard

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/yeanur-ys/nextGENjournalism/apps/go-backend/internal/redisstore"
)

type Handler struct {
	Redis *redis.Client
	DB    *pgxpool.Pool
}

func NewHandler(rdb *redis.Client, db *pgxpool.Pool) *Handler {
	return &Handler{Redis: rdb, DB: db}
}

type entry struct {
	JournalistID string  `json:"journalistId"`
	DisplayName  string  `json:"displayName"`
	RankScore    float64 `json:"rankScore"`
}

// Top implements F-19/NFR-3: the leaderboard is read entirely from the Redis
// sorted set (near-instant regardless of the total historical node count) —
// Postgres is only touched once, in a single batched query, to attach a
// human display name to each of the (at most 50) journalist IDs returned.
func (h *Handler) Top(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	results, err := h.Redis.ZRevRangeWithScores(ctx, redisstore.LeaderboardKey, 0, 49).Result()
	if err != nil {
		http.Error(w, "failed to load leaderboard", http.StatusInternalServerError)
		return
	}

	entries := make([]entry, 0, len(results))
	ids := make([]string, 0, len(results))
	for _, z := range results {
		member, ok := z.Member.(string)
		if !ok {
			continue
		}
		entries = append(entries, entry{JournalistID: member, RankScore: z.Score})
		ids = append(ids, member)
	}

	names := make(map[string]string, len(ids))
	if h.DB != nil && len(ids) > 0 {
		rows, err := h.DB.Query(ctx, `SELECT id, display_name FROM users WHERE id = ANY($1)`, ids)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var id, name string
				if err := rows.Scan(&id, &name); err == nil {
					names[id] = name
				}
			}
		}
	}
	for i := range entries {
		if name, ok := names[entries[i].JournalistID]; ok {
			entries[i].DisplayName = name
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(entries)
}
