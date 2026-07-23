package consensus

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// ApplySlashingTx settles every vote on a resolved claim (FR-8/F-13).
//
// The previous implementation set votes.aligned_with_consensus and stopped
// there, with a comment claiming the penalty was "folded into
// ranking.AuditorTrustWeight the next time each auditor's trust weight is
// recalculated" — but nothing in the codebase ever recalculated it, so
// ranking.AuditorTrustWeight was dead code outside its own unit test and no
// auditor was ever actually penalised for anything. F-13's acceptance
// criterion ("an auditor who votes against the final consensus receives a
// reputation penalty") failed silently.
//
// Now, for every auditor who voted on this claim:
//   - the stake they locked at vote time is released from users.locked_stake;
//   - their V_s / V_f counters move (SRS formula 3 inputs);
//   - misaligned voters forfeit the staked reputation outright;
//   - trust_weight is recomputed from the updated counters.
//
// It all runs inside the caller's transaction so a claim can never be marked
// resolved without its stakes being settled.
//
// AlignedRewardRate exists because a ledger that only ever subtracts is not an
// economy. With slashing alone, every auditor's reputation decays
// monotonically toward zero no matter how well they judge — and since a vote
// now requires available reputation to stake against, the whole consensus
// mechanism eventually deadlocks with nobody able to vote on anything. This
// was observable on the seeded dataset: after settling ~1,500 historical
// votes, every auditor sat at exactly 0 and could not cast a single new vote.
//
// Paying aligned voters a fraction of their stake makes staking a genuine
// risk/reward decision: a correct call earns half the stake, a wrong one
// forfeits all of it. The 2:1 downside keeps it rational to abstain unless
// reasonably confident, which is the behaviour FR-6/FR-8 are trying to buy.
const AlignedRewardRate = 0.5

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
	// ranking.AuditorTrustWeight so it stays inside the transaction; the Go
	// function remains the reference implementation and is what the unit tests
	// pin the formula against.
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
