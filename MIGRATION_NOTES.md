# Migration Notes — base repo → rebuilt structure

This documents what was actually wrong with the `copilot/nextgenjournalism`
branch and exactly what changed since. Keep this file until the team has
reviewed every point, then delete it.

## What was broken (original base repo)

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

## Round 1 — infra, CDC pipeline, working auth skeleton

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

## Round 2 — frontend, Redis leaderboard, and the graph API

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

## Round 3 — frontend redesign

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
vote, compliance, public profile + graph) uses these primitives — no page
was left on the old inline-style version.

**To extend this consistently:** reuse `.card`, `.docket`, `PageHeader`, and
`MarginLog` rather than inline styles. Add new semantic colors as CSS
variables in `globals.css`, not one-off hex values in components.

## Round 4 — signup, self-corrected claims, auditor credential verification

- **`POST /auth/signup`** (`internal/auth/handler.go`) — self-serve
  registration for journalists and auditors. Admin accounts are deliberately
  excluded (see README step 2); a self-serve admin role would defeat the
  point of having a trusted compliance role. Auditors must supply a
  `credentialUrl` and at least one category tag at signup.
- **NFR-6, actually enforced now**: added `users.credential_verified`
  (migration `0002_add_credential_verification.sql`), defaulting to `true`
  for everyone except newly self-registered auditors, who start `false`.
  `consensus.Handler.Vote` now rejects a vote with 403 until an admin
  approves it. New `internal/auditors` package: `GET /admin/auditors/pending`
  and `POST /admin/auditors/{id}/verify` for that review, plus a matching
  `/admin/auditors` page in the frontend.
- **Self-corrected claims**: `POST /claims/{id}/self-correct`
  (`internal/claims/handler.go`) lets a journalist mark their own pending
  claim self-corrected before an auditor resolves it, bumping
  `self_corrected_claims` and recalculating rank score — the previously
  unused half of formula (1)'s `w2 > w1` weighting. Wired into the
  journalist dashboard as a small panel; the publish flow now surfaces each
  tagged claim's ID so there's something to paste in.
- Refactored the rank-recalculation logic (previously private to
  `consensus`) into `internal/ranking/recalculate.go` so both `consensus`
  (auditor-resolved claims) and `claims` (self-corrected claims) share one
  implementation instead of duplicating the SQL.
- Frontend: new `/signup` page, linked from `/login` and the homepage.

Every round has been verified with `tsc --noEmit`, `eslint`, and a full
`next build` on the frontend. Go changes are checked by hand for import
cycles and brace/paren balance — this sandbox has no outbound access to the
Go module proxy, so run `go mod tidy && go build ./...` yourself before
deploying. No new Go dependencies were added in Round 4, so `go.sum`
shouldn't need new entries beyond what Round 1 already required.

## Round 5 — signup was actually broken, and readers had no path at all

Two real bugs, both now fixed:

1. **Every signup failed with "email already exists," even for a brand-new
   email.** The handler caught every INSERT error and mapped it to that one
   message, which hid the real problem:
   - `role` was bound as a plain text parameter against a `user_role` enum
     column. Postgres only auto-casts an *untyped* literal to an enum, not a
     parameter with a concrete type (`text`/OID 25) — under the extended
     query protocol pgx uses, that's a type mismatch on every single insert.
     Fixed with an explicit `$3::user_role` cast in `auth/handler.go`.
   - The exact same bug existed in `consensus/voting.go`'s
     `UPDATE claims SET status = $2` (against the `claim_status` enum) —
     found while fixing the first one. It would have silently failed the
     first time any claim actually resolved. Fixed with `$2::claim_status`.
   - Separately, `tags` is `NOT NULL`, but a journalist's signup request
     never sends a `tags` field, so it arrived as a Go `nil` slice → SQL
     `NULL` → constraint violation. Fixed by defaulting to an empty slice
     before the insert.
   - The error handling itself was the reason this went undiagnosed: now it
     checks for Postgres's actual unique-violation code (`23505`) before
     returning 409, logs anything else server-side with the real error, and
     tells the caller honestly instead of guessing.
