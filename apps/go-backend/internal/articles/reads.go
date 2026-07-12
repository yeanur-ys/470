package articles

import (
	"context"
	"net/http"

	"github.com/yeanur-ys/nextGENjournalism/apps/go-backend/internal/redisstore"
)

// RecordRead implements the readership-volume side of FR-12: Postgres remains
// the source of truth for readership_volume (and flows to Neo4j via CDC so
// node size can scale in the graph), while Redis holds the same counter for
// instant, high-frequency reads (NFR-3) without hammering Postgres.
func (h *Handler) RecordRead(w http.ResponseWriter, r *http.Request) {
	articleID := r.PathValue("articleId")
	ctx := context.Background()

	if _, err := h.DB.Exec(ctx, `
		UPDATE articles SET readership_volume = readership_volume + 1 WHERE id = $1
	`, articleID); err != nil {
		http.Error(w, "failed to record read", http.StatusInternalServerError)
		return
	}

	if h.Redis != nil {
		// Best-effort: a dropped Redis increment doesn't lose data, Postgres
		// above is still the source of truth.
		h.Redis.Incr(ctx, redisstore.ArticleReadsKey(articleID))
	}

	w.WriteHeader(http.StatusNoContent)
}
