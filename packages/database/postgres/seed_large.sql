-- ============================================================================
-- Large demo dataset for nextGENjournalism.
--
-- Generates a corpus big enough to actually exercise the features the small
-- seed.sql can't: Louvain community detection needs hundreds of nodes across
-- overlapping topics before its output means anything, semantic zooming needs
-- more nodes than fit on screen, and the leaderboard needs enough journalists
-- to rank. Roughly:
--
--   14 journalists, 16 auditors (3 unverified), 1 admin
--   ~900 articles in lineage chains across 12 claim categories
--   ~2,600 claims in every status
--   ~3,500 auditor votes, with slashing already settled
--   active appeals, and retracted tombstones
--
-- Every password is: password123
-- Sign in as admin@example.com to see the compliance and auditor-approval views.
--
-- Safe to re-run: it deletes only the rows it generated (identified by the
-- demo email domain and the generated id prefixes) before regenerating.
-- ============================================================================

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- One transaction for the whole script. Required, not stylistic: the ON COMMIT
-- DROP temp tables below only survive between statements inside a transaction
-- block — run without BEGIN, psql autocommits each statement and every temp
-- table is dropped the instant it's created.
BEGIN;

-- Reproducible: setseed makes every random() call below deterministic, so the
-- same dataset comes out on every run and screenshots stay comparable.
SELECT setseed(0.4207);

-- ---------------------------------------------------------------------------
-- Clean slate for generated rows (children first, for the FKs)
-- ---------------------------------------------------------------------------
-- Scoped by the generated ARTICLES rather than by generated claim/vote id
-- prefixes. Rows created through the running API against a demo article get
-- real random UUIDs, so an id-prefix match misses them and the article delete
-- below then fails on a foreign key. Keying off article_id catches everything
-- attached to demo data regardless of how it was created.
DELETE FROM votes WHERE claim_id IN (
  SELECT c.id FROM claims c
  JOIN articles a ON a.id = c.article_id
  WHERE a.id::text LIKE 'aaa%'
);
DELETE FROM appeals WHERE article_id IN (SELECT id FROM articles WHERE id::text LIKE 'aaa%');
DELETE FROM claims  WHERE article_id IN (SELECT id FROM articles WHERE id::text LIKE 'aaa%');
-- Two things have to be undone before generated articles can be removed:
--   * parent_article_id self-references, so the links must be cut first;
--   * the FR-13/F-20 immutability lock refuses DELETE on any article past the
--     readership threshold, which most generated rows are. Clearing the flag
--     here is deliberate and scoped strictly to demo rows (the 'aaa%' id
--     prefix) — it is the one place that's legitimate, because these articles
--     were never real. Nothing else in the codebase clears it.
UPDATE articles
SET parent_article_id = NULL, immutability_locked = false
WHERE id::text LIKE 'aaa%';
DELETE FROM articles WHERE id::text LIKE 'aaa%';
DELETE FROM users WHERE email LIKE '%@demo.nextgenjournalism.test';

-- ---------------------------------------------------------------------------
-- Categories. These double as claim tags and as auditor expertise tags, which
-- is what the cross-tag consensus rule (FR-7) is checked against.
-- ---------------------------------------------------------------------------
CREATE TEMP TABLE tmp_category(idx int PRIMARY KEY, name text) ON COMMIT DROP;
INSERT INTO tmp_category(idx, name) VALUES
  (0,  'Economic Analyst'),
  (1,  'Geopolitical Analyst'),
  (2,  'Security Analyst'),
  (3,  'Public Health Analyst'),
  (4,  'Climate Analyst'),
  (5,  'Legal Analyst'),
  (6,  'Technology Analyst'),
  (7,  'Elections Analyst'),
  (8,  'Labour Analyst'),
  (9,  'Housing Analyst'),
  (10, 'Education Analyst'),
  (11, 'Transport Analyst');

