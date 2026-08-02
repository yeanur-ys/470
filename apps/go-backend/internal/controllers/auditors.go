package controllers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yeanur-ys/nextGENjournalism/apps/go-backend/internal/auth"
	"github.com/yeanur-ys/nextGENjournalism/apps/go-backend/internal/models"
)

type AuditorsController struct {
	DB *pgxpool.Pool
}

func NewAuditorsController(db *pgxpool.Pool) *AuditorsController {
	return &AuditorsController{DB: db}
}

// Me returns the requesting auditor's own standing for their personal
// dashboard (verification status, reputation, trust weight, vote record).
func (c *AuditorsController) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims == nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	stats, err := models.GetAuditorStats(r.Context(), c.DB, claims.UserID)
	if err != nil {
		if errors.Is(err, models.ErrAuditorNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, "failed to load your auditor profile", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats)
}

// Pending lists auditor accounts awaiting admin credential review (NFR-6).
func (c *AuditorsController) Pending(w http.ResponseWriter, r *http.Request) {
	result, err := models.PendingAuditors(r.Context(), c.DB)
	if err != nil {
		http.Error(w, "failed to load pending auditors", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// Verify approves an auditor's linked credentials, granting them voting rights.
func (c *AuditorsController) Verify(w http.ResponseWriter, r *http.Request) {
	auditorID := r.PathValue("auditorId")

	if err := models.VerifyAuditor(r.Context(), c.DB, auditorID); err != nil {
		if errors.Is(err, models.ErrAuditorNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, "failed to verify auditor", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
