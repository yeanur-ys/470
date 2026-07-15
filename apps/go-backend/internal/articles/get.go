package articles

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5"
)

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

// Get is the public reading endpoint: no account required. Anyone can load a
// story and see the verdict on every claim tagged inside it — that's the
// whole point of a transparent, trustless evaluation architecture (SRS 1.2).
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	articleID := r.PathValue("articleId")
	ctx := context.Background()

	row := h.DB.QueryRow(ctx, `
		SELECT id, journalist_id, parent_article_id, title, body, signature,
		       readership_volume, verified_claims, self_corrected_claims, false_claims,
		       is_retracted, created_at
		FROM articles
		WHERE id = $1
	`, articleID)

	var a Article
	if err := row.Scan(
		&a.ID, &a.JournalistID, &a.ParentArticleID, &a.Title, &a.Body, &a.Signature,
		&a.ReadershipVolume, &a.VerifiedClaims, &a.SelfCorrectedClaims, &a.FalseClaims,
		&a.IsRetracted, &a.CreatedAt,
	); err != nil {
		http.Error(w, "article not found", http.StatusNotFound)
		return
	}

	// The tombstone written by compliance.Handler.Retract stores an internal
	// hash in `body` for auditability — readers should see a plain notice,
	// not that raw format.
	if a.IsRetracted {
		a.Title = "[This story was retracted]"
		a.Body = "This story was removed following a valid legal retraction request (GDPR/DMCA). " +
			"It remains listed, greyed out, to preserve the historical record — see the lineage graph."
	}

	claimRows, err := h.DB.Query(ctx, `
		SELECT id, text, tag, status FROM claims WHERE article_id = $1 ORDER BY created_at ASC
	`, articleID)
	if err != nil {
		http.Error(w, "failed to load claims", http.StatusInternalServerError)
		return
	}
	defer claimRows.Close()

	claims, err := pgx.CollectRows(claimRows, pgx.RowToStructByName[ArticleClaim])
	if err != nil {
		http.Error(w, "failed to read claims", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ArticleDetail{Article: a, Claims: claims})
}
