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
    -- Auditor credential linking for Sybil resistance (NFR-6).
    -- Defaults to FALSE: this is a security gate, so it fails closed. Any row
    -- inserted without naming the column (seeds, fixtures, future tooling)
    -- must not silently acquire voting rights.
    credential_url  TEXT,
    credential_verified BOOLEAN NOT NULL DEFAULT false, -- an admin flips this after reviewing the credential
    tags            TEXT[] NOT NULL DEFAULT '{}', -- auditor category tags, e.g. {"Economic Analyst"}
    -- Auditor reputation ledger. trust_weight is SRS formula (3) materialised,
    -- recomputed on every claim resolution; locked_stake is reputation
    -- currently committed to open votes (FR-6/F-12 require the stake to be
    -- locked before the vote is accepted, not merely recorded).
    trust_weight     DOUBLE PRECISION NOT NULL DEFAULT 0, -- W_a
    successful_votes INTEGER NOT NULL DEFAULT 0,          -- V_s
    failed_votes     INTEGER NOT NULL DEFAULT 0,          -- V_f
    locked_stake     DOUBLE PRECISION NOT NULL DEFAULT 0,
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
    -- F-14 is "Proof of Evidence Appeals Protocol": staking alone isn't the
    -- feature, the new primary evidence is.
    evidence_text   TEXT,
    evidence_url    TEXT,
    status          appeal_status NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at     TIMESTAMPTZ
);

CREATE INDEX idx_articles_journalist ON articles(journalist_id);
CREATE INDEX idx_articles_parent ON articles(parent_article_id);
CREATE INDEX idx_articles_created ON articles(created_at DESC);
CREATE INDEX idx_claims_article ON claims(article_id);
CREATE INDEX idx_claims_status ON claims(status);
CREATE INDEX idx_votes_claim ON votes(claim_id);
CREATE INDEX idx_votes_auditor ON votes(auditor_id);
CREATE INDEX idx_appeals_article ON appeals(article_id);

-- Only one appeal may be active per article, or an author could stack appeals
-- to keep their node pulsing amber (FR-9) indefinitely.
CREATE UNIQUE INDEX uniq_active_appeal_per_article
  ON appeals (article_id) WHERE status = 'active';

-- FR-13/F-20/NFR-7 Immutability Lock. The flag is set automatically once an
-- article crosses the readership threshold, and deletion of a locked row is
-- refused at the storage layer — so the guarantee doesn't depend on every
-- future endpoint remembering to check it.
CREATE OR REPLACE FUNCTION ngj_apply_immutability_lock() RETURNS TRIGGER AS $$
BEGIN
  IF NEW.readership_volume >= 1000 THEN
    NEW.immutability_locked := true;
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_articles_immutability_lock
  BEFORE INSERT OR UPDATE OF readership_volume ON articles
  FOR EACH ROW EXECUTE FUNCTION ngj_apply_immutability_lock();

CREATE OR REPLACE FUNCTION ngj_block_locked_delete() RETURNS TRIGGER AS $$
BEGIN
  IF OLD.immutability_locked THEN
    RAISE EXCEPTION 'article % is immutability-locked (readership %); retract it instead of deleting',
      OLD.id, OLD.readership_volume
      USING ERRCODE = 'restrict_violation';
  END IF;
  RETURN OLD;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_articles_block_locked_delete
  BEFORE DELETE ON articles
  FOR EACH ROW EXECUTE FUNCTION ngj_block_locked_delete();

-- Publication used by Debezium (pgoutput plugin) to stream CDC events, infra/debezium/register-postgres.json
CREATE PUBLICATION ngj_publication FOR TABLE articles, claims, votes, appeals;
