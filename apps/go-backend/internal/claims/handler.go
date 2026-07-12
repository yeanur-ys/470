package claims

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

type createClaimRequest struct {
	Text string `json:"text"`
	Tag  string `json:"tag"`
}

// Create implements FR-3: journalists encapsulate specific statements in
// #Claim tags at publish time (or afterwards) so auditors can vote on them.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	articleID := r.PathValue("articleId")

	var req createClaimRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Text == "" || req.Tag == "" {
		http.Error(w, "text and tag are required", http.StatusBadRequest)
		return
	}

	var id string
	err := h.DB.QueryRow(context.Background(), `
		INSERT INTO claims (article_id, text, tag) VALUES ($1, $2, $3) RETURNING id
	`, articleID, req.Text, req.Tag).Scan(&id)
	if err != nil {
		http.Error(w, "failed to create claim", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"id": id})
}

// Pending lists claims awaiting cross-tag consensus (FR-7), for auditors to
// pick up and vote on.
func (h *Handler) Pending(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(context.Background(), `
		SELECT c.id, c.article_id, a.title AS article_title, c.text, c.tag
		FROM claims c
		JOIN articles a ON a.id = c.article_id
		WHERE c.status = 'pending'
		ORDER BY c.created_at ASC
		LIMIT 100
	`)
	if err != nil {
		http.Error(w, "failed to load claims", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[PendingClaim])
	if err != nil {
		http.Error(w, "failed to read claims", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}
