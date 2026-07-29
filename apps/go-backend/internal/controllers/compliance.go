package controllers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yeanur-ys/nextGENjournalism/apps/go-backend/internal/models"
)

type ComplianceController struct {
	DB *pgxpool.Pool
}

func NewComplianceController(db *pgxpool.Pool) *ComplianceController {
	return &ComplianceController{DB: db}
}

// Retract implements FR-13/FR-14/FR-15.
func (c *ComplianceController) Retract(w http.ResponseWriter, r *http.Request) {
	articleID := r.PathValue("articleId")

	result, err := models.RetractArticle(r.Context(), c.DB, articleID)
	if err != nil {
		if errors.Is(err, models.ErrArticleNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, "failed to apply retraction", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if result.AlreadyRetracted {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "already_retracted",
			"detail": "this article was already tombstoned; no further penalty applied",
		})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "retracted", "tombstoneHash": result.TombstoneHash})
}
