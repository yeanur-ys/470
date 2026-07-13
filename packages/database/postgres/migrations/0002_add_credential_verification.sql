-- +migrate Up
-- Adds the NFR-6 credential-verification gate. Existing rows default to
-- true (no behavior change for already-seeded accounts); the signup
-- endpoint explicitly sets this to false for newly self-registered auditors.

ALTER TABLE users
  ADD COLUMN IF NOT EXISTS credential_verified BOOLEAN NOT NULL DEFAULT true;
