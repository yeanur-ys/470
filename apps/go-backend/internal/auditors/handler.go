package auditors

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	DB *pgxpool.Pool
}

func NewHandler(db *pgxpool.Pool) *Handler {
	return &Handler{DB: db}
}

// Pending lists auditor accounts whose linked credentials an admin hasn't
// approved yet. Until approved, that auditor's votes are rejected (see
// consensus.Handler.Vote), which is how NFR-6 (Sybil resistance) is enforced.
func (h *Handler) Pending(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(context.Background(), `
		SELECT id, email, display_name, COALESCE(credential_url, '') AS credential_url, tags
		FROM users
		WHERE role = 'auditor' AND credential_verified = false
		ORDER BY created_at ASC
	`)
	if err != nil {
		http.Error(w, "failed to load pending auditors", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[PendingAuditor])
	if err != nil {
		http.Error(w, "failed to read pending auditors", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// Verify approves an auditor's linked credentials, granting them voting rights.
func (h *Handler) Verify(w http.ResponseWriter, r *http.Request) {
	auditorID := r.PathValue("auditorId")

	tag, err := h.DB.Exec(context.Background(), `
		UPDATE users SET credential_verified = true WHERE id = $1 AND role = 'auditor'
	`, auditorID)
	if err != nil {
		http.Error(w, "failed to verify auditor", http.StatusInternalServerError)
		return
	}
	if tag.RowsAffected() == 0 {
		http.Error(w, "auditor not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
