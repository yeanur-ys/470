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

	var title, body, journalistID string
	err := h.DB.QueryRow(ctx, `
		SELECT title, body, journalist_id FROM articles WHERE id = $1
	`, articleID).Scan(&title, &body, &journalistID)
	if err != nil {
		http.Error(w, "article not found", http.StatusNotFound)
		return
	}

	hash := tombstoneHash(title, body)
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		http.Error(w, "failed to start transaction", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(ctx)

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
	if _, err := tx.Exec(ctx, `
		UPDATE users SET rank_score = rank_score - 2 WHERE id = $1
	`, journalistID); err != nil {
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
