package consensus

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ApplySlashing marks every vote on claimID as aligned or not with the final
// verdict, and returns the auditor IDs that should be penalized (FR-8). The
// actual reputation deduction is folded into ranking.AuditorTrustWeight the
// next time each auditor's trust weight is recalculated (see internal/ranking).
func ApplySlashing(ctx context.Context, db *pgxpool.Pool, claimID string, finalVerdict bool) ([]string, error) {
	rows, err := db.Query(ctx, `
		UPDATE votes
		SET aligned_with_consensus = (verdict = $2)
		WHERE claim_id = $1
		RETURNING auditor_id
	`, claimID, finalVerdict)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var slashed []string
	for rows.Next() {
		var auditorID string
		if err := rows.Scan(&auditorID); err != nil {
			return nil, err
		}
		slashed = append(slashed, auditorID)
	}
	return slashed, rows.Err()
}
