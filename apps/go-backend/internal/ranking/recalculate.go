package ranking

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// BumpArticleCounterAndRecalculate increments the article's verified/self-corrected/false
// claim counter (counterColumn), recomputes the author's Journalist Rank Score
// (SRS formula 1), and persists it. Callers must only ever pass one of the
// three fixed column-name literals below — never anything derived from
// request input — since it's interpolated directly into the SQL.
//
// The Postgres update flows to Neo4j automatically through the existing
// Debezium -> Kafka -> CDC-sync pipeline, so the graph's Corruption Factor
// visualization stays in sync without extra plumbing here.
func BumpArticleCounterAndRecalculate(ctx context.Context, db *pgxpool.Pool, claimID, counterColumn string) (journalistID string, rankScore float64, err error) {
	var readership, verified, selfCorrected, falseClaims float64

	query := `
		UPDATE articles a
		SET ` + counterColumn + ` = ` + counterColumn + ` + 1
		FROM claims c
		WHERE c.id = $1 AND a.id = c.article_id
		RETURNING a.journalist_id, a.readership_volume, a.verified_claims, a.self_corrected_claims, a.false_claims
	`
	if err := db.QueryRow(ctx, query, claimID).Scan(&journalistID, &readership, &verified, &selfCorrected, &falseClaims); err != nil {
		return "", 0, err
	}

	rankScore = JournalistRankScore(readership, verified, selfCorrected, falseClaims, DefaultW1, DefaultW2, DefaultW3)

	if _, err := db.Exec(ctx, `UPDATE users SET rank_score = $2 WHERE id = $1`, journalistID, rankScore); err != nil {
		return "", 0, err
	}

	return journalistID, rankScore, nil
}
