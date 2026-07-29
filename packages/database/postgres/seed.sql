-- Seed data for local exploration / demos.
-- Login as any of the accounts below with password: password123
--
--   journalist1@example.com  (Amara Osei)   — the budget/transit story cluster
--   journalist2@example.com  (Devon Cole)   — the layoffs story cluster + a retraction
--   auditor1@example.com     (Priya Nair)   — tag: Economic Analyst, verified
--   auditor2@example.com     (Sam Okafor)   — tag: Geopolitical Analyst, verified
--   auditor3@example.com     (Lin Zhao)     — tag: Security Analyst, NOT yet verified
--                                              (log in as admin to approve them)
--   admin@example.com        (Admin)
--
-- Safe to re-run: everything is upserted by fixed id.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ---------- Users ----------
INSERT INTO users (id, email, password_hash, role, display_name, rank_score, credential_url, credential_verified, tags)
VALUES
  ('11111111-1111-1111-1111-111111111111', 'journalist1@example.com', crypt('password123', gen_salt('bf')), 'journalist', 'Amara Osei', 0, NULL, true, '{}'),
  ('22222222-2222-2222-2222-222222222222', 'journalist2@example.com', crypt('password123', gen_salt('bf')), 'journalist', 'Devon Cole', 0, NULL, true, '{}'),
  ('33333333-3333-3333-3333-333333333333', 'auditor1@example.com', crypt('password123', gen_salt('bf')), 'auditor', 'Priya Nair', 0, 'https://orcid.org/0000-0001-example-1', true, '{"Economic Analyst"}'),
  ('44444444-4444-4444-4444-444444444444', 'auditor2@example.com', crypt('password123', gen_salt('bf')), 'auditor', 'Sam Okafor', 0, 'https://orcid.org/0000-0001-example-2', true, '{"Geopolitical Analyst"}'),
  ('55555555-5555-5555-5555-555555555555', 'auditor3@example.com', crypt('password123', gen_salt('bf')), 'auditor', 'Lin Zhao', 0, 'https://orcid.org/0000-0001-example-3', false, '{"Security Analyst"}'),
  ('66666666-6666-6666-6666-666666666666', 'admin@example.com', crypt('password123', gen_salt('bf')), 'admin', 'Admin', 0, NULL, true, '{}')
ON CONFLICT (id) DO NOTHING;

-- ---------- Articles: Amara's budget story chain (cluster "budget") ----------
INSERT INTO articles (id, journalist_id, parent_article_id, title, body, signature, readership_volume, verified_claims, self_corrected_claims, false_claims, is_retracted, created_at)
VALUES
  ('a0000000-0000-0000-0000-000000000001', '11111111-1111-1111-1111-111111111111', NULL,
   'City Budget Shows $40M Shortfall', 'An analysis of the city''s latest financial disclosure reveals a $40 million gap between projected and actual revenue for the fiscal year.', 'sig-a1',
   5000, 2, 0, 0, false, now() - interval '140 days'),

  ('a0000000-0000-0000-0000-000000000002', '11111111-1111-1111-1111-111111111111', 'a0000000-0000-0000-0000-000000000001',
   'Budget Shortfall Traced to Pension Fund', 'Follow-up reporting traces the majority of the shortfall to underfunded municipal pension obligations dating back a decade.', 'sig-a2',
   3000, 1, 0, 0, false, now() - interval '95 days'),

  ('a0000000-0000-0000-0000-000000000003', '11111111-1111-1111-1111-111111111111', 'a0000000-0000-0000-0000-000000000002',
   'Officials Deny Pension Mismanagement', 'City officials pushed back on the pension fund reporting, calling one cited statistic inaccurate.', 'sig-a3',
   1200, 0, 0, 1, false, now() - interval '9 days'),

  ('a0000000-0000-0000-0000-000000000004', '11111111-1111-1111-1111-111111111111', 'a0000000-0000-0000-0000-000000000003',
   'Follow-up: New Documents Surface', 'Newly obtained records shed further light on the pension fund''s management over the past decade.', 'sig-a4',
   800, 1, 0, 0, false, now() - interval '2 days')
