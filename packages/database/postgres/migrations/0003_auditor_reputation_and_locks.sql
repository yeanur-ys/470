-- 0003: real auditor reputation economics + the immutability lock.
--
-- Before this migration the consensus system was reputation-flavoured but not
-- reputation-backed: votes.stake was written and never read, ApplySlashing set
-- a boolean and deducted nothing, and ranking.AuditorTrustWeight (SRS formula
-- 3) was never called by any production code path. F-10/F-12/F-13's acceptance
-- criteria all failed as a result. These columns give those formulas somewhere
-- to live.

-- Auditor reputation ledger. trust_weight is SRS formula (3) materialised:
--   Wa = log10(1 + Vs) * (1 - Vf/Vtotal)
-- recomputed from successful_votes/failed_votes every time a claim resolves.
ALTER TABLE users ADD COLUMN IF NOT EXISTS trust_weight      DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS successful_votes  INTEGER NOT NULL DEFAULT 0; -- V_s
ALTER TABLE users ADD COLUMN IF NOT EXISTS failed_votes      INTEGER NOT NULL DEFAULT 0; -- V_f
-- Reputation currently locked in open (unresolved) votes. FR-6/F-12 require
-- the stake to be *deducted or locked* before the vote is accepted, not merely
-- recorded, so an auditor cannot stake the same reputation on ten claims.
ALTER TABLE users ADD COLUMN IF NOT EXISTS locked_stake      DOUBLE PRECISION NOT NULL DEFAULT 0;

-- NFR-6 is a security gate, so it must fail closed. The original schema
-- defaulted credential_verified to TRUE, meaning any row inserted without
-- naming the column explicitly (seeds, fixtures, a future admin tool) silently
-- got voting rights. Journalists/admins don't use this flag at all, so
-- defaulting it to FALSE costs them nothing and closes the hole.
ALTER TABLE users ALTER COLUMN credential_verified SET DEFAULT false;

-- FR-13/F-20/NFR-7: once an article passes the readership threshold it can no
-- longer be hard-deleted. The column already existed but nothing ever set it.
-- The trigger below makes the lock automatic rather than something a caller
-- has to remember, and the delete guard makes it actually enforced at the
-- storage layer instead of relying on every future endpoint being careful.
ALTER TABLE articles ADD COLUMN IF NOT EXISTS immutability_locked BOOLEAN NOT NULL DEFAULT false;

CREATE OR REPLACE FUNCTION ngj_apply_immutability_lock() RETURNS TRIGGER AS $$
BEGIN
  IF NEW.readership_volume >= 1000 THEN
    NEW.immutability_locked := true;
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_articles_immutability_lock ON articles;
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

DROP TRIGGER IF EXISTS trg_articles_block_locked_delete ON articles;
CREATE TRIGGER trg_articles_block_locked_delete
  BEFORE DELETE ON articles
  FOR EACH ROW EXECUTE FUNCTION ngj_block_locked_delete();

-- Backfill the lock for rows that already crossed the threshold.
UPDATE articles SET immutability_locked = true WHERE readership_volume >= 1000;

-- FR-14 evidence submission (F-14 was "Partial": you could stake on an appeal
-- but had nowhere to put the "proof of evidence" the feature is named after).
ALTER TABLE appeals ADD COLUMN IF NOT EXISTS evidence_text TEXT;
ALTER TABLE appeals ADD COLUMN IF NOT EXISTS evidence_url  TEXT;

-- One active appeal per article — without this an author could stack appeals
-- to keep a node pulsing amber indefinitely.
CREATE UNIQUE INDEX IF NOT EXISTS uniq_active_appeal_per_article
  ON appeals (article_id) WHERE status = 'active';

CREATE INDEX IF NOT EXISTS idx_votes_auditor ON votes(auditor_id);
CREATE INDEX IF NOT EXISTS idx_claims_status ON claims(status);
CREATE INDEX IF NOT EXISTS idx_articles_created ON articles(created_at DESC);
