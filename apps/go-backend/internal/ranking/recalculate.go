package ranking

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BumpArticleCounterAndRecalculate increments the article's
// verified/self-corrected/false claim counter, recomputes the author's
// Journalist Rank Score (SRS formula 1), and persists it — outside any
// transaction.
func BumpArticleCounterAndRecalculate(ctx context.Context, db *pgxpool.Pool, claimID, counterColumn string) (journalistID string, rankScore float64, err error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return "", 0, err
	}
	defer tx.Rollback(ctx)

	journalistID, rankScore, err = BumpArticleCounterAndRecalculateTx(ctx, tx, claimID, counterColumn)
	if err != nil {
		return "", 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", 0, err
	}
	return journalistID, rankScore, nil
}

// BumpArticleCounterAndRecalculateTx is the same operation enlisted in an
// existing transaction. Callers must only ever pass one of the three fixed
// column-name literals (verified_claims, self_corrected_claims, false_claims)
// as counterColumn — never anything derived from request input — since it is
// interpolated directly into the SQL.
//
// The Postgres update flows to Neo4j automatically through the existing
// Debezium -> Kafka -> CDC-sync pipeline, so the graph's Corruption Factor
// visualization stays in sync without extra plumbing here.
func BumpArticleCounterAndRecalculateTx(ctx context.Context, tx pgx.Tx, claimID, counterColumn string) (journalistID string, rankScore float64, err error) {
	var readership, verified, selfCorrected, falseClaims float64

	query := `
		UPDATE articles a
		SET ` + counterColumn + ` = ` + counterColumn + ` + 1
		FROM claims c
		WHERE c.id = $1 AND a.id = c.article_id
		RETURNING a.journalist_id, a.readership_volume, a.verified_claims, a.self_corrected_claims, a.false_claims
	`
	if err := tx.QueryRow(ctx, query, claimID).Scan(
		&journalistID, &readership, &verified, &selfCorrected, &falseClaims,
	); err != nil {
		return "", 0, err
	}

	rankScore = JournalistRankScore(readership, verified, selfCorrected, falseClaims, DefaultW1, DefaultW2, DefaultW3)

	if _, err := tx.Exec(ctx, `UPDATE users SET rank_score = $2 WHERE id = $1`, journalistID, rankScore); err != nil {
		return "", 0, err
	}

	return journalistID, rankScore, nil
}
