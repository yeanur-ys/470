package models

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

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

var ErrNoConsensusYet = errors.New("claim has not reached cross-tag consensus")

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

var (
	ErrAuditorNotVerified = errors.New("your credentials are still pending admin verification")
	ErrInvalidStake       = errors.New("a positive stake is required")
	ErrInsufficientStake  = errors.New("stake exceeds your available reputation")
)

// CastVote implements FR-6/F-12: an auditor stakes reputation before voting
// on a claim, and the stake is actually locked against their balance —
// F-12's acceptance criterion is "the system deducts or locks an auditor's
// staked reputation weight before allowing a vote". Returns whether the
// claim resolved as a result of this vote, and the verdict if so.
func CastVote(ctx context.Context, db *pgxpool.Pool, rdb *redis.Client, auditorID, claimID string, stake float64, verdict bool) (resolved bool, finalVerdict bool, err error) {
	if stake <= 0 {
		return false, false, ErrInvalidStake
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return false, false, err
	}
	defer tx.Rollback(ctx)

	// Lock the auditor row for the balance check so two concurrent votes
	// can't both pass an available-reputation check against the same balance.
	var credentialVerified bool
	var rankScore, lockedStake float64
	if err := tx.QueryRow(ctx,
		`SELECT credential_verified, rank_score, locked_stake FROM users WHERE id = $1 FOR UPDATE`,
		auditorID,
	).Scan(&credentialVerified, &rankScore, &lockedStake); err != nil {
		return false, false, err
	}

	// NFR-6: Sybil resistance — an unverified credential cannot vote.
	if !credentialVerified {
		return false, false, ErrAuditorNotVerified
	}

	// The claim must still be open. Voting on an already-resolved claim would
	// lock reputation that no resolution will ever settle, stranding it.
	var status string
	if err := tx.QueryRow(ctx, `SELECT status FROM claims WHERE id = $1`, claimID).Scan(&status); err != nil {
		return false, false, ErrClaimNotFound
	}
	if status != "pending" {
		return false, false, ErrClaimAlreadyResolved
	}

	// An auditor may revise an open vote; only the delta needs to be covered.
	var previousStake float64
	_ = tx.QueryRow(ctx,
		`SELECT stake FROM votes WHERE claim_id = $1 AND auditor_id = $2`, claimID, auditorID,
	).Scan(&previousStake)

	if available := rankScore - lockedStake + previousStake; stake > available {
		return false, false, ErrInsufficientStake
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO votes (claim_id, auditor_id, stake, verdict)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (claim_id, auditor_id) DO UPDATE SET stake = $3, verdict = $4
	`, claimID, auditorID, stake, verdict); err != nil {
		return false, false, err
	}

	if _, err := tx.Exec(ctx,
		`UPDATE users SET locked_stake = locked_stake - $2 + $3 WHERE id = $1`,
		auditorID, previousStake, stake,
	); err != nil {
		return false, false, err
	}

	if err := tx.Commit(ctx); err != nil {
		return false, false, err
	}

	// Consensus evaluation runs in its own transaction, after the vote is
	// durably recorded — so a vote is never lost just because the resolution
	// that followed it hit a conflict.
	finalVerdict, err = TryResolve(ctx, db, rdb, claimID)
	if errors.Is(err, ErrNoConsensusYet) {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	return true, finalVerdict, nil
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

	journalistID, rankScore, err := BumpArticleCounterAndRecalculateTx(ctx, tx, claimID, counterColumn)
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
	return UpdateLeaderboardScore(ctx, rdb, journalistID, rankScore)
}

// AlignedRewardRate exists because a ledger that only ever subtracts is not an
// economy. With slashing alone, every auditor's reputation decays
// monotonically toward zero no matter how well they judge — and since a vote
// requires available reputation to stake against, the whole consensus
// mechanism eventually deadlocks with nobody able to vote on anything.
//
// Paying aligned voters a fraction of their stake makes staking a genuine
// risk/reward decision: a correct call earns half the stake, a wrong one
// forfeits all of it. The 2:1 downside keeps it rational to abstain unless
// reasonably confident, which is the behaviour FR-6/FR-8 are trying to buy.
const AlignedRewardRate = 0.5

// ApplySlashingTx settles every vote on a resolved claim (FR-8/F-13): the
// stake each auditor locked at vote time is released from
// users.locked_stake; their V_s/V_f counters move (SRS formula 3 inputs);
// misaligned voters forfeit the staked reputation outright; trust_weight is
// recomputed from the updated counters. Runs inside the caller's transaction
// so a claim can never be marked resolved without its stakes being settled.
func ApplySlashingTx(ctx context.Context, tx pgx.Tx, claimID string, finalVerdict bool) error {
	if _, err := tx.Exec(ctx, `
		UPDATE votes SET aligned_with_consensus = (verdict = $2)
		WHERE claim_id = $1
	`, claimID, finalVerdict); err != nil {
		return err
	}

	// Settle stakes and vote counters in one statement so an auditor who
	// somehow has two rows for this claim (they can't — there's a UNIQUE
	// constraint — but defensively) is still only settled once per row.
	//
	// GREATEST(..., 0) on locked_stake and rank_score keeps the ledger from
	// going negative through rounding or a manually-edited row; reputation
	// bottoming out at zero is the intended floor, not a wrap-around.
	if _, err := tx.Exec(ctx, `
		WITH settled AS (
			SELECT v.auditor_id,
			       SUM(v.stake)                                          AS released,
			       SUM(CASE WHEN v.verdict = $2 THEN 0 ELSE v.stake END) AS forfeited,
			       SUM(CASE WHEN v.verdict = $2 THEN v.stake * $3 ELSE 0 END) AS rewarded,
			       COUNT(*) FILTER (WHERE v.verdict = $2)                AS aligned,
			       COUNT(*) FILTER (WHERE v.verdict <> $2)               AS misaligned
			FROM votes v
			WHERE v.claim_id = $1
			GROUP BY v.auditor_id
		)
		UPDATE users u
		SET locked_stake     = GREATEST(u.locked_stake - s.released, 0),
		    rank_score       = GREATEST(u.rank_score - s.forfeited + s.rewarded, 0),
		    successful_votes = u.successful_votes + s.aligned,
		    failed_votes     = u.failed_votes + s.misaligned
		FROM settled s
		WHERE u.id = s.auditor_id
	`, claimID, finalVerdict, AlignedRewardRate); err != nil {
		return err
	}

	// Recompute W_a = log10(1 + V_s) * (1 - V_f / V_total) for everyone whose
	// counters just moved. Kept in SQL rather than round-tripping through
	// AuditorTrustWeight so it stays inside the transaction; the Go function
	// remains the reference implementation and is what the unit tests pin the
	// formula against.
	_, err := tx.Exec(ctx, `
		UPDATE users u
		SET trust_weight = CASE
			WHEN (u.successful_votes + u.failed_votes) = 0 THEN 0
			ELSE log(10, 1 + u.successful_votes)
			     * (1 - u.failed_votes::double precision / (u.successful_votes + u.failed_votes))
		END
		WHERE u.id IN (SELECT auditor_id FROM votes WHERE claim_id = $1)
	`, claimID)
	return err
}