-- ---------------------------------------------------------------------------
-- Journalists
-- ---------------------------------------------------------------------------
CREATE TEMP TABLE tmp_journalist(idx int PRIMARY KEY, id uuid, name text) ON COMMIT DROP;
INSERT INTO tmp_journalist(idx, id, name)
SELECT i,
       ('bbbbbbbb-0000-4000-8000-' || lpad(i::text, 12, '0'))::uuid,
       (ARRAY[
         'Amara Osei','Devon Cole','Priya Raman','Mateo Silva','Nadia Haddad',
         'Ruth Bergman','Kenji Watanabe','Lucia Ferreira','Omar Diallo','Grete Lind',
         'Ivan Petrov','Sofia Marchetti','Thabo Nkosi','Elena Vargas'
       ])[i + 1]
FROM generate_series(0, 13) AS i;

INSERT INTO users (id, email, password_hash, role, display_name, rank_score, credential_verified, tags)
SELECT j.id,
       'journalist' || (j.idx + 1) || '@demo.nextgenjournalism.test',
       crypt('password123', gen_salt('bf')),
       'journalist',
       j.name,
       0,
       true,   -- not used for journalists; the gate is auditor-only (NFR-6)
       '{}'
FROM tmp_journalist j;

-- ---------------------------------------------------------------------------
-- Auditors. Tag sets deliberately overlap in places: FR-7 resolves a claim
-- only on agreement between auditors whose tag sets are DISJOINT, so a
-- dataset where everyone holds a unique tag would never exercise the
-- "overlapping tags don't count as cross-tag" branch.
--
-- The last three are left credential_verified = false so the admin approval
-- queue (NFR-6) has something in it, and so their votes are correctly refused.
-- ---------------------------------------------------------------------------
CREATE TEMP TABLE tmp_auditor(idx int PRIMARY KEY, id uuid, name text, tags text[], verified boolean) ON COMMIT DROP;
INSERT INTO tmp_auditor(idx, id, name, tags, verified) VALUES
  (0,  'cccccccc-0000-4000-8000-000000000000'::uuid, 'Priya Nair',      '{"Economic Analyst"}',                          true),
  (1,  'cccccccc-0000-4000-8000-000000000001'::uuid, 'Sam Okafor',      '{"Geopolitical Analyst"}',                      true),
  (2,  'cccccccc-0000-4000-8000-000000000002'::uuid, 'Lin Zhao',        '{"Security Analyst"}',                          true),
  (3,  'cccccccc-0000-4000-8000-000000000003'::uuid, 'Hana Kim',        '{"Public Health Analyst"}',                     true),
  (4,  'cccccccc-0000-4000-8000-000000000004'::uuid, 'Tomas Novak',     '{"Climate Analyst"}',                           true),
  (5,  'cccccccc-0000-4000-8000-000000000005'::uuid, 'Aisha Bello',     '{"Legal Analyst"}',                             true),
  (6,  'cccccccc-0000-4000-8000-000000000006'::uuid, 'Erik Lund',       '{"Technology Analyst"}',                        true),
  (7,  'cccccccc-0000-4000-8000-000000000007'::uuid, 'Maria Santos',    '{"Elections Analyst"}',                         true),
  (8,  'cccccccc-0000-4000-8000-000000000008'::uuid, 'Jonah Weiss',     '{"Labour Analyst"}',                            true),
  (9,  'cccccccc-0000-4000-8000-000000000009'::uuid, 'Fatima Zahra',    '{"Housing Analyst"}',                           true),
  (10, 'cccccccc-0000-4000-8000-00000000000a'::uuid, 'Peter Adeyemi',   '{"Education Analyst"}',                         true),
  (11, 'cccccccc-0000-4000-8000-00000000000b'::uuid, 'Sara Lindqvist',  '{"Transport Analyst"}',                         true),
  -- overlapping / multi-tag auditors
  (12, 'cccccccc-0000-4000-8000-00000000000c'::uuid, 'Daniel Cruz',     '{"Economic Analyst","Labour Analyst"}',         true),
  (13, 'cccccccc-0000-4000-8000-00000000000d'::uuid, 'Yuki Tanaka',     '{"Technology Analyst","Security Analyst"}',     false),
  (14, 'cccccccc-0000-4000-8000-00000000000e'::uuid, 'Nour Khalil',     '{"Climate Analyst","Transport Analyst"}',       false),
  (15, 'cccccccc-0000-4000-8000-00000000000f'::uuid, 'Greg Mensah',     '{"Legal Analyst","Elections Analyst"}',         false);

