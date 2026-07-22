package leaderboard

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/yeanur-ys/nextGENjournalism/apps/go-backend/internal/redisstore"
)

// Rebuild repopulates the Redis leaderboard sorted set from Postgres.
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
// Running this at startup makes Redis a true derived cache: authoritative for
// reads, reconstructible from the source of truth at any time.
func Rebuild(ctx context.Context, db *pgxpool.Pool, rdb *redis.Client) (int, error) {
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
