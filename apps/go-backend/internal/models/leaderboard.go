package models

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/yeanur-ys/nextGENjournalism/apps/go-backend/internal/redisstore"
)

// LeaderboardEntry is the response shape for GET /leaderboard — kept here
// rather than in views/ for the same reason PendingAuditor is: it's a direct
// projection assembled by the repository function below, not something a
// Controller needs to reshape further.
type LeaderboardEntry struct {
	JournalistID string  `json:"journalistId"`
	DisplayName  string  `json:"displayName"`
	RankScore    float64 `json:"rankScore"`
}

// Top implements F-19/NFR-3: the leaderboard is read entirely from the Redis
// sorted set (near-instant regardless of the total historical node count) —
// Postgres is only touched once, in a single batched query, to attach a
// human display name to each of the (at most 50) journalist IDs returned.
func Top(ctx context.Context, rdb *redis.Client, db *pgxpool.Pool) ([]LeaderboardEntry, error) {
	results, err := rdb.ZRevRangeWithScores(ctx, redisstore.LeaderboardKey, 0, 49).Result()
	if err != nil {
		return nil, err
	}

	entries := make([]LeaderboardEntry, 0, len(results))
	ids := make([]string, 0, len(results))
	for _, z := range results {
		member, ok := z.Member.(string)
		if !ok {
			continue
		}
		entries = append(entries, LeaderboardEntry{JournalistID: member, RankScore: z.Score})
		ids = append(ids, member)
	}

	names := make(map[string]string, len(ids))
	if db != nil && len(ids) > 0 {
		rows, err := db.Query(ctx, `SELECT id, display_name FROM users WHERE id = ANY($1)`, ids)
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
	return entries, nil
}

// UpdateLeaderboardScore pushes one journalist's freshly recalculated rank
// score into the Redis sorted set. Both claim self-correction and consensus
// resolution need this after recomputing a rank score — giving them one
// shared method instead of each re-inlining rdb.ZAdd(ctx,
// redisstore.LeaderboardKey, ...) independently.
func UpdateLeaderboardScore(ctx context.Context, rdb *redis.Client, journalistID string, score float64) error {
	if rdb == nil {
		return nil
	}
	return rdb.ZAdd(ctx, redisstore.LeaderboardKey, redis.Z{Score: score, Member: journalistID}).Err()
}

// RebuildLeaderboard repopulates the Redis leaderboard sorted set from
// Postgres.
//
// The leaderboard read path is Redis-only, which is correct for NFR-3 (reads
// must be near-instant regardless of corpus size) but left the system with no
// way to ever recover the set. Redis here is a cache with no persistence
// configured, and nothing wrote to it except the live vote/self-correct paths
// — so a restarted Redis, a fresh environment, or a database seeded directly
// in SQL all produced a permanently empty leaderboard while Postgres held
// perfectly good rank scores. There was no code path anywhere that could
// reconcile the two.
//
// Called directly from cmd/api/main.go at startup — there is no HTTP request
// behind this one, so it stays a plain Model function with no controller.
// Running it at startup makes Redis a true derived cache: authoritative for
// reads, reconstructible from the source of truth at any time.
func RebuildLeaderboard(ctx context.Context, db *pgxpool.Pool, rdb *redis.Client) (int, error) {
	if db == nil || rdb == nil {
		return 0, nil
	}

	rows, err := db.Query(ctx, `
		SELECT id, rank_score FROM users WHERE role = 'journalist'
	`)
	if err != nil {
		return 0, fmt.Errorf("loading rank scores: %w", err)
	}
	defer rows.Close()

	members := make([]redis.Z, 0, 128)
	for rows.Next() {
		var id string
		var score float64
		if err := rows.Scan(&id, &score); err != nil {
			return 0, fmt.Errorf("scanning rank score: %w", err)
		}
		members = append(members, redis.Z{Score: score, Member: id})
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if len(members) == 0 {
		return 0, nil
	}

	// Replace rather than merge: a journalist deleted from Postgres must not
	// linger in the ranking forever.
	pipe := rdb.TxPipeline()
	pipe.Del(ctx, redisstore.LeaderboardKey)
	pipe.ZAdd(ctx, redisstore.LeaderboardKey, members...)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, fmt.Errorf("writing leaderboard: %w", err)
	}

	return len(members), nil
}