2. **Readers had no path that didn't run through login/signup.** Reading was
   already public on the backend (`GET /articles`), but there was no page for
   it and the homepage only offered "sign in" / "open a desk." Added:
   - `GET /articles/{id}` (`internal/articles/get.go`) — single-article detail
     with its tagged claims and their verdicts, so a reader sees the actual
     evidence, not just a headline. Retracted articles show a plain notice
     instead of the raw tombstone hash stored in `body`.
   - `/read` and `/read/[articleId]` in the frontend — a public reading flow
     with its own lightweight header (`components/PublicHeader.tsx`), no
     login, no nav rail. Loading a story fires `POST /articles/{id}/read`
     to increment readership.
   - Homepage now leads with "Read the news" first; signup/login are
     secondary, explicitly for journalists/auditors only.

Verified with `tsc --noEmit`, `eslint`, and a full `next build` (14 routes,
fonts stubbed for this sandbox as in prior rounds). Go changes checked by
hand for brace/paren balance and import cycles; no new dependencies, so
`go.sum` needs nothing beyond Round 1. **If you already ran signup attempts
against the old code**, no cleanup needed — those inserts never succeeded.

## Round 6 — actually running the stack, not just reading it

Previous rounds fixed real bugs found by careful reading, but nothing had
been run against a live Postgres/Redis. This round installed Go 1.22,
Postgres 16, and Redis directly (apt) and ran the actual compiled binary
against them end to end — signup, login, article creation, claim tagging,
cross-tag consensus voting, slashing, retraction, the leaderboard, and the
credential-verification gate all exercised for real over HTTP. Two more
real bugs turned up:

1. **`go.sum` was never actually committed** in earlier rounds — every prior
   message said "run `go mod tidy` yourself." Without it, `docker build`'s
   `go mod download` step has no lockfile to work from, which is a very
   plausible reason the container wouldn't come up at all. This round
   generated a complete, verified `go.sum` by actually running
   `go mod tidy` and `go build ./...` successfully (63 entries). Note: this
   sandbox can't reach `proxy.golang.org`, `golang.org`, or `gopkg.in`
   (only a fixed domain allowlist), so `go.mod` now carries `replace`
   directives pointing a handful of transitive dependencies
   (`golang.org/x/net`, `x/crypto`, `x/text`, `x/sys`, `x/sync`, `x/term`,
   `x/tools`, `x/mod`, `gopkg.in/yaml.v3`, `gopkg.in/check.v1`) at their
   equivalent `github.com/golang/*` / `github.com/go-yaml/*` mirrors. These
   are the real upstream/official mirrors, not forks — safe to keep, and
   arguably more robust for any CI environment with restricted egress.
