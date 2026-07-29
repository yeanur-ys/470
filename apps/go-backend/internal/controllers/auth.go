package controllers

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/yeanur-ys/nextGENjournalism/apps/go-backend/internal/auth"
	"github.com/yeanur-ys/nextGENjournalism/apps/go-backend/internal/models"
	"github.com/yeanur-ys/nextGENjournalism/apps/go-backend/internal/views"
)

type AuthController struct {
	DB     *pgxpool.Pool
	Tokens *auth.TokenService
	Redis  *redis.Client
}

func NewAuthController(db *pgxpool.Pool, tokens *auth.TokenService, redisClient *redis.Client) *AuthController {
	return &AuthController{DB: db, Tokens: tokens, Redis: redisClient}
}

// clientIP extracts the request's source address, stripping the port. Good
// enough for a single-instance deployment; a real load balancer setup would
// need to trust a specific X-Forwarded-For hop instead of trusting the
// client-supplied header outright. HTTP-shape work, stays in the Controller —
// the domain rule that uses it (models.CheckLoginRateLimit) doesn't know or
// care what an *http.Request looks like.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login implements FR-1/FR-2: verified account access for journalists, auditors and admins.
func (c *AuthController) Login(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if !models.CheckLoginRateLimit(r.Context(), c.Redis, clientIP(r)) {
		http.Error(w, "too many login attempts — try again in a few minutes", http.StatusTooManyRequests)
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" || req.Password == "" {
		http.Error(w, "email and password are required", http.StatusBadRequest)
		return
	}

	user, err := models.AuthenticateUser(r.Context(), c.DB, req.Email, req.Password)
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := c.Tokens.Issue(user.ID, user.Role)
	if err != nil {
		http.Error(w, "could not issue token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(views.SessionView{Token: token, Role: user.Role, UserID: user.ID})
}

type signupRequest struct {
	Email         string   `json:"email"`
	Password      string   `json:"password"`
	DisplayName   string   `json:"displayName"`
	Role          string   `json:"role"`          // "journalist" or "auditor" only — see below
	CredentialURL string   `json:"credentialUrl"` // required for auditors, NFR-6
	Tags          []string `json:"tags"`          // auditor category tags, required for auditors
}

// Signup implements FR-1/FR-2 self-registration. Readers never hit this
// endpoint at all — reading is public and requires no account.
func (c *AuthController) Signup(w http.ResponseWriter, r *http.Request) {
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

	user, err := models.CreateUser(r.Context(), c.DB, models.NewUserInput{
		Email:         req.Email,
		Password:      req.Password,
		DisplayName:   req.DisplayName,
		Role:          req.Role,
		CredentialURL: req.CredentialURL,
		Tags:          req.Tags,
	})
	if err != nil {
		switch {
		case errors.Is(err, models.ErrEmailTaken):
			http.Error(w, err.Error(), http.StatusConflict)
		case errors.Is(err, models.ErrPasswordTooShort),
			errors.Is(err, models.ErrInvalidRole),
			errors.Is(err, models.ErrAuditorCredentialsMissing):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			// models.CreateUser already logged the detailed error server-side.
			http.Error(w, "could not create the account — see server logs", http.StatusInternalServerError)
		}
		return
	}

	token, err := c.Tokens.Issue(user.ID, user.Role)
	if err != nil {
		http.Error(w, "account created, but could not issue a session — please log in", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(views.SessionView{Token: token, Role: user.Role, UserID: user.ID})
}