ON CONFLICT (id) DO NOTHING;

-- ---------- Articles: Amara's transit story chain (cluster "transit") ----------
INSERT INTO articles (id, journalist_id, parent_article_id, title, body, signature, readership_volume, verified_claims, self_corrected_claims, false_claims, is_retracted, created_at)
VALUES
  ('a0000000-0000-0000-0000-000000000005', '11111111-1111-1111-1111-111111111111', NULL,
   'New Transit Line Opens Downtown', 'The city''s newest light-rail line opened to the public today after three years of construction.', 'sig-a5',
   10000, 3, 1, 0, false, now() - interval '5 days'),

  ('a0000000-0000-0000-0000-000000000006', '11111111-1111-1111-1111-111111111111', 'a0000000-0000-0000-0000-000000000005',
   'Transit Ridership Exceeds Projections', 'Early ridership figures for the new line are running well ahead of the transit authority''s initial projections.', 'sig-a6',
   2000, 1, 0, 0, false, now() - interval '1 days')
ON CONFLICT (id) DO NOTHING;

-- ---------- Articles: Devon's layoffs story chain (cluster "layoffs") ----------
INSERT INTO articles (id, journalist_id, parent_article_id, title, body, signature, readership_volume, verified_claims, self_corrected_claims, false_claims, is_retracted, created_at)
VALUES
  ('b0000000-0000-0000-0000-000000000001', '22222222-2222-2222-2222-222222222222', NULL,
   'Tech Layoffs Hit Regional Startups', 'A wave of layoffs has swept through the region''s startup sector over the past month.', 'sig-b1',
   1500, 1, 0, 1, false, now() - interval '220 days'),

  ('b0000000-0000-0000-0000-000000000002', '22222222-2222-2222-2222-222222222222', 'b0000000-0000-0000-0000-000000000001',
   'Startup Founders Push Back on Layoff Report', 'Several founders named in the original report dispute the total headcount figures cited.', 'sig-b2',
   900, 0, 0, 2, false, now() - interval '110 days'),

  ('b0000000-0000-0000-0000-000000000003', '22222222-2222-2222-2222-222222222222', 'b0000000-0000-0000-0000-000000000002',
   'Correction: Layoff Numbers Revised', 'This follow-up revises the previously reported headcount figures after founders provided corrected data.', 'sig-b3',
   600, 0, 1, 0, false, now() - interval '20 days'),

  ('b0000000-0000-0000-0000-000000000004', '22222222-2222-2222-2222-222222222222', NULL,
   '[retracted]', 'tombstone:seed-data-demo-retraction-hash', 'sig-b4',
   400, 0, 0, 1, true, now() - interval '260 days')
ON CONFLICT (id) DO NOTHING;

UPDATE articles SET retracted_at = now() - interval '250 days' WHERE id = 'b0000000-0000-0000-0000-000000000004' AND retracted_at IS NULL;
UPDATE users SET rank_score = rank_score - 2 WHERE id = '22222222-2222-2222-2222-222222222222'; -- FR-15 penalty for the retraction above

