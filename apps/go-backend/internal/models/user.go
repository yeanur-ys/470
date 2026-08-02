package models

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

const pgUniqueViolation = "23505"

// User mirrors the full users table (packages/database/postgres/schema.sql),
// not just the fields any one caller happens to need — later domains
// (auditors, leaderboard, consensus) add methods against this same struct
// without ever having to touch its field set again.
type User struct {
	ID                 string
	Email              string
	PasswordHash       string
	Role               string
	DisplayName        string
	RankScore          float64
	CredentialURL      string
	CredentialVerified bool
	Tags               []string
	TrustWeight        float64
	SuccessfulVotes    int
	FailedVotes        int
	LockedStake        float64
	CreatedAt          time.Time
}

var (
	ErrInvalidCredentials        = errors.New("invalid credentials")
	ErrEmailTaken                = errors.New("an account with that email already exists")
	ErrPasswordTooShort          = errors.New("password must be at least 8 characters")
	ErrInvalidRole               = errors.New(`role must be "journalist" or "auditor"`)
	ErrAuditorCredentialsMissing = errors.New("auditors must provide a credentialUrl and at least one category tag")
	ErrAuditorNotFound           = errors.New("auditor not found")
)

// normalizeEmail matches how the address is actually used as an identifier:
// case-insensitively, with no surrounding whitespace. Without this, the raw
// TEXT UNIQUE column let "Jane@Example.com" and "jane@example.com" collide at
// login (bcrypt.CompareHashAndPassword never even runs — the SELECT just
// finds no row) despite being the same account by any reasonable definition,
// and mobile keyboards autocapitalize the first letter often enough that this
// isn't an edge case. Called from inside AuthenticateUser/CreateUser rather
// than left for callers to remember, so it can never be skipped at one call
// site and not another.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// AuthenticateUser implements FR-1/FR-2 login: verified account access for
// journalists, auditors and admins. Returns ErrInvalidCredentials for both a
// missing account and a wrong password — the caller must not be able to
// distinguish the two from the response.
func AuthenticateUser(ctx context.Context, db *pgxpool.Pool, email, password string) (User, error) {
	var u User
	err := db.QueryRow(ctx,
		`SELECT id, role, password_hash FROM users WHERE email = $1`, normalizeEmail(email),
	).Scan(&u.ID, &u.Role, &u.PasswordHash)
	if err != nil {
		return User{}, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return User{}, ErrInvalidCredentials
	}
	return u, nil
}

// NewUserInput is Signup's request shape as a Model-layer input — decoding
// JSON stays the Controller's job, but the fields below are what CreateUser
// actually needs to enforce signup's domain rules and write the row.
type NewUserInput struct {
	Email         string
	Password      string
	DisplayName   string
	Role          string // "journalist" or "auditor" only
	CredentialURL string // required for auditors, NFR-6
	Tags          []string
}

// validateNewUser is pure — no ctx, no DB — so it's directly unit-testable
// the same way ranking/calculator_test.go and consensus/voting_test.go
// already test this codebase's other pure business rules.
func validateNewUser(in NewUserInput) error {
	if len(in.Password) < 8 {
		return ErrPasswordTooShort
	}
	if in.Role != "journalist" && in.Role != "auditor" {
		return ErrInvalidRole
	}
	if in.Role == "auditor" && (in.CredentialURL == "" || len(in.Tags) == 0) {
		return ErrAuditorCredentialsMissing
	}
	return nil
}