INSERT INTO users (id, email, password_hash, role, display_name, rank_score, credential_url, credential_verified, tags, successful_votes, failed_votes, trust_weight)
SELECT a.id,
       'auditor' || (a.idx + 1) || '@demo.nextgenjournalism.test',
       crypt('password123', gen_salt('bf')),
       'auditor',
       a.name,
       -- Starting reputation. Auditors need a real balance to stake against,
       -- since a vote now locks reputation rather than merely recording a
       -- number, and the historical votes settled below will move it.
       120 + (a.idx % 5) * 20,
       'https://orcid.org/0000-0002-' || lpad(a.idx::text, 4, '0') || '-000' || (a.idx % 10),
       a.verified,
       a.tags,
       0, 0, 0
FROM tmp_auditor a;

-- Admin
INSERT INTO users (id, email, password_hash, role, display_name, credential_verified, tags)
VALUES ('dddddddd-0000-4000-8000-000000000000'::uuid,
        'admin@demo.nextgenjournalism.test',
        crypt('password123', gen_salt('bf')),
        'admin', 'Platform Admin', true, '{}');

-- ---------------------------------------------------------------------------
-- Articles.
--
-- Each article belongs to a (journalist, category) "beat". Within a beat,
-- articles are chained oldest -> newest via parent_article_id, which is what
-- produces the SEQUENCE_OF lineage (FR-4/F-02). Roughly every 5th article
-- starts a fresh chain so the graph has multiple story arcs per beat rather
-- than one 60-node conga line.
--
-- Readership is drawn from a heavy-tailed distribution (most stories are read
-- a little, a few go viral) because that is what makes FR-12's log-scaled node
-- sizing and SRS formula (1)'s log10 dampener visibly do something.
-- ---------------------------------------------------------------------------
CREATE TEMP TABLE tmp_article AS
SELECT
  n                                                                       AS n,
  ('aaaaaaaa-0000-4000-8000-' || lpad(n::text, 12, '0'))::uuid            AS id,
  -- The multipliers must be coprime with their moduli, or the sequence only
  -- reaches a fraction of the range: (n * 7) % 14 yields nothing but 0 and 7,
  -- since gcd(7,14) = 7. gcd(5,14) = 1 and gcd(7,12) = 1 both cycle through
  -- every value, so articles spread across all 14 journalists and all 12
  -- categories. The n/14 term decorrelates the two so a journalist isn't
  -- locked to a fixed rotation of beats.
  (n * 5) % 14                                                            AS journalist_idx,
  (n * 7 + (n / 14)) % 12                                                 AS category_idx,
  now() - ((1100 - n * 1.2) || ' days')::interval                         AS created_at,
  -- heavy tail: exp of a uniform draw, floored at 12 reads
  GREATEST(12, floor(exp(random() * 9.6))::bigint)                        AS readership_volume,
  random()                                                                AS r1,
  random()                                                                AS r2
FROM generate_series(1, 900) AS n;

CREATE INDEX ON tmp_article(journalist_idx, category_idx, created_at);

-- Chain articles within each beat. NULLIF(..., ...) breaks the chain every
-- 5th article so each beat contains several separate story arcs.
CREATE TEMP TABLE tmp_article_linked AS
SELECT a.*,
       CASE WHEN (a.chain_pos % 5) = 0 THEN NULL ELSE a.prev_id END AS parent_article_id
FROM (
  SELECT t.*,
         LAG(t.id) OVER w                            AS prev_id,
         ROW_NUMBER() OVER w                         AS chain_pos
  FROM tmp_article t
  WINDOW w AS (PARTITION BY t.journalist_idx, t.category_idx ORDER BY t.created_at)
) a;

