-- nextGENjournalism relational schema (Section 3.1 of the SRS)

CREATE TYPE user_role AS ENUM ('journalist', 'auditor', 'admin');
CREATE TYPE claim_status AS ENUM ('pending', 'verified', 'self_corrected', 'false');
CREATE TYPE appeal_status AS ENUM ('active', 'upheld', 'rejected');

CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           TEXT UNIQUE NOT NULL,
    password_hash   TEXT NOT NULL,
    role            user_role NOT NULL,
    display_name    TEXT NOT NULL,
    -- Journalist Rank Score (R), formula in SRS Section 4
    rank_score      DOUBLE PRECISION NOT NULL DEFAULT 0,
    -- Auditor credential linking for Sybil resistance (NFR-6)
    credential_url  TEXT,
    credential_verified BOOLEAN NOT NULL DEFAULT true, -- false for newly self-registered auditors until an admin approves
    tags            TEXT[] NOT NULL DEFAULT '{}', -- auditor category tags, e.g. {"Economic Analyst"}
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE articles (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    journalist_id       UUID NOT NULL REFERENCES users(id),
    parent_article_id   UUID REFERENCES articles(id), -- FR-4 Sequence Stitching
    title               TEXT NOT NULL,
    body                TEXT NOT NULL,
    signature           TEXT NOT NULL, -- cryptographic author signature, NFR-4
    readership_volume   BIGINT NOT NULL DEFAULT 0,
    verified_claims      INTEGER NOT NULL DEFAULT 0, -- C_vd
    self_corrected_claims INTEGER NOT NULL DEFAULT 0, -- C_sc
    false_claims         INTEGER NOT NULL DEFAULT 0, -- C_f
    is_retracted        BOOLEAN NOT NULL DEFAULT false, -- FR-14
    retracted_at        TIMESTAMPTZ,
    immutability_locked  BOOLEAN NOT NULL DEFAULT false, -- FR-13
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE claims (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    article_id  UUID NOT NULL REFERENCES articles(id),
    text        TEXT NOT NULL,
    tag         TEXT NOT NULL, -- category used for cross-tag validation, FR-7
    status      claim_status NOT NULL DEFAULT 'pending',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE votes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    claim_id    UUID NOT NULL REFERENCES claims(id),
    auditor_id  UUID NOT NULL REFERENCES users(id),
    stake       DOUBLE PRECISION NOT NULL, -- reputation staked, FR-6
    verdict     BOOLEAN NOT NULL, -- true = confirms claim
    aligned_with_consensus BOOLEAN, -- set once claim resolves, drives slashing (FR-8)
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (claim_id, auditor_id)
);

CREATE TABLE appeals (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    article_id      UUID NOT NULL REFERENCES articles(id),
    journalist_id   UUID NOT NULL REFERENCES users(id),
    staked_percent  DOUBLE PRECISION NOT NULL, -- FR-5
    status          appeal_status NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at     TIMESTAMPTZ
);

CREATE INDEX idx_articles_journalist ON articles(journalist_id);
CREATE INDEX idx_articles_parent ON articles(parent_article_id);
CREATE INDEX idx_claims_article ON claims(article_id);
CREATE INDEX idx_votes_claim ON votes(claim_id);
CREATE INDEX idx_appeals_article ON appeals(article_id);

-- Publication used by Debezium (pgoutput plugin) to stream CDC events, infra/debezium/register-postgres.json
CREATE PUBLICATION ngj_publication FOR TABLE articles, claims, votes, appeals;
