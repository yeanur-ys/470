package models

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Claim mirrors the claims table.
type Claim struct {
	ID        string    `json:"id"`
	ArticleID string    `json:"articleId"`
	Text      string    `json:"text"`
	Tag       string    `json:"tag"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

// PendingClaim is what auditors browse: a claim joined with enough article
// context to decide whether to vote on it.
type PendingClaim struct {
	ID           string `json:"id"`
	ArticleID    string `json:"articleId"`
	ArticleTitle string `json:"articleTitle"`
	Text         string `json:"text"`
	Tag          string `json:"tag"`
}

var (
	ErrNotClaimArticleOwner  = errors.New("you can only tag claims on your own articles")
	ErrClaimArticleRetracted = errors.New("this article has been retracted")
	ErrClaimNotFound         = errors.New("claim not found")
	ErrNotClaimOwner         = errors.New("you can only self-correct your own claims")
	ErrClaimAlreadyResolved  = errors.New("this claim has already been resolved")
)

// CreateClaim implements FR-3: journalists encapsulate specific statements in
// #Claim tags at publish time (or afterwards) so auditors can vote on them.
//
// Ownership is enforced here rather than left to the route's role check:
// "is a journalist" is not "is THIS article's journalist" — without this,
// any signed-in journalist could attach claims to a rival's article and
// drive their Corruption Factor (FR-10) and Rank Score (FR-16) down by
// having those claims voted false.
func CreateClaim(ctx context.Context, db *pgxpool.Pool, journalistID, articleID, text, tag string) (string, error) {
	var ownerID string
	var isRetracted bool
	if err := db.QueryRow(ctx,
		`SELECT journalist_id, is_retracted FROM articles WHERE id = $1`, articleID,
	).Scan(&ownerID, &isRetracted); err != nil {
		return "", ErrArticleNotFound
	}
	if ownerID != journalistID {
		return "", ErrNotClaimArticleOwner
	}
	if isRetracted {
		return "", ErrClaimArticleRetracted
	}

	var id string
	err := db.QueryRow(ctx, `
		INSERT INTO claims (article_id, text, tag) VALUES ($1, $2, $3) RETURNING id
	`, articleID, text, tag).Scan(&id)
	return id, err
}

// PendingClaims lists claims awaiting cross-tag consensus (FR-7), for
// auditors to pick up and vote on.
func PendingClaims(ctx context.Context, db *pgxpool.Pool) ([]PendingClaim, error) {
	rows, err := db.Query(ctx, `
		SELECT c.id, c.article_id, a.title AS article_title, c.text, c.tag
		FROM claims c
		JOIN articles a ON a.id = c.article_id
		WHERE c.status = 'pending'
		ORDER BY c.created_at ASC
		LIMIT 100
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return pgx.CollectRows(rows, pgx.RowToStructByName[PendingClaim])
}

// SelfCorrectClaim lets a journalist mark their own claim as self-corrected
// before auditors resolve it. SRS formula (1) weighs self-correction (w2)
// higher than baseline verification (w1) to reward getting ahead of a
// mistake.
func SelfCorrectClaim(ctx context.Context, db *pgxpool.Pool, rdb *redis.Client, journalistID, claimID string) error {
	var articleJournalistID, status string
	err := db.QueryRow(ctx, `
		SELECT a.journalist_id, c.status
		FROM claims c
		JOIN articles a ON a.id = c.article_id
		WHERE c.id = $1
	`, claimID).Scan(&articleJournalistID, &status)
	if err != nil {
		return ErrClaimNotFound
	}
	if articleJournalistID != journalistID {
		return ErrNotClaimOwner
	}
	if status != "pending" {
		return ErrClaimAlreadyResolved
	}

	if _, err := db.Exec(ctx, `UPDATE claims SET status = 'self_corrected' WHERE id = $1`, claimID); err != nil {
		return err
	}

	authorID, rankScore, err := BumpArticleCounterAndRecalculate(ctx, db, claimID, "self_corrected_claims")
	if err != nil {
		return err
	}

	return UpdateLeaderboardScore(ctx, rdb, authorID, rankScore)
}