INSERT INTO articles (id, journalist_id, parent_article_id, title, body, signature,
                      readership_volume, verified_claims, self_corrected_claims, false_claims,
                      is_retracted, created_at)
SELECT
  a.id,
  j.id,
  a.parent_article_id,
  c.name || ': ' || (ARRAY[
    'Records Show a Widening Gap','Officials Dispute the Figures','New Documents Surface',
    'What the Data Actually Says','Follow-up: Three Months On','The Numbers Behind the Claim',
    'A Second Source Comes Forward','Internal Memo Contradicts Statement','Audit Review Finds Discrepancy',
    'Committee Opens an Inquiry','Correction and Context','Analysis: Who Benefits'
  ])[1 + (a.n % 12)] || ' (#' || a.n || ')',
  'Reporting on ' || lower(c.name) || ' matters. This story is part of an ongoing series; '
    || 'figures cited here are tagged as individual claims and submitted for independent audit. '
    || 'Story reference ' || a.n || '.',
  'sig-' || encode(digest('article-' || a.n, 'sha256'), 'hex'),
  a.readership_volume,
  0, 0, 0,
  false,
  a.created_at
FROM tmp_article_linked a
JOIN tmp_journalist j ON j.idx = a.journalist_idx
JOIN tmp_category   c ON c.idx = a.category_idx;

-- ---------------------------------------------------------------------------
-- Claims: 1-4 per article. Most carry the article's own category, but ~25%
-- are cross-category — that overlap is what creates topic edges between beats
-- and gives Louvain a non-trivial community structure to find.
--
-- Status mix is weighted toward resolved so the graph has real Corruption
-- Factor variation, with a healthy pending backlog for auditors to work.
-- ---------------------------------------------------------------------------
INSERT INTO claims (id, article_id, text, tag, status, created_at)
SELECT
  ('cccccccc-1111-4000-8000-' || lpad((a.n * 10 + k)::text, 12, '0'))::uuid,
  a.id,
  (ARRAY[
    'The reported total is accurate as of the filing date.',
    'The cited percentage reflects year-over-year change, not absolute change.',
    'A named official confirmed the figure on the record.',
    'The underlying dataset was published by the agency itself.',
    'This contradicts the department''s earlier public statement.',
    'The sample covers the full reporting period, not a partial quarter.'
  ])[1 + ((a.n + k) % 6)],
  -- 75% own category, 25% a neighbouring one
  CASE WHEN ((a.n + k) % 4) = 0
       THEN cx.name
       ELSE c.name
  END,
  (CASE
     WHEN a.r1 < 0.42 THEN 'verified'
     WHEN a.r1 < 0.58 THEN 'false'
     WHEN a.r1 < 0.70 THEN 'self_corrected'
     ELSE 'pending'
   END)::claim_status,
  a.created_at + ((k || ' hours')::interval)
FROM tmp_article_linked a
JOIN tmp_category c  ON c.idx = a.category_idx
JOIN tmp_category cx ON cx.idx = (a.category_idx + 1 + (a.n % 5)) % 12
CROSS JOIN LATERAL generate_series(1, 1 + (a.n % 4)) AS k;

-- ---------------------------------------------------------------------------
-- Roll the resolved claim counts up onto their articles (C_vd / C_sc / C_f),
-- which is what drives both the Corruption Factor (formula 2) and the
-- Journalist Rank Score (formula 1).
-- ---------------------------------------------------------------------------
UPDATE articles a
SET verified_claims       = t.verified,
    self_corrected_claims = t.self_corrected,
    false_claims          = t.false_count
FROM (
  SELECT article_id,
         COUNT(*) FILTER (WHERE status = 'verified')       AS verified,
         COUNT(*) FILTER (WHERE status = 'self_corrected') AS self_corrected,
         COUNT(*) FILTER (WHERE status = 'false')          AS false_count
  FROM claims
  WHERE id::text LIKE 'cccccccc-1111%'
  GROUP BY article_id
) t
WHERE a.id = t.article_id;

