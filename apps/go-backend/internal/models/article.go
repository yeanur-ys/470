package models

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/yeanur-ys/nextGENjournalism/apps/go-backend/internal/redisstore"
)

// Article mirrors the articles table and is reused directly as the JSON
// response for every read endpoint — the same convention already used by
// User/PendingAuditor above.
type Article struct {
	ID                  string  `json:"id"`
	JournalistID        string  `json:"journalistId"`
	ParentArticleID     *string `json:"parentArticleId,omitempty"`
	Title               string  `json:"title"`
	Body                string  `json:"body"`
	Signature           string  `json:"signature"`
	ReadershipVolume    int64   `json:"readershipVolume"`
	VerifiedClaims      int     `json:"verifiedClaims"`
	SelfCorrectedClaims int     `json:"selfCorrectedClaims"`
	FalseClaims         int     `json:"falseClaims"`
	IsRetracted         bool    `json:"isRetracted"`
	// Only populated by ArticlesByJournalist — an article can carry at most
	// one active appeal at a time (uniq_active_appeal_per_article), so this is
	// what lets the appeals form warn before a doomed second attempt on an
	// article that already has one open.
	HasActiveAppeal bool      `json:"hasActiveAppeal,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
}

const articleColumns = `id, journalist_id, parent_article_id, title, body, signature,
	       readership_volume, verified_claims, self_corrected_claims, false_claims,
	       is_retracted, created_at`

// defaultArticlesPageSize/maxArticlesPageSize bound the public reader listing's
// page size — the same denial-of-service reasoning as ClampGraphLimit (below,
// graph.go). The default of 100 preserves the endpoint's old hardcoded LIMIT
// 100 for any caller that doesn't ask for a specific page size (the admin
// compliance ledger wants "everything", not 20 at a time).
const (
	defaultArticlesPageSize = 100
	maxArticlesPageSize     = 100
)

// ClampArticlesPageSize turns a raw ?pageSize= query value into a bounded int.
func ClampArticlesPageSize(raw string) int {
	if raw == "" {
		return defaultArticlesPageSize
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return defaultArticlesPageSize
	}
	if n > maxArticlesPageSize {
		return maxArticlesPageSize
	}
	return n
}

// ClampArticlesPage turns a raw ?page= query value into a 1-based page number.
func ClampArticlesPage(raw string) int {
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 1
	}
	return n
}

// PaginatedArticles is the public reader listing's response shape: one page
// of stories plus enough information (total, page, pageSize) for the client
// to render "page N of M" / a Next control without a second round trip.
type PaginatedArticles struct {
	Articles []Article `json:"articles"`
	Total    int       `json:"total"`
	Page     int       `json:"page"`
	PageSize int       `json:"pageSize"`
}

// ListArticles is the public reader listing, paginated newest-first. Readers
// get 20 at a time (F-list pagination); the admin compliance ledger asks for
// pageSize=100 to see everything on one screen, matching the endpoint's old
// hardcoded behaviour before pagination existed.
func ListArticles(ctx context.Context, db *pgxpool.Pool, page, pageSize int) (PaginatedArticles, error) {
	var total int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM articles`).Scan(&total); err != nil {
		return PaginatedArticles{}, err
	}

	// pgx.RowToStructByName requires every exported Article field to have a
	// matching returned column, including HasActiveAppeal -- which only
	// ArticlesByJournalist's query actually computes. The public listing has
	// no single "requesting journalist" to scope an appeal check to, so it's
	// hardcoded false here (also its zero value, so this is a no-op for the
	// JSON response, purely there to satisfy the scan).
	rows, err := db.Query(ctx, `
		SELECT `+articleColumns+`, false AS has_active_appeal
		FROM articles
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, pageSize, (page-1)*pageSize)
	if err != nil {
		return PaginatedArticles{}, err
	}
	defer rows.Close()

	articles, err := pgx.CollectRows(rows, pgx.RowToStructByName[Article])
	if err != nil {
		return PaginatedArticles{}, err
	}
	if articles == nil {
		articles = []Article{}
	}

	return PaginatedArticles{Articles: articles, Total: total, Page: page, PageSize: pageSize}, nil
}

// ArticlesByJournalist implements the journalist dashboard's article list
// (FR-1: each journalist manages their own graph of articles).
func ArticlesByJournalist(ctx context.Context, db *pgxpool.Pool, journalistID string) ([]Article, error) {
	rows, err := db.Query(ctx, `
		SELECT `+articleColumns+`,
		       EXISTS(
		         SELECT 1 FROM appeals ap WHERE ap.article_id = articles.id AND ap.status = 'active'
		       ) AS has_active_appeal
		FROM articles
		WHERE journalist_id = $1
		ORDER BY created_at DESC
	`, journalistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return pgx.CollectRows(rows, pgx.RowToStructByName[Article])
}

// ArticleClaim is the narrow claim projection an article-detail page needs —
// distinct from the full claims.Claim entity (added in the claims migration
// step), which carries fields (ownership, timestamps) a reader has no
// business seeing.
type ArticleClaim struct {
	ID     string `json:"id"`
	Text   string `json:"text"`
	Tag    string `json:"tag"`
	Status string `json:"status"`
}

type ArticleDetail struct {
	Article
	Claims []ArticleClaim `json:"claims"`
}

var ErrArticleNotFound = errors.New("article not found")

// GetArticleDetail is the public reading endpoint: no account required.
// Anyone can load a story and see the verdict on every claim tagged inside
// it — that's the whole point of a transparent, trustless evaluation
// architecture (SRS 1.2).
func GetArticleDetail(ctx context.Context, db *pgxpool.Pool, articleID string) (ArticleDetail, error) {
	row := db.QueryRow(ctx, `SELECT `+articleColumns+` FROM articles WHERE id = $1`, articleID)

	var a Article
	if err := row.Scan(
		&a.ID, &a.JournalistID, &a.ParentArticleID, &a.Title, &a.Body, &a.Signature,
		&a.ReadershipVolume, &a.VerifiedClaims, &a.SelfCorrectedClaims, &a.FalseClaims,
		&a.IsRetracted, &a.CreatedAt,
	); err != nil {
		return ArticleDetail{}, ErrArticleNotFound
	}

	// The tombstone written by RetractArticle stores an internal hash in
	// `body` for auditability — readers should see a plain notice, not that
	// raw format.
	if a.IsRetracted {
		a.Title = "[This story was retracted]"
		a.Body = "This story was removed following a valid legal retraction request (GDPR/DMCA). " +
			"It remains listed, greyed out, to preserve the historical record — see the lineage graph."
	}

	claimRows, err := db.Query(ctx, `
		SELECT id, text, tag, status FROM claims WHERE article_id = $1 ORDER BY created_at ASC
	`, articleID)
	if err != nil {
		return ArticleDetail{}, err
	}
	defer claimRows.Close()

	claims, err := pgx.CollectRows(claimRows, pgx.RowToStructByName[ArticleClaim])
	if err != nil {
		return ArticleDetail{}, err
	}

	return ArticleDetail{Article: a, Claims: claims}, nil
}

// NewArticleInput is Create's request shape as a Model-layer input.
type NewArticleInput struct {
	Title           string
	Body            string
	Signature       string
	ParentArticleID *string
}

// CreateArticle implements FR-3 (claim tagging happens separately through
// the claims endpoints) and FR-4 (Sequence Stitching via ParentArticleID).
func CreateArticle(ctx context.Context, db *pgxpool.Pool, journalistID string, in NewArticleInput) (string, error) {
	var id string
	err := db.QueryRow(ctx, `
		INSERT INTO articles (journalist_id, parent_article_id, title, body, signature)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, journalistID, in.ParentArticleID, in.Title, in.Body, in.Signature).Scan(&id)
	return id, err
}

