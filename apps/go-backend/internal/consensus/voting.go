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

// Vote is the minimal shape EvaluateCrossTagConsensus needs — deliberately
// excluding anything DB-shaped (auditor ID, timestamps) so this stays a pure,
// easily testable function.
//
// Tags is the auditor's FULL tag set, not just their first tag. The previous
// implementation read `u.tags[1]` and threw the rest away, which made an
// auditor tagged {"Economic Analyst","Security Analyst"} indistinguishable
// from one tagged {"Economic Analyst"} — and "non-overlapping" (FR-7) is a
// property of whole tag sets, so it cannot be decided from one tag each.
type Vote struct {
	Tags        []string
	Verdict     bool
	Stake       float64
	TrustWeight float64
}

// effectiveWeight is the influence a single vote carries: the reputation
// staked on it, amplified by the auditor's Auditor Trust Weight (SRS formula
// 3). The +1 is a bootstrap floor — W_a is log10(1+V_s)*(1-V_f/V_total),
// which is exactly 0 for an auditor who has never voted, and multiplying by a
// raw 0 would mean a platform full of new auditors could never resolve
// anything at all. With the floor, a brand-new auditor still carries their
// stake, and a proven one carries meaningfully more.
func effectiveWeight(v Vote) float64 {
	return v.Stake * (1 + v.TrustWeight)
}

// hasNonOverlappingPair reports whether the given tag sets contain at least
// two that are mutually disjoint. This is the literal reading of FR-7/F-11:
// "consensus from auditors holding non-overlapping category tags". Two
// auditors who both cover "Economic Analyst" are not independent evidence
// about a claim no matter how many other tags they hold, so they don't count
// as a cross-tag pair.
func hasNonOverlappingPair(tagSets [][]string) bool {
	for i := 0; i < len(tagSets); i++ {
		for j := i + 1; j < len(tagSets); j++ {
			if disjoint(tagSets[i], tagSets[j]) {
				return true
			}
		}
	}
	return false
}

func disjoint(a, b []string) bool {
	if len(a) == 0 || len(b) == 0 {
		return false // an untagged auditor brings no category coverage at all
	}
	inA := make(map[string]bool, len(a))
	for _, t := range a {
		inA[t] = true
	}
	for _, t := range b {
		if inA[t] {
			return false
		}
	}
	return true
}

// EvaluateCrossTagConsensus implements FR-7/F-10/F-11. A verdict resolves only
// when BOTH conditions hold on the same side:
//
//  1. Cross-tag coverage (FR-7/F-11): at least two auditors on that side hold
//     mutually non-overlapping category tags.
//  2. Weighted majority (FR-10): that side carries strictly more reputation
//     weight — sum(stake * (1 + trustWeight)) — than the other. F-10's
//     acceptance criterion is explicit that outcomes come from "auditor vote
//     weight rather than simple majority count alone", so a single
//     high-reputation auditor can outweigh several low-reputation ones.
//
// Returns (verdict, true) once resolved, or (false, false) while still
// pending. Both sides are evaluated explicitly rather than by ranging over a
// map: the previous version iterated a map[bool]map[string]bool and returned
// the first side it found with 2+ tags, so when both sides qualified the
// verdict depended on Go's randomised map iteration order — the same votes
// could resolve true on one request and false on the next.
func EvaluateCrossTagConsensus(votes []Vote) (verdict bool, resolved bool) {
	var forTags, againstTags [][]string
	var forWeight, againstWeight float64

	for _, v := range votes {
		if v.Verdict {
			forTags = append(forTags, v.Tags)
			forWeight += effectiveWeight(v)
		} else {
			againstTags = append(againstTags, v.Tags)
			againstWeight += effectiveWeight(v)
		}
	}

	forQualifies := hasNonOverlappingPair(forTags) && forWeight > againstWeight
	againstQualifies := hasNonOverlappingPair(againstTags) && againstWeight > forWeight

	switch {
	case forQualifies:
		return true, true
	case againstQualifies:
		return false, true
	default:
		// Includes the exact-tie case, which stays deliberately unresolved:
		// resolving a dead heat would mean picking a winner arbitrarily.
		return false, false
	}
}

// TryResolve loads every vote on a claim (with the voting auditor's full tag
// set and current trust weight), asks EvaluateCrossTagConsensus whether that's
// enough, and if so commits the whole resolution — claim status, article
// counters, journalist rank, leaderboard, and auditor slashing — in one
// transaction.
func TryResolve(ctx context.Context, db *pgxpool.Pool, rdb *redis.Client, claimID string) (bool, error) {
	rows, err := db.Query(ctx, `
		SELECT u.tags, v.verdict, v.stake, u.trust_weight
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
		if err := rows.Scan(&v.Tags, &v.Verdict, &v.Stake, &v.TrustWeight); err != nil {
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

// resolveClaim commits a resolution atomically. Everything here used to run as
// separate un-transacted statements, so a failure partway through could leave
// a claim marked resolved with the article counters never bumped, or auditors
// slashed for a verdict the claim never actually recorded.
func resolveClaim(ctx context.Context, db *pgxpool.Pool, rdb *redis.Client, claimID string, verdict bool) error {
	status := "false"
	// counterColumn is one of two fixed, non-user-controlled literals below;
	// never derived from request input, so string-building it into SQL is safe.
	counterColumn := "false_claims"
	if verdict {
		status = "verified"
		counterColumn = "verified_claims"
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Guard against a double resolution: two auditors voting concurrently can
	// both see "resolved" and both try to commit, which would bump the
	// article's counters twice for one claim. The row lock plus the status
	// re-check makes the second one a no-op.
	var currentStatus string
	if err := tx.QueryRow(ctx,
		`SELECT status FROM claims WHERE id = $1 FOR UPDATE`, claimID,
	).Scan(&currentStatus); err != nil {
		return err
	}
	if currentStatus != "pending" {
		return nil // already resolved by a concurrent voter
	}

	if _, err := tx.Exec(ctx, `UPDATE claims SET status = $2::claim_status WHERE id = $1`, claimID, status); err != nil {
		return err
	}

	journalistID, rankScore, err := ranking.BumpArticleCounterAndRecalculateTx(ctx, tx, claimID, counterColumn)
	if err != nil {
		return err
	}

	// FR-8/F-13: settle every auditor's stake and reputation against the
	// resolved verdict.
	if err := ApplySlashingTx(ctx, tx, claimID, verdict); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	// Redis is a cache of the leaderboard, not the source of truth, so it's
	// updated after the commit — a failure here must not roll back a
	// legitimate consensus result.
	if rdb != nil {
		if err := rdb.ZAdd(ctx, redisstore.LeaderboardKey, redis.Z{Score: rankScore, Member: journalistID}).Err(); err != nil {
			return err
		}
	}
	return nil
}