// CreateUser implements FR-1/FR-2 self-registration. Admin accounts are
// intentionally excluded — see README step 2 for provisioning those
// directly, since self-serve admin signup would defeat the point of having a
// trusted compliance role at all. Auditors are created with
// credential_verified = false (NFR-6): they can sign in immediately but
// can't cast a vote until an admin approves their linked credentials (see
// VerifyAuditor).
func CreateUser(ctx context.Context, db *pgxpool.Pool, in NewUserInput) (User, error) {
	in.Email = normalizeEmail(in.Email)
	if err := validateNewUser(in); err != nil {
		return User{}, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, err
	}

	credentialVerified := in.Role != "auditor" // journalists don't need this gate at all

	// tags is NOT NULL; a nil Go slice (the common case for journalists, who
	// never send a tags field) would otherwise be sent as SQL NULL and fail
	// the constraint on every single signup, not just duplicates.
	tags := in.Tags
	if tags == nil {
		tags = []string{}
	}

	u := User{Email: in.Email, Role: in.Role, DisplayName: in.DisplayName, CredentialVerified: credentialVerified}
	err = db.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, role, display_name, credential_url, credential_verified, tags)
		VALUES ($1, $2, $3::user_role, $4, NULLIF($5, ''), $6, $7)
		RETURNING id
	`, in.Email, string(passwordHash), in.Role, in.DisplayName, in.CredentialURL, credentialVerified, tags).Scan(&u.ID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return User{}, ErrEmailTaken
		}
		// Anything else (bad enum value, constraint violation, connection
		// drop, ...) is a real server-side problem — log it with detail here
		// (the email is in scope) and let the caller get a generic message.
		log.Printf("signup failed for %s: %v", in.Email, err)
		return User{}, err
	}
	return u, nil
}

// Login rate limiting (NFR-6-adjacent hardening, not a numbered SRS
// requirement): bcrypt makes each guess expensive, which is a real mitigation
// on its own, but it also makes login a cheap way to burn CPU by just firing
// unlimited attempts. Capped per IP over a fixed window instead. This is a
// domain rule about login, not about parsing a request, so it lives here;
// extracting the IP from *http.Request is the Controller's job.
const (
	loginRateLimitWindow = 5 * time.Minute
	loginRateLimitMax    = 10
)

// CheckLoginRateLimit returns false once an IP has made loginRateLimitMax
// attempts within loginRateLimitWindow. Fails open on a Redis error — an
// unreachable rate limiter shouldn't take login down with it, since bcrypt
// cost is still a real backstop on its own.
func CheckLoginRateLimit(ctx context.Context, rdb *redis.Client, ip string) bool {
	if rdb == nil {
		return true
	}
	key := "ratelimit:login:" + ip
	count, err := rdb.Incr(ctx, key).Result()
	if err != nil {
		log.Printf("login rate limit check failed (allowing request): %v", err)
		return true
	}
	if count == 1 {
		rdb.Expire(ctx, key, loginRateLimitWindow)
	}
	return count <= loginRateLimitMax
}

// PendingAuditor is a projection of User for the admin credential-review
// queue (NFR-6, Sybil resistance). Kept here rather than in views/, and
// scanned straight from SQL via pgx.RowToStructByName, the same convention
// articles.Article and claims.Claim already use elsewhere: the JSON tags
// alone are enough, since pgx's default naming matches Go field names to
// snake_case column aliases without needing a separate db tag.
type PendingAuditor struct {
	ID            string   `json:"id"`
	Email         string   `json:"email"`
	DisplayName   string   `json:"displayName"`
	CredentialURL string   `json:"credentialUrl"`
	Tags          []string `json:"tags"`
}

// PendingAuditors lists auditor accounts whose linked credentials an admin
// hasn't approved yet. Until approved, that auditor's votes are rejected
// (see the consensus vote handler's credential_verified check), which is how
// NFR-6 is enforced.
func PendingAuditors(ctx context.Context, db *pgxpool.Pool) ([]PendingAuditor, error) {
	rows, err := db.Query(ctx, `
		SELECT id, email, display_name, COALESCE(credential_url, '') AS credential_url, tags
		FROM users
		WHERE role = 'auditor' AND credential_verified = false
		ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return pgx.CollectRows(rows, pgx.RowToStructByName[PendingAuditor])
}

// AuditorStats is an auditor's own standing, for their personal dashboard.
// Everything here is about the requesting auditor themselves — distinct from
// PendingAuditor, which is the admin's view of *other* auditors awaiting
// review.
type AuditorStats struct {
	ID                 string   `json:"id"`
	DisplayName        string   `json:"displayName"`
	CredentialVerified bool     `json:"credentialVerified"`
	CredentialURL      string   `json:"credentialUrl"`
	Tags               []string `json:"tags"`
	RankScore          float64  `json:"rankScore"`
	TrustWeight        float64  `json:"trustWeight"`
	SuccessfulVotes    int      `json:"successfulVotes"`
	FailedVotes        int      `json:"failedVotes"`
	VotesCast          int      `json:"votesCast"`      // every vote row, including still-open ones
	LockedStake        float64  `json:"lockedStake"`    // reputation committed to open votes
	AvailableStake     float64  `json:"availableStake"` // rank_score - locked_stake, the max a new vote can stake
}

// GetAuditorStats loads the requesting auditor's own standing. Drives the
// personal auditor dashboard (verification banner, reputation, vote record) so
// an auditor can see *why* they can or can't vote (NFR-6) rather than only
// finding out at the moment a vote is rejected.
func GetAuditorStats(ctx context.Context, db *pgxpool.Pool, auditorID string) (AuditorStats, error) {
	var s AuditorStats
	err := db.QueryRow(ctx, `
		SELECT id, display_name, credential_verified, COALESCE(credential_url, ''), tags,
		       rank_score, trust_weight, successful_votes, failed_votes, locked_stake,
		       (SELECT count(*) FROM votes WHERE auditor_id = u.id) AS votes_cast
		FROM users u
		WHERE id = $1 AND role = 'auditor'
	`, auditorID).Scan(
		&s.ID, &s.DisplayName, &s.CredentialVerified, &s.CredentialURL, &s.Tags,
		&s.RankScore, &s.TrustWeight, &s.SuccessfulVotes, &s.FailedVotes, &s.LockedStake, &s.VotesCast,
	)
	if err != nil {
		return AuditorStats{}, ErrAuditorNotFound
	}
	s.AvailableStake = s.RankScore - s.LockedStake
	if s.AvailableStake < 0 {
		s.AvailableStake = 0
	}
	return s, nil
}

// VerifyAuditor approves an auditor's linked credentials, granting them
// voting rights.
func VerifyAuditor(ctx context.Context, db *pgxpool.Pool, auditorID string) error {
	tag, err := db.Exec(ctx, `
		UPDATE users SET credential_verified = true WHERE id = $1 AND role = 'auditor'
	`, auditorID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrAuditorNotFound
	}
	return nil
}
