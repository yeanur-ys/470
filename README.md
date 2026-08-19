# nextGENjournalism

A transparent journalism platform: role-based Next.js dashboards, a Go API +
CDC-sync backend, a Python graph-analysis worker, and a
Postgres/Neo4j/Redis/Kafka data layer. `apps/frontend` and `apps/go-backend`
are each internally structured as MVC (see below); the repo as a whole is a
pnpm/Turborepo workspace. See `SEED_DATA.md` for the small hand-written demo
dataset.

## Layout

```text
nextGENjournalism/
├── .github/workflows/            # CI: frontend-ci.yml, backend-ci.yml
├── apps/
│   ├── frontend/                 # Next.js App Router — MVC, see below
│   ├── go-backend/                # Go API — MVC, see below
│   └── python-worker/             # Louvain clustering worker (reads/writes Neo4j)
├── packages/
│   ├── config-eslint/             # @ngj/config-eslint
│   ├── config-typescript/         # @ngj/config-typescript
│   └── database/                  # @ngj/database: postgres schema, neo4j queries, redis keys
├── infra/
│   ├── docker-compose.yml         # single source of truth for local infra
│   ├── postgres/postgresql.conf   # wal_level=logical for Debezium
│   ├── debezium/register-postgres.json
│   └── neo4j/conf/neo4j.conf
├── package.json
├── pnpm-workspace.yaml
└── turbo.json
```

### Frontend MVC (`apps/frontend/src/`)

| Layer | Path | Role |
|---|---|---|
| View | `app/**/page.tsx`, `layout.tsx` | Route components — render only, no fetching or business logic |
| Controller | `hooks/use*.ts` | Data fetching, local state, and view-specific business rules (e.g. `useJournalistDashboard` decides which margin notes to show) |
| Model | `lib/models/*.ts` | One file per backend domain (`articles.ts`, `claims.ts`, `votes.ts`, …) — thin wrappers over `lib/api.ts` calling go-backend endpoints |
| Shared | `types/domain.ts` | Types shared by every layer |
| — | `components/` | Reusable presentational pieces (`DashboardNav`, `HighlightableText`, `ui/Button`, …) shared across views |
| — | `graph/` | Sigma.js/WebGL-specific rendering: shaders, semantic zoom, cluster labels — used by `components/LineageGraph.tsx` and `app/graph/page.tsx` |
| — | `lib/auth.ts`, `lib/crypto.ts` | Session storage/role helpers and client-side article signing, independent of any one domain |

