package auth

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

const pgUniqueViolation = "23505"

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

type signupRequest struct {
	Email         string   `json:"email"`
	Password      string   `json:"password"`
	DisplayName   string   `json:"displayName"`
	Role          string   `json:"role"`          // "journalist" or "auditor" only — see below
	CredentialURL string   `json:"credentialUrl"` // required for auditors, NFR-6
	Tags          []string `json:"tags"`           // auditor category tags, required for auditors
}

// Signup implements FR-1/FR-2 self-registration. Admin accounts are
// intentionally excluded — see README step 2 for provisioning those
// directly, since self-serve admin signup would defeat the point of having
// a trusted compliance role at all. Auditors are created with
// credential_verified = false (NFR-6): they can sign in immediately but
// can't cast a vote until an admin approves their linked credentials
// (see auditors.Handler.Verify).
//
// Readers never hit this endpoint at all — reading is public (see
// articles.Handler.List / .Get) and requires no account.
func (h *Handler) Signup(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req signupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Email == "" || req.Password == "" || req.DisplayName == "" {
		http.Error(w, "email, password and displayName are required", http.StatusBadRequest)
		return
	}
	if len(req.Password) < 8 {
		http.Error(w, "password must be at least 8 characters", http.StatusBadRequest)
		return
	}
	if req.Role != "journalist" && req.Role != "auditor" {
		http.Error(w, `role must be "journalist" or "auditor"`, http.StatusBadRequest)
		return
	}
	if req.Role == "auditor" && (req.CredentialURL == "" || len(req.Tags) == 0) {
		http.Error(w, "auditors must provide a credentialUrl and at least one category tag", http.StatusBadRequest)
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "could not process password", http.StatusInternalServerError)
		return
	}

	credentialVerified := req.Role != "auditor" // journalists don't need this gate at all

	// users.tags is NOT NULL; a nil Go slice (the common case for journalists,
	// who never send a tags field) would otherwise be sent as SQL NULL and
	// fail the constraint on every single signup, not just duplicates.
	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}

	var userID string
	err = h.DB.QueryRow(context.Background(), `
		INSERT INTO users (email, password_hash, role, display_name, credential_url, credential_verified, tags)
		VALUES ($1, $2, $3::user_role, $4, NULLIF($5, ''), $6, $7)
		RETURNING id
	`, req.Email, string(passwordHash), req.Role, req.DisplayName, req.CredentialURL, credentialVerified, tags).Scan(&userID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			http.Error(w, "an account with that email already exists", http.StatusConflict)
			return
		}
		// Anything else (bad enum value, constraint violation, connection
		// drop, ...) is a real server-side problem — log it with detail and
		// tell the caller honestly, instead of guessing "already exists".
		log.Printf("signup failed for %s: %v", req.Email, err)
		http.Error(w, "could not create the account — see server logs", http.StatusInternalServerError)
		return
	}

	token, err := h.Tokens.Issue(userID, req.Role)
	if err != nil {
		http.Error(w, "account created, but could not issue a session — please log in", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(loginResponse{Token: token, Role: req.Role, UserID: userID})
}
