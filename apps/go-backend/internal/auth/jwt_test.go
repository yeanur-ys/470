package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestTokenService_IssueAndParseRoundTrip(t *testing.T) {
	tokens := NewTokenService("test-secret")

	signed, err := tokens.Issue("user-123", "journalist")
	if err != nil {
		t.Fatalf("Issue() returned an error: %v", err)
	}

	claims, err := tokens.Parse(signed)
	if err != nil {
		t.Fatalf("Parse() returned an error for a token we just issued: %v", err)
	}

	if claims.UserID != "user-123" {
		t.Errorf("UserID = %q, want %q", claims.UserID, "user-123")
	}
	if claims.Role != "journalist" {
		t.Errorf("Role = %q, want %q", claims.Role, "journalist")
	}
}

func TestTokenService_RejectsWrongSecret(t *testing.T) {
	issuer := NewTokenService("correct-secret")
	verifier := NewTokenService("different-secret")

	signed, err := issuer.Issue("user-123", "journalist")
	if err != nil {
		t.Fatalf("Issue() returned an error: %v", err)
	}

	// This is the scenario RequireRole ultimately depends on: a token signed
	// with any secret other than this server's must never verify — otherwise
	// the whole role-guard system (server.RequireRole) is only as strong as
	// an attacker's ability to sign their own "admin" claim.
	if _, err := verifier.Parse(signed); err == nil {
		t.Fatal("Parse() accepted a token signed with a different secret")
	}
}

func TestTokenService_RejectsExpiredToken(t *testing.T) {
	tokens := NewTokenService("test-secret")

	claims := Claims{
		UserID: "user-123",
		Role:   "journalist",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)), // already expired
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("failed to sign test token: %v", err)
	}

	if _, err := tokens.Parse(signed); err == nil {
		t.Fatal("Parse() accepted an expired token")
	}
}

func TestTokenService_RejectsTamperedPayload(t *testing.T) {
	tokens := NewTokenService("test-secret")

	signed, err := tokens.Issue("user-123", "journalist")
	if err != nil {
		t.Fatalf("Issue() returned an error: %v", err)
	}

	// Flip a character in the payload segment (index 1 of the three
	// dot-separated JWT segments) without re-signing, simulating an
	// attacker trying to smuggle a different role past a stolen signature.
	parts := strings.Split(signed, ".")
	if len(parts) != 3 {
		t.Fatalf("expected a 3-segment JWT, got %d segments", len(parts))
	}
	tampered := parts[0] + "." + parts[1] + "x" + "." + parts[2]

	if _, err := tokens.Parse(tampered); err == nil {
		t.Fatal("Parse() accepted a token with a tampered payload")
	}
}