A page never calls `lib/models` directly — it calls a hook, which calls a
model function. Keeping business rules (what counts as noteworthy, what an
auditor's available stake is) in hooks rather than in `lib/models` or the page
component is deliberate: models stay dumb HTTP wrappers, views stay dumb
render functions.

### Backend MVC (`apps/go-backend/internal/`)

| Layer | Path | Role |
|---|---|---|
| Model | `models/` | Domain types + all DB/business logic (`article.go`, `claim.go`, `vote.go`, `user.go`, `ranking.go` for the trust-weight formula) — the only layer that talks to Postgres/Neo4j directly |
| Controller | `controllers/` | HTTP handlers — decode request, call a model function, hand the result to a view |
| View | `views/` | Response-shape structs (e.g. `SessionView`) — keeps JSON wire format decisions out of both controllers and models |
| — | `server/` | Route table (`routes.go`), middleware, CORS |
| — | `auth/` | JWT issuing/verification, request-context helpers |
| — | `db/`, `redisstore/`, `kafka/` | Connection setup for Postgres/Neo4j, Redis, and the Kafka producer/consumer used by CDC-sync |
| — | `config/` | Env var loading |

`cmd/api/main.go` wires all of the above together and starts the HTTP server.

## Prerequisites

- Node.js 20+, `corepack enable` (for pnpm 9)
- Go 1.22+
- Python 3.11+
- Docker + Docker Compose

## 1. Bring up infrastructure

```bash
pnpm infra:up          # postgres, kafka, zookeeper, debezium, neo4j, redis
pnpm infra:logs        # tail logs; ctrl-C to stop tailing (containers keep running)
```

Wait until `pg_isready` passes (compose healthcheck), then register the
Debezium connector so Postgres writes start streaming into Kafka:

```bash
pnpm debezium:register
```

**If you already have a running database from before this update**, apply the
pending migrations in order (adds `users.credential_verified` for NFR-6, then
the auditor reputation ledger + locks):

```bash
docker exec -i ngj-postgres psql -U ngj -d nextgenjournalism < packages/database/postgres/migrations/0002_add_credential_verification.sql
docker exec -i ngj-postgres psql -U ngj -d nextgenjournalism < packages/database/postgres/migrations/0003_auditor_reputation_and_locks.sql
```

## 2. Seed an admin account

Journalists and auditors can now sign up themselves (step 5). Admin accounts
are deliberately excluded from self-serve signup, so seed one directly:

```bash
docker exec -it ngj-postgres psql -U ngj -d nextgenjournalism -c "
INSERT INTO users (email, password_hash, role, display_name) VALUES
  ('admin@example.com', crypt('password123', gen_salt('bf')), 'admin', 'Demo Admin');
"
```

> This uses Postgres's `pgcrypto` extension for `crypt()`/`gen_salt()`. If it's
> not enabled: `docker exec -it ngj-postgres psql -U ngj -d nextgenjournalism -c "CREATE EXTENSION IF NOT EXISTS pgcrypto;"`
> first. The Go backend itself uses `bcrypt` for every account created through
> `/auth/signup` — this seed step just needs *some* bcrypt-compatible hash.

**Want more than one empty account to look at?** `SEED_DATA.md` has a small
demo dataset — two journalists, three auditors, article chains, a dispute and
a retraction.

### Large demo dataset (recommended)

For anything involving the graph, use the large seed instead. Community
detection, semantic zooming and the leaderboard all need hundreds of nodes
before they mean anything:

```bash
# Apply migrations first (0003 adds the auditor reputation ledger + locks)
docker exec -i ngj-postgres psql -U ngj -d nextgenjournalism \
  < packages/database/postgres/migrations/0003_auditor_reputation_and_locks.sql

docker exec -i ngj-postgres psql -U ngj -d nextgenjournalism \
  < packages/database/postgres/seed_large.sql
```

Generates 14 journalists, 16 auditors (3 awaiting credential approval), 900
articles in lineage chains across 12 claim categories, ~2,250 claims in every
status, ~1,500 settled votes, 12 active appeals and 8 retractions. Every
password is `password123`; accounts are `journalist1@demo.nextgenjournalism.test`
… , `auditor1@…`, `admin@demo.nextgenjournalism.test`.

It's deterministic (`setseed`) and safe to re-run — it clears only rows
belonging to those demo accounts (by email domain) before regenerating them,
so it won't touch anything created against a demo account through the running
API. **Register the Debezium connector first** (step above), so the inserts
stream into Neo4j through the CDC pipeline rather than needing a separate
graph load.

## 3. Run the Go backend

```bash
cd apps/go-backend
cp .env.example .env
go mod tidy          # resolves go.sum from go.mod — needs network access
go run ./cmd/api
```

Confirm it's up:

```bash
curl http://localhost:8080/health
```

Sign up as a journalist and grab a token:

```bash
curl -X POST http://localhost:8080/auth/signup \
  -H "Content-Type: application/json" \
  -d '{"email":"journalist@example.com","password":"password123","displayName":"Demo Journalist","role":"journalist"}'
```

Or for an auditor (requires a credential URL + at least one tag — you won't be
able to vote until an admin approves it, see the `/admin/auditors/*` routes
below). New auditors start with a small bootstrap reputation (`rank_score`
10.0) so their first vote isn't blocked by a zero stakeable balance once
they're verified:

```bash
curl -X POST http://localhost:8080/auth/signup \
  -H "Content-Type: application/json" \
  -d '{"email":"auditor1@example.com","password":"password123","displayName":"Demo Auditor","role":"auditor","credentialUrl":"https://orcid.org/0000-0000-0000-0000","tags":["Economic Analyst"]}'
```

Use the returned token on protected routes:

```bash
curl -X POST http://localhost:8080/articles \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"title":"First story","body":"...","signature":"dev-sig"}'
```

## 4. Run the Python worker

```bash
cd apps/python-worker
python3 -m venv .venv && source .venv/bin/activate
pip install -r requirements.txt
NEO4J_URI=bolt://localhost:7687 NEO4J_PASSWORD=ngj_dev_password python src/main.py
```

It polls Neo4j every `POLL_INTERVAL_SECONDS` (default 30s), re-clusters the
`SEQUENCE_OF` graph with Louvain, and writes `clusterId` back onto each
`Article` node for the frontend's semantic zoom to read.

## 5. Run the frontend

```bash
pnpm install
cd apps/frontend
cp .env.example .env.local 2>/dev/null || echo "NEXT_PUBLIC_API_URL=http://localhost:8080" > .env.local
pnpm dev
```

Open http://localhost:3010 → **Read the news** needs no account at all —
`/read` lists every story, `/read/[id]` shows the full text plus every
tagged claim's verdict. Select any passage of the body text to highlight it
(amber/green/blue) — highlights persist per-article in the browser via
`localStorage`, since readers have no accounts to save them against. That's
the entire reader experience; it never touches `/login` or `/signup`.

For the other three roles: **Open a desk** to sign up as a journalist or
auditor, or **Sign in** if you already have an account (admins are seeded
directly — step 2). The header shows a **Profile** link and **Log out** once
signed in. Each role lands on its own dashboard:

- **Journalist** (`/journalist/dashboard`): lists your articles, links to
  **Publish** (writes + client-side signs an article, then lets you tag
  `#Claim` statements) and **Appeals** (stake rank score to dispute a ruling).
  The dashboard also has a **self-correct** panel — paste a claim ID from the
  publish flow to mark it self-corrected before an auditor resolves it.
  Their **profile** (`/profile/[journalistId]`) is public — see below.
- **Auditor** (`/auditor/dashboard`): lists claims awaiting cross-tag
  consensus; click into one to stake reputation and vote. New auditor
  signups can't vote until an admin approves their linked credential
  (NFR-6) — see Admin below. Their **profile** (`/auditor/profile`) is
  private to them and shows reputation, trust weight, available-to-stake
  balance, vote record, and linked credential.
- **Admin** (`/admin/dashboard`): lists every article; **Auditors** reviews
  and approves newly signed-up auditors' credentials; **Compliance** applies
  a GDPR/DMCA retraction (tombstones the content, greys out the node,
  deducts the author's rank score). Their **profile** (`/admin/profile`) is
  an identity card with quick links into these sections.
- **Public profile** (`/profile/[journalistId]`): anyone can view a
  journalist's lineage graph — a live Sigma.js/WebGL rendering of their
  article graph read straight from Neo4j, colored by Corruption Factor and
  sized by readership. Only journalists have a public profile this way;
  auditors and admins don't have a public directory, so their profile pages
  live under their own role tree instead (above).
- **Epistemic graph** (`/graph`): the whole platform in one view, also public.
  Nodes are sized by readership (FR-12) and coloured either by Louvain topic
  community (the default here) or by Corruption Factor (FR-10) — toggle at the
  top. Each community is labelled on the canvas with the claim category that
  dominates it. Scroll to zoom (semantic zoom collapses communities to their
  most-read stories as you pull back), hover a node to isolate its
  neighbourhood, click for details, and click a legend chip to hide a
  community or time period. Articles under an active appeal pulse amber (FR-9).

## API reference (go-backend)

| Method | Path | Role | Notes |
|---|---|---|---|
| GET | `/health` | public | liveness check |
| POST | `/auth/login` | public | returns `{ token, role, userId }` |
| POST | `/auth/signup` | public | journalist or auditor only — see NFR-6 note above |
| GET | `/articles` | public | latest 100 articles |
| GET | `/articles/{id}` | public | single article + its tagged claims and their verdicts — the reader page |
| GET | `/articles/mine` | journalist | your own articles |
| POST | `/articles` | journalist | create (FR-3/FR-4) |
| POST | `/articles/{id}/read` | public | increments readership (Postgres + Redis) |
| POST | `/articles/{id}/claims` | journalist | tag a `#Claim` (FR-3) |
| POST | `/claims/{id}/self-correct` | journalist | mark your own pending claim self-corrected |
| POST | `/appeals` | journalist | stake rank score to dispute (FR-5) |
| GET | `/claims/pending` | auditor | claims awaiting consensus (FR-7) |
| POST | `/claims/{id}/votes` | auditor | requires `credential_verified`; stake + vote (FR-6); auto-resolves + slashes (FR-7/FR-8) |
| POST | `/admin/articles/{id}/retract` | admin | tombstone + rank penalty (FR-13/14/15) |
| GET | `/admin/auditors/pending` | admin | auditors awaiting credential review (NFR-6) |
| POST | `/admin/auditors/{id}/verify` | admin | approve an auditor's linked credential |
| GET | `/journalists/{id}/graph` | public | nodes/edges for one journalist's Sigma.js graph, read from Neo4j |
| GET | `/graph` | public | the platform-wide epistemic graph: every article, lineage + co-tag edges, Louvain clusters named by dominant tag (`?limit=` up to 10000) |
| GET | `/leaderboard` | public | top 50 by rank score (Redis sorted set) |

## Or run everything through Docker

```bash
pnpm infra:up   # builds and starts go-backend + python-worker too
```

`go-backend`'s environment (Postgres/Redis/Neo4j/Kafka hostnames) is set
directly in `infra/docker-compose.yml` for the container network — you don't
need to create `apps/go-backend/.env` for this path. That file (from
`.env.example`) is only for running the binary natively on your host machine
(step 3), where those services are reached via `localhost` instead of their
in-network service names.

(The frontend isn't containerized yet — run it with `pnpm dev` per step 5
while iterating; add it to `infra/docker-compose.yml` once its Dockerfile
exists.)

## Workspace-wide checks

```bash
pnpm lint
pnpm check-types
pnpm build
```
