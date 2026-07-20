package consensus

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/yeanur-ys/nextGENjournalism/apps/go-backend/internal/ranking"
	"github.com/yeanur-ys/nextGENjournalism/apps/go-backend/internal/redisstore"
)

var ErrNoConsensusYet = errors.New("claim has not reached cross-tag consensus")

// Vote is the minimal shape EvaluateCrossTagConsensus needs — just enough to
// decide whether a verdict has reached consensus, deliberately excluding
// anything DB-shaped (auditor ID, stake, timestamps) so this stays a pure,
// easily testable function.
type Vote struct {
	Tag     string
	Verdict bool
}

// EvaluateCrossTagConsensus implements FR-7/F-11: a verdict resolves only
// once auditors holding at least two distinct, non-overlapping category tags
// agree on it. Two auditors who share the same tag agreeing is NOT enough —
// that's the entire point of "cross-tag" validation (it's what stops a
// single expertise clique from rubber-stamping a claim). Returns
// (verdict, true) once resolved, or (false, false) while still pending.
func EvaluateCrossTagConsensus(votes []Vote) (verdict bool, resolved bool) {
	tagsForVerdict := map[bool]map[string]bool{true: {}, false: {}}
	for _, v := range votes {
		tagsForVerdict[v.Verdict][v.Tag] = true
	}
	for verdict, tags := range tagsForVerdict {
		if len(tags) >= 2 {
			return verdict, true
		}
	}
	return false, false
}

// TryResolve implements FR-7: a claim only resolves once auditors holding
// non-overlapping category tags agree on the same verdict. It returns the
// resolved verdict, or ErrNoConsensusYet if that bar hasn't been reached.
func TryResolve(ctx context.Context, db *pgxpool.Pool, rdb *redis.Client, claimID string) (bool, error) {
	rows, err := db.Query(ctx, `
		SELECT u.tags[1], v.verdict
		FROM votes v
		JOIN users u ON u.id = v.auditor_id
		WHERE v.claim_id = $1
	`, claimID)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	var votes []Vote
	for rows.Next() {
		var v Vote
		if err := rows.Scan(&v.Tag, &v.Verdict); err != nil {
			return false, err
		}
		votes = append(votes, v)
	}
	if err := rows.Err(); err != nil {
		return false, err
	}

	verdict, resolved := EvaluateCrossTagConsensus(votes)
	if !resolved {
		return false, ErrNoConsensusYet
	}

	if err := resolveClaim(ctx, db, rdb, claimID, verdict); err != nil {
		return false, err
	}
	return verdict, nil
}

func resolveClaim(ctx context.Context, db *pgxpool.Pool, rdb *redis.Client, claimID string, verdict bool) error {
	status := "false"
	// counterColumn is one of two fixed, non-user-controlled literals below;
	// never derived from request input, so string-building it into SQL is safe.
	counterColumn := "false_claims"
	if verdict {
		status = "verified"
		counterColumn = "verified_claims"
	}

	if _, err := db.Exec(ctx, `UPDATE claims SET status = $2::claim_status WHERE id = $1`, claimID, status); err != nil {
		return err
	}

	journalistID, rankScore, err := ranking.BumpArticleCounterAndRecalculate(ctx, db, claimID, counterColumn)
	if err != nil {
		return err
	}

	if rdb != nil {
		if err := rdb.ZAdd(ctx, redisstore.LeaderboardKey, redis.Z{Score: rankScore, Member: journalistID}).Err(); err != nil {
			return err
		}
	}

	// FR-8: slash auditors who voted against the resolved consensus.
	_, err = ApplySlashing(ctx, db, claimID, verdict)
	return err
}
