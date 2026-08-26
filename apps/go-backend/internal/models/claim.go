package models

import (
	"context"
	"errors"
	"strconv"
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

// JournalistClaim is what a journalist browses on their own dashboard: every
// claim tagged across every one of their own articles, so they can find the
// id of a specific claim to self-correct without ever having to know it in
// advance. Before this, the only place a claim's id was ever shown was the
// publish flow's own confirmation table, immediately after tagging it — once
// the journalist navigated away (or the claim was tagged in an earlier
// session), that id was gone from the UI for good.
type JournalistClaim struct {
	ID           string `json:"id"`
	ArticleID    string `json:"articleId"`
	ArticleTitle string `json:"articleTitle"`
	Text         string `json:"text"`
	Tag          string `json:"tag"`
	Status       string `json:"status"`
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

// defaultPendingClaimsPageSize/maxPendingClaimsPageSize bound the auditor
// docket's page size. Before pagination existed, this endpoint had a bare
// LIMIT 100 with no offset: on a large backlog (a seeded demo corpus can
// easily carry 600+ pending claims), a claim tagged today sits far behind
// hundreds of older oldest-first rows and never appeared in the first 100 —
// invisible to every auditor indefinitely, not just delayed.
const (
	defaultPendingClaimsPageSize = 20
	maxPendingClaimsPageSize     = 100
)

// ClampPendingClaimsPageSize turns a raw ?pageSize= query value into a
// bounded int.
func ClampPendingClaimsPageSize(raw string) int {
	if raw == "" {
		return defaultPendingClaimsPageSize
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return defaultPendingClaimsPageSize
	}
	if n > maxPendingClaimsPageSize {
		return maxPendingClaimsPageSize
	}
	return n
}

// ClampPendingClaimsPage turns a raw ?page= query value into a 1-based page
// number.
func ClampPendingClaimsPage(raw string) int {
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 1
	}
	return n
}

// PaginatedPendingClaims is the docket's response shape: one page of the
// FIFO queue plus enough information for the client to page through the rest
// of the backlog instead of only ever seeing its oldest 100 entries.
type PaginatedPendingClaims struct {
	Claims   []PendingClaim `json:"claims"`
	Total    int            `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"pageSize"`
}

// PendingClaims lists claims awaiting cross-tag consensus (FR-7), oldest
// first (a fair FIFO queue) and paginated so a growing backlog never hides
// newer claims outright, only pushes them a page or two further back.
func PendingClaims(ctx context.Context, db *pgxpool.Pool, page, pageSize int) (PaginatedPendingClaims, error) {
	var total int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM claims WHERE status = 'pending'`).Scan(&total); err != nil {
		return PaginatedPendingClaims{}, err
	}

	rows, err := db.Query(ctx, `
		SELECT c.id, c.article_id, a.title AS article_title, c.text, c.tag
		FROM claims c
		JOIN articles a ON a.id = c.article_id
		WHERE c.status = 'pending'
		ORDER BY c.created_at ASC
		LIMIT $1 OFFSET $2
	`, pageSize, (page-1)*pageSize)
	if err != nil {
		return PaginatedPendingClaims{}, err
	}
	defer rows.Close()

	claims, err := pgx.CollectRows(rows, pgx.RowToStructByName[PendingClaim])
	if err != nil {
		return PaginatedPendingClaims{}, err
	}
	if claims == nil {
		claims = []PendingClaim{}
	}

	return PaginatedPendingClaims{Claims: claims, Total: total, Page: page, PageSize: pageSize}, nil
}

// ClaimsByJournalist lists every claim tagged across every one of the given
// journalist's own articles, newest article first — the dashboard's source
// for letting a journalist self-correct a claim on any of their articles, not
// just the one most recently published.
func ClaimsByJournalist(ctx context.Context, db *pgxpool.Pool, journalistID string) ([]JournalistClaim, error) {
	rows, err := db.Query(ctx, `
		SELECT c.id, c.article_id, a.title AS article_title, c.text, c.tag, c.status
		FROM claims c
		JOIN articles a ON a.id = c.article_id
		WHERE a.journalist_id = $1
		ORDER BY a.created_at DESC, c.created_at ASC
	`, journalistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	claims, err := pgx.CollectRows(rows, pgx.RowToStructByName[JournalistClaim])
	if err != nil {
		return nil, err
	}
	if claims == nil {
		claims = []JournalistClaim{}
	}
	return claims, nil
}

// SelfCorrectClaim lets a journalist mark their own claim as self-corrected
// before auditors resolve it, optionally rewriting the claim's text/tag in
// the same action. SRS formula (1) weighs self-correction (w2) higher than
// baseline verification (w1) to reward getting ahead of a mistake -- and a
// "correction" in the journalistic sense means fixing what the claim actually
// says, not just relabelling it, so the edit and the status change are one
// action rather than two. Passing an empty string for newText/newTag leaves
// that field unchanged.
func SelfCorrectClaim(ctx context.Context, db *pgxpool.Pool, rdb *redis.Client, journalistID, claimID, newText, newTag string) error {
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

	if _, err := db.Exec(ctx, `
		UPDATE claims
		SET status = 'self_corrected',
		    text   = CASE WHEN $2 <> '' THEN $2 ELSE text END,
		    tag    = CASE WHEN $3 <> '' THEN $3 ELSE tag END
		WHERE id = $1
	`, claimID, newText, newTag); err != nil {
		return err
	}

	authorID, rankScore, err := BumpArticleCounterAndRecalculate(ctx, db, claimID, "self_corrected_claims")
	if err != nil {
		return err
	}

	return UpdateLeaderboardScore(ctx, rdb, authorID, rankScore)
}
