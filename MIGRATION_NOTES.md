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

## What's intentionally left as a next step

This covers Sprint 1 (infra + CDC + a working auth/API skeleton) plus the
consensus/compliance business logic. Still open, in the order the SRS's own
sprint plan implies:

- Wiring `apps/frontend`'s dashboards to the new endpoints (currently static
  layouts with no data fetching).
- Rendering the Sigma.js graph against live `/articles` + Neo4j data and
  driving node color from `CorruptionFactor` (FR-10) client-side.
- Redis: writing read-count increments and the rank-score leaderboard
  (Sorted Set) — the schema/keys exist (`packages/database/redis`) but nothing
  populates them yet.
- Auditor onboarding / credential verification (NFR-6) — currently any user
  row with `role = 'auditor'` is trusted as-is.
- Real password reset for `password_hash` — no user is seeded yet; see the
  README for a one-off `psql` insert to create your first accounts.