-- ---------------------------------------------------------------------------
-- Votes on every resolved claim, from auditors whose tags cover that claim's
-- category. Most auditors voted with the eventual consensus; a deterministic
-- minority voted against it and are marked misaligned, so the slashing
-- ledger (FR-8/F-13) has real history rather than a table of all-true rows.
-- ---------------------------------------------------------------------------
INSERT INTO votes (claim_id, auditor_id, stake, verdict, aligned_with_consensus, created_at)
SELECT
  cl.id,
  au.id,
  ROUND((1 + (abs(hashtext(cl.id::text || au.id::text)) % 5))::numeric, 2)::double precision,
  aligned.is_aligned = (cl.status = 'verified'),
  aligned.is_aligned,
  cl.created_at + interval '6 hours'
FROM claims cl
JOIN tmp_category cat ON cat.name = cl.tag
JOIN tmp_auditor  au  ON au.tags && ARRAY[cl.tag] AND au.verified
CROSS JOIN LATERAL (
  -- ~20% of votes go against the final consensus
  SELECT (abs(hashtext(cl.id::text || au.id::text)) % 10) >= 2 AS is_aligned
) aligned
WHERE cl.id::text LIKE 'cccccccc-1111%'
  AND cl.status IN ('verified', 'false')
ON CONFLICT (claim_id, auditor_id) DO NOTHING;

-- Settle the auditor reputation ledger from those votes, matching
-- consensus.ApplySlashingTx exactly: V_s / V_f counters, full stake forfeited
-- on misaligned votes, and AlignedRewardRate (0.5) of the stake paid out on
-- aligned ones. The reward half matters — settling ~1,500 historical votes
-- with slashing alone drives every auditor to exactly 0 reputation, and since
-- voting requires available reputation to stake, none of them can then cast a
-- single vote in the demo.
UPDATE users u
SET successful_votes = s.aligned,
    failed_votes     = s.misaligned,
    rank_score       = GREATEST(u.rank_score - s.forfeited + s.rewarded, 0)
FROM (
  SELECT auditor_id,
         COUNT(*) FILTER (WHERE aligned_with_consensus)                     AS aligned,
         COUNT(*) FILTER (WHERE NOT aligned_with_consensus)                 AS misaligned,
         COALESCE(SUM(stake) FILTER (WHERE NOT aligned_with_consensus), 0)  AS forfeited,
         COALESCE(SUM(stake * 0.5) FILTER (WHERE aligned_with_consensus), 0) AS rewarded
  FROM votes
  GROUP BY auditor_id
) s
WHERE u.id = s.auditor_id;

UPDATE users
SET trust_weight = CASE
      WHEN (successful_votes + failed_votes) = 0 THEN 0
      ELSE log(10, 1 + successful_votes)
           * (1 - failed_votes::double precision / (successful_votes + failed_votes))
    END
WHERE role = 'auditor';

-- ---------------------------------------------------------------------------
-- Retractions (FR-14): a handful of high-reach stories tombstoned, keeping the
-- node in the graph but greyed out.
-- ---------------------------------------------------------------------------
WITH picked AS (
  SELECT id, title, body, journalist_id, readership_volume
  FROM articles
  WHERE id::text LIKE 'aaa%' AND false_claims >= 2
  ORDER BY readership_volume DESC
  LIMIT 8
)
-- The separator must be a real NUL byte to match compliance.tombstoneHash in
-- the Go backend (sha256 of title + "\x00" + body). A NUL can't appear inside
-- a Postgres `text` value at all, so the concatenation is done in bytea.
UPDATE articles a
SET title        = '[retracted]',
    body         = 'tombstone:' || encode(
                     digest(
                       convert_to(p.title, 'UTF8') || '\x00'::bytea || convert_to(p.body, 'UTF8'),
                       'sha256'
                     ), 'hex'),
    is_retracted = true,
    retracted_at = now() - interval '10 days'
