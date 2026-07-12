# Migration Notes — base repo → rebuilt structure

This documents what was actually wrong with the `copilot/nextgenjournalism`
branch and exactly what changed. Keep this file until the team has reviewed
every point, then delete it.

## What was broken

1. **Leftover scaffolding never cleaned up.** `apps/docs`, `apps/web`, and
   `apps/api` are the default output of `create-turbo`/`go mod init` and were
   never removed. `apps/api` is an empty Go module that duplicates
   `apps/go-backend`. None of the three appear in your own target tree
   (`Gemini_Code_Repository_Structure.txt`) or the SRS.
2. **Duplicate, conflicting package sets.** `packages/ui`, `packages/eslint-config`,
   and `packages/typescript-config` (create-turbo defaults, namespaced `@repo/*`)
   coexisted with `packages/config-eslint` and `packages/config-typescript`
   (your intended, differently-named packages). `apps/frontend` depended on
   `@repo/eslint-config` / `@repo/typescript-config`, which is a **different
   package** than the one that actually held your project's rules — so lint/type
   config wasn't the one you thought it was.
3. **Two `docker-compose.yml` files** (root and `infra/`), with no clear source
   of truth.
4. **The auth middleware was a single global role guard hard-coded to
   `"journalist"`** (`RoleGuard("journalist", mux)` in the old `routes.go`).
   Auditors and admins could never pass it — every non-journalist route was
   permanently forbidden. It also trusted a client-supplied `X-Role` header,
   which any caller can set to anything.
5. **No real persistence anywhere.** Every handler (`auth`, `articles`,
   `consensus`, `compliance`) was a stub that validated input shape and
   returned canned data — nothing touched Postgres, Neo4j, or Redis. `go.sum`
   was empty and no Postgres/Neo4j/JWT/Kafka client libraries were declared as
   dependencies.
6. **The Python worker never queried Neo4j.** `config.py` loaded `NEO4J_URI`
   but nothing used it; `main.py` called `compute_louvain_clusters([])` on an
   empty list forever and never wrote results anywhere.
7. **Postgres wasn't configured for CDC.** `wal_level = logical` (required by
   Debezium's `pgoutput` plugin) was never set, and there was no publication
   for Debezium to attach to.
8. **No login → JWT → role-checked-route pipeline existed at all**, so FR-1/FR-2
   and the auditor/admin/journalist separation in Section 2.3 of the SRS had
   no working implementation path.

## What changed

- Removed: `apps/docs`, `apps/web`, `apps/api`, `packages/ui`,
  `packages/eslint-config`, `packages/typescript-config`, root `docker-compose.yml`.
- Consolidated to one `infra/docker-compose.yml` covering Postgres, Kafka,
  Zookeeper, Debezium Connect, Neo4j, Redis, `go-backend`, and `python-worker`.
- `packages/config-eslint` and `packages/config-typescript` are now the real,
  only shared config packages (`@ngj/config-eslint`, `@ngj/config-typescript`);
  `apps/frontend` now points at them instead of the missing `@repo/*` names.
- Postgres: full schema (`users`, `articles`, `claims`, `votes`, `appeals`)
  covering FR-1 through FR-15, `wal_level = logical`, and a
  `ngj_publication` for Debezium.
- Neo4j: constraints/indexes for `Article`, `Journalist`, `Tag`, plus the
  precompiled Cypher queries the frontend/worker use.
- Go backend rewritten to actually connect to Postgres (`pgx`) and Neo4j
  (`neo4j-go-driver`), issue and verify JWTs (`golang-jwt`), and hash
  passwords (`bcrypt`). The single global role guard was replaced by
  `Authenticate` (parses the JWT once) + `RequireRole(...)` applied per route,
  so journalist/auditor/admin routes no longer collide.
- Implemented for real: login, article creation + listing, appeals, auditor
  voting with cross-tag consensus resolution (FR-7), slashing (FR-8), and
  the retraction/tombstone flow (FR-13/14/15).
- Added a Kafka consumer that reads the Debezium CDC topic for `articles` and
  upserts the corresponding Neo4j nodes/edges — the "Data Synchronization
  Layer" described in SRS Section 5.2 now has an actual implementation path.
- Python worker now pulls the real `SEQUENCE_OF` edge graph from Neo4j, runs
  Louvain clustering on it, and writes `clusterId` back onto each node.
- Frontend `lib/api.ts` now attaches the JWT from `lib/auth.ts` to every
  request automatically instead of sending nothing.

## Round 3 — frontend redesign (this update)

The frontend previously used ad-hoc inline styles with no visual identity.
Replaced with a small, consistent design system in `apps/frontend/src/app/globals.css`:

- **Palette** — `--ink` (carbon black), `--paper` (cool onion-skin, not cream),
  `--stamp-green` (verified), `--pen-red` (false/retracted/alert), `--brass`
  (pending), `--wire-blue` (links/info). All in `globals.css` as CSS variables.
