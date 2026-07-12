package auth

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	DB     *pgxpool.Pool
	Tokens *TokenService
}

func NewHandler(db *pgxpool.Pool, tokens *TokenService) *Handler {
	return &Handler{DB: db, Tokens: tokens}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token  string `json:"token"`
	Role   string `json:"role"`
	UserID string `json:"userId"`
}

// Login implements FR-1/FR-2: verified account access for journalists, auditors and admins.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" || req.Password == "" {
		http.Error(w, "email and password are required", http.StatusBadRequest)
		return
	}

	var userID, role, passwordHash string
	err := h.DB.QueryRow(context.Background(),
		`SELECT id, role, password_hash FROM users WHERE email = $1`, req.Email,
	).Scan(&userID, &role, &passwordHash)
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := h.Tokens.Issue(userID, role)
	if err != nil {
		http.Error(w, "could not issue token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(loginResponse{Token: token, Role: role, UserID: userID})
}
