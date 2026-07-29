package models

import (
	"context"
	"math"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Default weights: w2 > w1 rewards self-correction over baseline verification,
// w3 heavily penalizes proven false claims (SRS Section 4).
const (
	DefaultW1 = 1.0
	DefaultW2 = 1.5
	DefaultW3 = 4.0
)

// JournalistRankScore implements SRS formula (1):
// R = log10(1+V) + w1*Cvd + w2*Csc - w3*Cf
func JournalistRankScore(readershipVolume float64, verifiedClaims, selfCorrectedClaims, falseClaims float64, w1, w2, w3 float64) float64 {
	return math.Log10(1+readershipVolume) + w1*verifiedClaims + w2*selfCorrectedClaims - w3*falseClaims
}

// CorruptionFactor implements SRS formula (2), computed here server-side and
// cached; the frontend shader (node.fragment.glsl) also computes it client-side
// for smooth interpolation between updates.
func CorruptionFactor(verifiedClaims, selfCorrectedClaims, falseClaims float64) float64 {
	total := verifiedClaims + selfCorrectedClaims + falseClaims
	if total == 0 {
		return 0
	}
	return falseClaims / total
}

// AuditorTrustWeight implements SRS formula (3):
// Wa = log10(1+Vs) * (1 - Vf/Vtotal)
func AuditorTrustWeight(successfulVotes, failedVotes, totalVotes float64) float64 {
	if totalVotes == 0 {
		return 0
	}
	return math.Log10(1+successfulVotes) * (1 - failedVotes/totalVotes)
}

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
