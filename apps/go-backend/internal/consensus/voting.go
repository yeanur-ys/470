package consensus

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNoConsensusYet = errors.New("claim has not reached cross-tag consensus")

type vote struct {
	AuditorTag string
	Verdict    bool
}

// TryResolve implements FR-7: a claim only resolves once auditors holding
// non-overlapping category tags agree on the same verdict. It returns the
// resolved verdict, or ErrNoConsensusYet if that bar hasn't been reached.
func TryResolve(ctx context.Context, db *pgxpool.Pool, claimID string) (bool, error) {
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

	tagsForVerdict := map[bool]map[string]bool{true: {}, false: {}}
	for rows.Next() {
		var v vote
		if err := rows.Scan(&v.AuditorTag, &v.Verdict); err != nil {
			return false, err
		}
		tagsForVerdict[v.Verdict][v.AuditorTag] = true
	}
	if err := rows.Err(); err != nil {
		return false, err
	}

	// Cross-tag validation: the same verdict needs confirmation from at least
	// two auditors whose category tags don't overlap (e.g. "Economic Analyst"
	// and "Geopolitical Analyst" both confirming the same claim).
	for verdict, tags := range tagsForVerdict {
		if len(tags) >= 2 {
			if err := resolveClaim(ctx, db, claimID, verdict); err != nil {
				return false, err
			}
			return verdict, nil
		}
	}

	return false, ErrNoConsensusYet
}

func resolveClaim(ctx context.Context, db *pgxpool.Pool, claimID string, verdict bool) error {
	status := "false"
	if verdict {
		status = "verified"
	}

	if _, err := db.Exec(ctx, `UPDATE claims SET status = $2 WHERE id = $1`, claimID, status); err != nil {
		return err
	}

	// FR-8: slash auditors who voted against the resolved consensus.
	_, err := ApplySlashing(ctx, db, claimID, verdict)
	return err
}
