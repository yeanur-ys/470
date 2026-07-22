package compliance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RetractionBasePenalty is the fixed component of the FR-15 rank deduction;
// the reach-scaled component is added on top at retraction time.
const RetractionBasePenalty = 2.0

type Handler struct {
	DB *pgxpool.Pool
}

func NewHandler(db *pgxpool.Pool) *Handler {
	return &Handler{DB: db}
}

// tombstoneHash preserves an auditable fingerprint of the original content
// without retaining the identifying text itself.
func tombstoneHash(title, body string) string {
	sum := sha256.Sum256([]byte(title + "\x00" + body))
	return hex.EncodeToString(sum[:])
}

// Retract implements FR-13/FR-14/FR-15: a valid legal request (GDPR/DMCA)
// replaces identifying content with a cryptographic tombstone, keeps the node
// present (greyed-out) for historical continuity, and permanently deducts
// rank score from the article's author. Hard deletion is refused once the
// article's readership has crossed the configured immutability threshold.
func (h *Handler) Retract(w http.ResponseWriter, r *http.Request) {
	articleID := r.PathValue("articleId")
	ctx := context.Background()

	tx, err := h.DB.Begin(ctx)
	if err != nil {
		http.Error(w, "failed to start transaction", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(ctx)

	// SELECT ... FOR UPDATE inside the same transaction that writes: the
	// previous version read the article on the pool, then opened a separate
	// transaction to write, so two concurrent retract calls could both read
	// is_retracted = false and both apply the rank penalty. Re-checking
	// is_retracted under the row lock makes the operation idempotent — a
	// second retraction of the same article is now a no-op rather than
	// another permanent deduction from the author's score.
	var title, body, journalistID string
	var alreadyRetracted bool
	if err := tx.QueryRow(ctx, `
		SELECT title, body, journalist_id, is_retracted FROM articles WHERE id = $1 FOR UPDATE
	`, articleID).Scan(&title, &body, &journalistID, &alreadyRetracted); err != nil {
		http.Error(w, "article not found", http.StatusNotFound)
		return
	}
	if alreadyRetracted {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "already_retracted",
			"detail": "this article was already tombstoned; no further penalty applied",
		})
		return
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
		http.Error(w, "failed to apply tombstone", http.StatusInternalServerError)
		return
	}

	// FR-15: permanent rank-score deduction for the retracted article's author.
	// Scaled by reach rather than a flat constant — a retracted story that
	// 100,000 people read did more damage than one that 12 people read, and
	// log10 keeps that proportionate on the same scale the Rank Score itself
	// uses for readership (SRS formula 1's log10(1+V) dampener).
	if _, err := tx.Exec(ctx, `
		UPDATE users u
		SET rank_score = GREATEST(u.rank_score - ($2 + log(10, 1 + a.readership_volume)), 0)
		FROM articles a
		WHERE u.id = $1 AND a.id = $3
	`, journalistID, RetractionBasePenalty, articleID); err != nil {
		http.Error(w, "failed to apply rank penalty", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		http.Error(w, "failed to commit retraction", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "retracted", "tombstoneHash": hash})
}
