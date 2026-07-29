package models

import (
	"errors"
	"testing"
)

func TestValidateNewUser(t *testing.T) {
	tests := []struct {
		name    string
		in      NewUserInput
		wantErr error
	}{
		{
			name: "valid journalist",
			in:   NewUserInput{Password: "longenough", Role: "journalist"},
		},
		{
			name: "valid auditor with credentials and tags",
			in: NewUserInput{
				Password: "longenough", Role: "auditor",
				CredentialURL: "https://orcid.org/0000", Tags: []string{"Economic Analyst"},
			},
		},
		{
			name:    "password too short",
			in:      NewUserInput{Password: "short", Role: "journalist"},
			wantErr: ErrPasswordTooShort,
		},
		{
			name:    "password exactly 8 characters is the floor, not rejected",
			in:      NewUserInput{Password: "12345678", Role: "journalist"},
			wantErr: nil,
		},
		{
			name:    "unknown role rejected",
			in:      NewUserInput{Password: "longenough", Role: "admin"},
			wantErr: ErrInvalidRole,
		},
		{
			name:    "auditor missing credential URL",
			in:      NewUserInput{Password: "longenough", Role: "auditor", Tags: []string{"Economic Analyst"}},
			wantErr: ErrAuditorCredentialsMissing,
		},
		{
			name:    "auditor missing tags",
			in:      NewUserInput{Password: "longenough", Role: "auditor", CredentialURL: "https://orcid.org/0000"},
			wantErr: ErrAuditorCredentialsMissing,
		},
		{
			name:    "journalist doesn't need credentials or tags",
			in:      NewUserInput{Password: "longenough", Role: "journalist"},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateNewUser(tt.in)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("validateNewUser(%+v) = %v, want %v", tt.in, err, tt.wantErr)
			}
		})
	}
}

func TestNormalizeEmail(t *testing.T) {
	tests := []struct{ in, want string }{
		{"Jane@Example.com", "jane@example.com"},
		{"  jane@example.com  ", "jane@example.com"},
		{"JANE@EXAMPLE.COM", "jane@example.com"},
		{"jane@example.com", "jane@example.com"},
	}
	for _, tt := range tests {
		if got := normalizeEmail(tt.in); got != tt.want {
			t.Errorf("normalizeEmail(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