- **Type** — Newsreader (serif display, used sparingly, italic for
  annotations), IBM Plex Sans (UI), IBM Plex Mono (IDs, scores, hashes, nav
  labels). Loaded via `next/font/google` in `app/layout.tsx`.
- **Shell** — `.app-shell` / `.nav-rail` / `.app-main`: a left "spine" nav
  (`components/DashboardNav.tsx`) instead of a top bar.
- **Docket layout** — `.docket` is a two-column grid (main content + a
  margin rail) used on every dashboard page.
- **The Margin Log** (`components/MarginLog.tsx`) — the one signature
  element. It reads like a copy editor's marginal note, but every line is
  computed from real fetched data (retracted counts, pending appeals, tag
  breakdowns, draft completeness). This is the actual "what's outstanding"
  feature requested — built into the design language rather than bolted on
  as a separate widget.
- **Primitives** — `PageHeader`, `Stamp` (ink-stamp status badges: ok /
  alert / pending / neutral), restyled `Button`/`Input`, `.ledger` (table),
  `.card`, `.notice`.

Every existing page (login, all three dashboards, publish, appeals, claim
vote, compliance, public profile + graph) now uses these primitives — no
page was left on the old inline-style version.

Verified with `tsc --noEmit`, `eslint`, and a full `next build`. Google
Fonts can't be fetched from this sandbox's restricted network, so the build
was also run once with the font import stubbed out to confirm the rest of
the app compiles; the real font-loaded `layout.tsx` still passes
`tsc --noEmit`. This will fetch normally in a real dev/CI environment.

**To extend this consistently:** reuse `.card`, `.docket`, `PageHeader`, and
`MarginLog` rather than inline styles. Add new semantic colors as CSS
variables in `globals.css`, not one-off hex values in components.

## What's intentionally left as a next step

- Auditor onboarding / credential verification (NFR-6) — currently any user
  row with `role = 'auditor'` is trusted as-is; there's no upload/verification
  flow for academic or professional credentials yet.
- Self-corrected claims: there's no endpoint yet for a journalist to mark
  their own claim as self-corrected (the `self_corrected_claims` column and
  `w2 > w1` weighting exist and work — nothing currently increments it).
- Admin appeals review UI: `appeals` rows are created (FR-5) and the pulsing
  amber state (FR-9) can be driven by `appeals.status == 'active'`, but there's
  no dashboard yet for an admin to resolve an appeal.
- The frontend isn't containerized (no `Dockerfile`/compose service) — run it
  with `pnpm dev` per the README while iterating.
- Sign-up / account creation flow — accounts are currently seeded directly in
  Postgres (see README step 2); there's no self-serve registration page.

## Round 2 — frontend, Redis leaderboard, and the graph API

Added on top of the Sprint 1 rebuild above:

- **Redis leaderboard**: `internal/redisstore`, `internal/leaderboard`.
  `GET /leaderboard` reads the `leaderboard:journalist_rank` sorted set;
  `POST /articles/{id}/read` increments both the Postgres counter (source of
  truth, flows to Neo4j via CDC) and a Redis counter (NFR-3, instant reads).
  Every time a claim resolves, the author's Journalist Rank Score (formula 1)
  is recalculated and pushed into the sorted set (`consensus/voting.go`).
- **Claims**: `internal/claims` — `POST /articles/{id}/claims` (journalists
  tag `#Claim` statements, FR-3) and `GET /claims/pending` (auditors browse
  what's awaiting cross-tag consensus, FR-7).
- **Graph API**: `internal/graph` — `GET /journalists/{id}/graph` reads
  directly from Neo4j (never Postgres), returning exactly what the frontend
  needs to render: `corruptionFactor`, `readershipVolume`, `clusterId`,
  `isRetracted`. The CDC consumer (`internal/kafka/consumer.go`) now also
  computes and writes `corruptionFactor` to each Neo4j node so this endpoint
  doesn't need a second data source.
- **Frontend, end to end**:
  - `lib/auth.ts` + `components/RoleGate.tsx`: session storage and a
    role-gate wrapper applied to all three dashboard layouts.
  - `app/login/page.tsx`: real login against `POST /auth/login`, redirects by
    role.
  - `lib/crypto.ts`: generates a per-session Web Crypto keypair and signs
    article title+body (NFR-4) instead of sending a placeholder signature.
  - Journalist dashboard/publish/appeals, auditor dashboard/claim-vote, admin
    dashboard/compliance: all replaced with real forms and data fetching
    against the endpoints above (previously static placeholder text).
  - `components/LineageGraph.tsx`: real Sigma.js + graphology renderer
    fetching `/journalists/{id}/graph`, coloring nodes by Corruption Factor
    (FR-10) and sizing them by readership (FR-12).
  - `graph/hooks/useSemanticZoom.ts`: real implementation — within each
    Louvain cluster, the highest-readership node stays visible when the
    camera zooms out past half of `maxCameraRatio`; the rest hide (FR-11).
  - Verified with `pnpm install`, `next typegen`, `tsc --noEmit`, `eslint`,
    and a full `next build` — all pass clean.