FROM picked p
WHERE a.id = p.id;

-- FR-15: permanent, reach-scaled rank deduction for each retraction.
UPDATE users u
SET rank_score = GREATEST(u.rank_score - pen.total, 0)
FROM (
  SELECT journalist_id, SUM(2 + log(10, 1 + readership_volume)) AS total
  FROM articles WHERE is_retracted AND id::text LIKE 'aaa%'
  GROUP BY journalist_id
) pen
WHERE u.id = pen.journalist_id;

-- ---------------------------------------------------------------------------
-- Active appeals (FR-5/F-14/FR-9): a few contested stories left pulsing amber.
-- ---------------------------------------------------------------------------
INSERT INTO appeals (article_id, journalist_id, staked_percent, evidence_text, evidence_url, status, created_at)
SELECT a.id, a.journalist_id, 10 + (abs(hashtext(a.id::text)) % 20),
       'Primary source documents obtained after publication contradict the auditors'' finding on this claim.',
       'https://example.org/evidence/' || left(a.id::text, 8),
       'active',
       now() - interval '3 days'
FROM articles a
WHERE a.id::text LIKE 'aaa%' AND NOT a.is_retracted AND a.false_claims >= 2
ORDER BY a.readership_volume DESC
LIMIT 12
ON CONFLICT DO NOTHING;

-- A few resolved appeals, for history.
INSERT INTO appeals (article_id, journalist_id, staked_percent, evidence_text, status, created_at, resolved_at)
SELECT a.id, a.journalist_id, 15,
       'Correction issued by the source agency after publication.',
       (CASE WHEN (abs(hashtext(a.id::text)) % 2) = 0 THEN 'upheld' ELSE 'rejected' END)::appeal_status,
       now() - interval '60 days', now() - interval '45 days'
FROM articles a
WHERE a.id::text LIKE 'aaa%' AND NOT a.is_retracted AND a.false_claims = 1
ORDER BY a.created_at
LIMIT 15
ON CONFLICT DO NOTHING;

-- ---------------------------------------------------------------------------
-- Journalist Rank Score, SRS formula (1):
--   R = log10(1+V) + w1*C_vd + w2*C_sc - w3*C_f      (w1=1.0, w2=1.5, w3=4.0)
-- summed across each journalist's articles, matching
-- ranking.JournalistRankScore in the Go backend.
-- ---------------------------------------------------------------------------
UPDATE users u
SET rank_score = GREATEST(u.rank_score + r.score, 0)
FROM (
  SELECT journalist_id,
         SUM(log(10, 1 + readership_volume)
             + 1.0 * verified_claims
             + 1.5 * self_corrected_claims
             - 4.0 * false_claims) AS score
  FROM articles
  WHERE id::text LIKE 'aaa%'
  GROUP BY journalist_id
) r
WHERE u.id = r.journalist_id;

-- ---------------------------------------------------------------------------
-- Summary
-- ---------------------------------------------------------------------------
SELECT 'journalists' AS entity, count(*) FROM users WHERE role = 'journalist'
UNION ALL SELECT 'auditors',        count(*) FROM users WHERE role = 'auditor'
UNION ALL SELECT 'auditors_pending',count(*) FROM users WHERE role = 'auditor' AND NOT credential_verified
UNION ALL SELECT 'articles',        count(*) FROM articles
UNION ALL SELECT 'lineage_edges',   count(*) FROM articles WHERE parent_article_id IS NOT NULL
UNION ALL SELECT 'retracted',       count(*) FROM articles WHERE is_retracted
UNION ALL SELECT 'locked',          count(*) FROM articles WHERE immutability_locked
UNION ALL SELECT 'claims',          count(*) FROM claims
UNION ALL SELECT 'claims_pending',  count(*) FROM claims WHERE status = 'pending'
UNION ALL SELECT 'votes',           count(*) FROM votes
UNION ALL SELECT 'appeals_active',  count(*) FROM appeals WHERE status = 'active';

COMMIT;