-- ---------- Claims (mix of resolved and pending, across categories) ----------
INSERT INTO claims (id, article_id, text, tag, status, created_at) VALUES
  ('c0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', 'The shortfall totals $40 million against projected revenue.', 'Economic Analyst', 'verified', now() - interval '139 days'),
  ('c0000000-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000001', 'The gap represents an 8% variance from the adopted budget.', 'Economic Analyst', 'verified', now() - interval '139 days'),
  ('c0000000-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000002', 'Pension underfunding accounts for the majority of the gap.', 'Economic Analyst', 'verified', now() - interval '94 days'),
  ('c0000000-0000-0000-0000-000000000004', 'a0000000-0000-0000-0000-000000000003', 'A city spokesperson confirmed the pension fund is fully solvent.', 'Economic Analyst', 'false', now() - interval '8 days'),
  ('c0000000-0000-0000-0000-000000000005', 'a0000000-0000-0000-0000-000000000004', 'Newly obtained records show three years of missed actuarial reviews.', 'Economic Analyst', 'verified', now() - interval '1 days'),
  ('c0000000-0000-0000-0000-000000000006', 'a0000000-0000-0000-0000-000000000005', 'The line cost $410 million to construct.', 'Economic Analyst', 'verified', now() - interval '4 days'),
  ('c0000000-0000-0000-0000-000000000007', 'a0000000-0000-0000-0000-000000000005', 'Construction began in 2022 and ran three years.', 'Economic Analyst', 'verified', now() - interval '4 days'),
  ('c0000000-0000-0000-0000-000000000008', 'a0000000-0000-0000-0000-000000000005', 'The agency initially projected 12,000 daily riders.', 'Economic Analyst', 'verified', now() - interval '4 days'),
  ('c0000000-0000-0000-0000-000000000009', 'a0000000-0000-0000-0000-000000000005', 'An early planning estimate of 8,000 riders was revised upward before opening.', 'Economic Analyst', 'self_corrected', now() - interval '4 days'),
  ('c0000000-0000-0000-0000-000000000010', 'b0000000-0000-0000-0000-000000000001', 'Approximately 400 workers were laid off across the sector.', 'Geopolitical Analyst', 'verified', now() - interval '219 days'),
  ('c0000000-0000-0000-0000-000000000011', 'b0000000-0000-0000-0000-000000000001', 'One named company laid off its entire 90-person staff.', 'Geopolitical Analyst', 'false', now() - interval '219 days'),
  ('c0000000-0000-0000-0000-000000000012', 'b0000000-0000-0000-0000-000000000002', 'The named company disputes the 90-person figure as inflated.', 'Geopolitical Analyst', 'false', now() - interval '109 days'),
  ('c0000000-0000-0000-0000-000000000013', 'b0000000-0000-0000-0000-000000000002', 'Founders provided payroll records showing 35 affected employees.', 'Geopolitical Analyst', 'false', now() - interval '109 days'),
  ('c0000000-0000-0000-0000-000000000014', 'b0000000-0000-0000-0000-000000000003', 'The corrected total across the sector is 340 layoffs.', 'Geopolitical Analyst', 'self_corrected', now() - interval '19 days'),
  -- pending: nothing has voted on these yet — sign in as an auditor to work these
  ('c0000000-0000-0000-0000-000000000015', 'a0000000-0000-0000-0000-000000000006', 'Ridership on opening week averaged 14,500 daily riders.', 'Security Analyst', 'pending', now()),
  ('c0000000-0000-0000-0000-000000000016', 'a0000000-0000-0000-0000-000000000004', 'The missed reviews were flagged internally as early as 2021.', 'Security Analyst', 'pending', now())
ON CONFLICT (id) DO NOTHING;

-- ---------- One example vote, so the votes table isn't empty either ----------
INSERT INTO votes (claim_id, auditor_id, stake, verdict, aligned_with_consensus)
VALUES ('c0000000-0000-0000-0000-000000000001', '33333333-3333-3333-3333-333333333333', 1.0, true, true)
ON CONFLICT (claim_id, auditor_id) DO NOTHING;

-- ---------- An active appeal, to demo the pulsing "under dispute" node ----------
INSERT INTO appeals (article_id, journalist_id, staked_percent, status)
SELECT 'a0000000-0000-0000-0000-000000000003', '11111111-1111-1111-1111-111111111111', 15, 'active'
WHERE NOT EXISTS (
  SELECT 1 FROM appeals WHERE article_id = 'a0000000-0000-0000-0000-000000000003' AND status = 'active'
);