// RecordArticleRead implements the readership-volume side of FR-12: Postgres
// remains the source of truth for readership_volume (and flows to Neo4j via
// CDC so node size can scale in the graph), while Redis holds the same
// counter for instant, high-frequency reads (NFR-3) without hammering
// Postgres.
func RecordArticleRead(ctx context.Context, db *pgxpool.Pool, rdb *redis.Client, articleID string) error {
	if _, err := db.Exec(ctx, `
		UPDATE articles SET readership_volume = readership_volume + 1 WHERE id = $1
	`, articleID); err != nil {
		return err
	}

	if rdb != nil {
		// Best-effort: a dropped Redis increment doesn't lose data, Postgres
		// above is still the source of truth.
		rdb.Incr(ctx, redisstore.ArticleReadsKey(articleID))
	}
	return nil
}

// Appeal mirrors the appeals table (FR-5/F-14).
type Appeal struct {
	ID            string     `json:"id"`
	ArticleID     string     `json:"articleId"`
	JournalistID  string     `json:"journalistId"`
	StakedPercent float64    `json:"stakedPercent"`
	EvidenceText  *string    `json:"evidenceText,omitempty"`
	EvidenceURL   *string    `json:"evidenceUrl,omitempty"`
	Status        string     `json:"status"`
	CreatedAt     time.Time  `json:"createdAt"`
	ResolvedAt    *time.Time `json:"resolvedAt,omitempty"`
}