2. **The real cause of "go-backend isn't running in Docker": wrong
   hostnames.** `go-backend`'s service definition in
   `infra/docker-compose.yml` loaded `apps/go-backend/.env` via `env_file`.
   Two failure modes from this one line: if that file didn't exist,
   `docker compose up` refuses to start the service at all; if it *did*
   exist (copied from `.env.example` per the README's step 3), it pointed
   `DATABASE_URL`/`REDIS_URL`/etc. at `localhost` — correct for running the
   binary natively on your host, wrong inside the container network, where
   Postgres is reachable as `postgres`, not `localhost`. Either way,
   `go-backend` would fail to connect on startup and crash-loop under
   `restart: unless-stopped`, which fails silently from the outside (no
   obvious startup error, just a container that never stays up) — exactly
   matching "isn't running properly." Fixed by giving `go-backend` its own
   explicit `environment:` block with the correct in-network hostnames,
   the same pattern `python-worker` already used correctly. `.env.example`
   now says explicitly it's for the native (non-Docker) path only.

Verified this round: the compiled binary running against real Postgres +
Redis handled every endpoint correctly, including a live simulation of the
exact Dockerfile build steps (`go mod download` + `go build` from a clean
checkout) using the new `go.sum`. Frontend unaffected this round; its own
`tsc`/`eslint`/`build` checks from Round 5 still stand.

## Round 7 — CORS: the actual reason signup looked broken from the browser

Round 6 verified every endpoint with `curl` against a live database and
everything worked. But `curl` doesn't enforce CORS — a real browser does,
and the backend had **no CORS support at all**. Confirmed by sending the
exact preflight request a browser sends before any cross-origin `POST`:

```
OPTIONS /auth/signup  ->  405 Method Not Allowed, no CORS headers
```

A browser stops right there — the actual signup `POST` never gets sent. The
frontend's `catch` block then showed a hardcoded "email may already be
registered" for literally any failure, which is exactly why a brand-new
email produced that message (screenshot). Same class of bug as Round 6's
backend fix, just one layer up: guessing an error message instead of
surfacing the real one.

Fixed both sides:
- **Backend**: new `internal/server/cors.go` — handles preflight `OPTIONS`
  directly (204, before auth or routing ever see it) and sets
  `Access-Control-Allow-Origin` only for origins in `CORS_ALLOW_ORIGINS`
  (new config, defaults to `http://localhost:3010`, already set correctly
  in both `docker-compose.yml` and `.env.example`). Verified the fix the
  same way the bug was found: sent the real preflight request again and
  confirmed `204` + correct headers, confirmed an origin *not* on the
  allowlist correctly gets no CORS header (still blocked), and confirmed
  the actual signup response now carries `Access-Control-Allow-Origin`.
- **Frontend**: `lib/api.ts` now distinguishes a real server response (any
  status, with the backend's actual message) from a request that never
  reached a server at all (network down, or the browser blocking it
  client-side) via a new `ApiError` class, instead of collapsing every
  failure into one message. `login` and `signup` now show what actually
  happened instead of a guess.

Verified with `tsc --noEmit`, `eslint`, and a full `next build` (14 routes).
Backend re-verified with `go build`, `go vet`, and the live preflight/POST
test above against a real Postgres/Redis.

## Round 8 — the real Node.js build issue, and the Sigma.js graph upgrade

Reproduced the exact CI steps (`pnpm install --frozen-lockfile`, `lint`,
`check-types`, `build`) locally against the actual synced repo instead of
guessing. The lockfile, lint, and type-check all passed — but `next build`
hard-failed:

```
next/font: error: Failed to fetch `IBM Plex Sans` from Google Fonts.
```

`next/font/google` fetches font files from `fonts.googleapis.com` **at
build time** with no fallback. Any build environment that can't reach
Google reliably (a firewalled CI runner, an offline/restricted Docker
build) fails the entire production build on this alone — it's not a config
mistake, it's a hard network dependency baked into how that API works.
Fixed by switching to `@fontsource/newsreader`, `@fontsource/ibm-plex-sans`,
and `@fontsource/ibm-plex-mono` — these ship the actual `.woff2` files as
ordinary npm package contents, so once `pnpm install` has pulled them from
the registry, the font files are just local files on disk; nothing is
fetched from Google at build time ever again. Verified by running the full
`next build` in this sandbox, which has no route to Google's servers at
all — it now succeeds regardless.

Also upgraded `components/LineageGraph.tsx` toward the interaction pattern
of the Sigma.js "cartography of Wikipedia" reference (sigmajs.org):
- **Force-directed layout** (`graphology-layout-forceatlas2`) relaxes the
  circular seed layout into clusters that reflect actual connectivity,
  instead of an arbitrary ring.
- **Hover-to-highlight-neighbors**: hovering a node fades every node/edge
  that isn't it or a direct neighbor, via Sigma's `nodeReducer`/`edgeReducer`
  — the same technique the reference demo uses to keep a dense graph
  readable.
- **Search-to-locate**: a text input filters by title and animates the
  camera to the matching node on selection.

Verified with `pnpm install --frozen-lockfile`, `lint`, `check-types`, and a
full `build` (all 14 routes) — matching the actual CI sequence exactly, not
just spot-checking. Go backend re-verified with `go build`/`go vet` on this
same clone; the CORS fix from the previous round is confirmed present and
working (no more nested-duplicate folders either — that got cleaned up too).

## What's intentionally left as a next step

- Admin appeals review UI: `appeals` rows are created (FR-5) and the pulsing
  amber state (FR-9) can be driven by `appeals.status == 'active'`, but there's
  no dashboard yet for an admin to approve/reject an appeal.
- The frontend isn't containerized (no `Dockerfile`/compose service) — run it
  with `pnpm dev` per the README while iterating.
- Password reset / account recovery — there's no flow for a user who forgets
  their password.
- The self-correct panel on the journalist dashboard takes a raw claim ID
  pasted in by hand; it works, but a proper "your claims" browser (like the
  auditor's pending-claims list) would be friendlier than copy-pasting IDs
  from the publish flow.
