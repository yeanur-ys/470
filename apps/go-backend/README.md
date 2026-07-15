# nextGENjournalism

A monorepo for a transparent journalism platform: role-based Next.js
dashboards, a Go API + CDC-sync backend, a Python graph-analysis worker, and a
Postgres/Neo4j/Redis/Kafka data layer. See `SRS.pdf` (or your original
`Document_470.pdf`) for the full requirements and `MIGRATION_NOTES.md` for
exactly what was fixed vs. the previous `copilot/nextgenjournalism` branch.

## Layout

```text
nextGENjournalism/
├── .github/workflows/            # CI: frontend-ci.yml, backend-ci.yml
├── apps/
│   ├── frontend/                 # Next.js App Router, role dashboards, Sigma.js graph
│   ├── go-backend/                # Go API: auth, articles, consensus, compliance, CDC-sync
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
new migration (adds `users.credential_verified` for NFR-6):

```bash
docker exec -i ngj-postgres psql -U ngj -d nextgenjournalism < packages/database/postgres/migrations/0002_add_credential_verification.sql
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
able to vote until an admin approves it, see the `/admin/auditors/*` routes below):

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
tagged claim's verdict. That's the entire reader experience; it never
touches `/login` or `/signup`.

For the other three roles: **Open a desk** to sign up as a journalist or
auditor, or **Sign in** if you already have an account (admins are seeded
directly — step 2). Each role lands on its own dashboard:

- **Journalist** (`/journalist/dashboard`): lists your articles, links to
  **Publish** (writes + client-side signs an article, then lets you tag
  `#Claim` statements) and **Appeals** (stake rank score to dispute a ruling).
  The dashboard also has a **self-correct** panel — paste a claim ID from the
  publish flow to mark it self-corrected before an auditor resolves it.
- **Auditor** (`/auditor/dashboard`): lists claims awaiting cross-tag
  consensus; click into one to stake reputation and vote. New auditor
  signups can't vote until an admin approves their linked credential
  (NFR-6) — see Admin below.
- **Admin** (`/admin/dashboard`): lists every article; **Auditors** reviews
  and approves newly signed-up auditors' credentials; **Compliance** applies
  a GDPR/DMCA retraction (tombstones the content, greys out the node,
  deducts the author's rank score).
- **Public profile** (`/profile/[journalistId]`): anyone can view a
  journalist's lineage graph — a live Sigma.js/WebGL rendering of their
  article graph read straight from Neo4j, colored by Corruption Factor and
  sized by readership.

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
| GET | `/journalists/{id}/graph` | public | nodes/edges for the Sigma.js graph, read from Neo4j |
| GET | `/leaderboard` | public | top 50 by rank score (Redis sorted set) |

## Or run everything through Docker

```bash
pnpm infra:up   # builds and starts go-backend + python-worker too
```

(The frontend isn't containerized yet — run it with `pnpm dev` per step 5
while iterating; add it to `infra/docker-compose.yml` once its Dockerfile
exists.)

## Workspace-wide checks

```bash
pnpm lint
pnpm check-types
pnpm build
```