var (
	ErrNotArticleOwner     = errors.New("you can only appeal rulings on your own articles")
	ErrMissingAppealInputs = errors.New("articleId and a positive stakedPercent are required")
	ErrStakePercentRange   = errors.New("stakedPercent must be between 0 and 100")
	ErrMissingEvidence     = errors.New("an appeal must include new evidence (evidenceText or evidenceUrl)")
	ErrAppealAlreadyActive = errors.New("you already have an active appeal on this article")
)

// NewAppealInput is CreateAppeal's request shape as a Model-layer input.
type NewAppealInput struct {
	ArticleID     string
	StakedPercent float64
	EvidenceText  string
	EvidenceURL   string
}

func validateNewAppeal(in NewAppealInput) error {
	if in.ArticleID == "" || in.StakedPercent <= 0 {
		return ErrMissingAppealInputs
	}
	if in.StakedPercent > 100 {
		return ErrStakePercentRange
	}
	// F-14: "Proof of Evidence" — an appeal with no new evidence is just a
	// complaint. One of the two evidence fields must carry something.
	if strings.TrimSpace(in.EvidenceText) == "" && strings.TrimSpace(in.EvidenceURL) == "" {
		return ErrMissingEvidence
	}
	return nil
}

// CreateAppeal implements FR-5/F-14, the Proof of Evidence Appeals Protocol:
// a journalist disputes a ruling on their own article by submitting new
// primary evidence and staking a percentage of their rank score. FR-9's
// pulsing amber UI state is driven by appeals.status == 'active'.
//
// The stake is deducted immediately rather than merely recorded — an appeal
// that costs nothing to file isn't a stake, and F-14's whole premise is that
// the journalist has skin in the game. Returns the rank score actually
// staked.
func CreateAppeal(ctx context.Context, db *pgxpool.Pool, journalistID string, in NewAppealInput) (float64, error) {
	if err := validateNewAppeal(in); err != nil {
		return 0, err
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	// Ownership check: without it any journalist could file appeals against
	// any article in the system, including a competitor's, and pin their
	// node into the "under dispute" state (FR-9) indefinitely.
	var ownerID string
	if err := tx.QueryRow(ctx, `SELECT journalist_id FROM articles WHERE id = $1`, in.ArticleID).Scan(&ownerID); err != nil {
		return 0, ErrArticleNotFound
	}
	if ownerID != journalistID {
		return 0, ErrNotArticleOwner
	}

	var rankScore float64
	if err := tx.QueryRow(ctx, `SELECT rank_score FROM users WHERE id = $1 FOR UPDATE`, journalistID).Scan(&rankScore); err != nil {
		return 0, err
	}
	staked := rankScore * (in.StakedPercent / 100)

	// uniq_active_appeal_per_article turns a second concurrent appeal into a
	// unique-violation here rather than a duplicate amber node.
	if _, err := tx.Exec(ctx, `
		INSERT INTO appeals (article_id, journalist_id, staked_percent, evidence_text, evidence_url)
		VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''))
	`, in.ArticleID, journalistID, in.StakedPercent, in.EvidenceText, in.EvidenceURL); err != nil {
		return 0, ErrAppealAlreadyActive
	}

	if _, err := tx.Exec(ctx,
		`UPDATE users SET rank_score = GREATEST(rank_score - $2, 0) WHERE id = $1`,
		journalistID, staked,
	); err != nil {
		return 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return staked, nil
}

// RetractionBasePenalty is the fixed component of the FR-15 rank deduction;
// the reach-scaled component is added on top at retraction time.
const RetractionBasePenalty = 2.0

// tombstoneHash preserves an auditable fingerprint of the original content
// without retaining the identifying text itself.
func tombstoneHash(title, body string) string {
	sum := sha256.Sum256([]byte(title + "\x00" + body))
	return hex.EncodeToString(sum[:])
}

// RetractionResult reports whether this call actually applied the tombstone
// or found the article already retracted (idempotent no-op).
type RetractionResult struct {
	AlreadyRetracted bool
	TombstoneHash    string
}

// RetractArticle implements FR-13/FR-14/FR-15: a valid legal request
// (GDPR/DMCA) replaces identifying content with a cryptographic tombstone,
// keeps the node present (greyed-out) for historical continuity, and
// permanently deducts rank score from the article's author. Hard deletion is
// refused once the article's readership has crossed the configured
// immutability threshold (enforced by a DB trigger, not here).
//
// Touches both articles and users in one transaction — it introduces no new
// entity of its own, only this method spanning the two that already exist.
func RetractArticle(ctx context.Context, db *pgxpool.Pool, articleID string) (RetractionResult, error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return RetractionResult{}, err
	}
	defer tx.Rollback(ctx)

	// SELECT ... FOR UPDATE inside the same transaction that writes: reading
	// on the pool then opening a separate transaction to write let two
	// concurrent retract calls both read is_retracted = false and both apply
	// the rank penalty. Re-checking is_retracted under the row lock makes the
	// operation idempotent — a second retraction of the same article is a
	// no-op rather than another permanent deduction from the author's score.
	var title, body, journalistID string
	var alreadyRetracted bool
	if err := tx.QueryRow(ctx, `
		SELECT title, body, journalist_id, is_retracted FROM articles WHERE id = $1 FOR UPDATE
	`, articleID).Scan(&title, &body, &journalistID, &alreadyRetracted); err != nil {
		return RetractionResult{}, ErrArticleNotFound
	}
	if alreadyRetracted {
		return RetractionResult{AlreadyRetracted: true}, nil
	}

	hash := tombstoneHash(title, body)

	if _, err := tx.Exec(ctx, `
		UPDATE articles
		SET title = '[retracted]',
		    body = $2,
		    is_retracted = true,
		    retracted_at = $3
		WHERE id = $1
	`, articleID, "tombstone:"+hash, time.Now()); err != nil {
		return RetractionResult{}, err
	}

	// FR-15: permanent rank-score deduction for the retracted article's
	// author. Scaled by reach rather than a flat constant — a retracted story
	// that 100,000 people read did more damage than one that 12 people read,
	// and log10 keeps that proportionate on the same scale the Rank Score
	// itself uses for readership (SRS formula 1's log10(1+V) dampener).
	if _, err := tx.Exec(ctx, `
		UPDATE users u
		SET rank_score = GREATEST(u.rank_score - ($2 + log(10, 1 + a.readership_volume)), 0)
		FROM articles a
		WHERE u.id = $1 AND a.id = $3
	`, journalistID, RetractionBasePenalty, articleID); err != nil {
		return RetractionResult{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return RetractionResult{}, err
	}
	return RetractionResult{TombstoneHash: hash}, nil
}
