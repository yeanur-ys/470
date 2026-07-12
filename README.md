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

## 2. Seed a user for each role

The schema doesn't ship demo data. Create one login per role so you can test
the JWT/role-guard pipeline:

```bash
docker exec -it ngj-postgres psql -U ngj -d nextgenjournalism -c "
INSERT INTO users (email, password_hash, role, display_name, tags) VALUES
  ('journalist@example.com', crypt('password123', gen_salt('bf')), 'journalist', 'Demo Journalist', '{}'),
  ('auditor1@example.com',   crypt('password123', gen_salt('bf')), 'auditor',    'Demo Auditor 1', '{\"Economic Analyst\"}'),
  ('auditor2@example.com',   crypt('password123', gen_salt('bf')), 'auditor',    'Demo Auditor 2', '{\"Geopolitical Analyst\"}'),
  ('admin@example.com',      crypt('password123', gen_salt('bf')), 'admin',      'Demo Admin', '{}');
"
```

> This uses Postgres's `pgcrypto` extension for `crypt()`/`gen_salt()`. If it's
> not enabled: `docker exec -it ngj-postgres psql -U ngj -d nextgenjournalism -c "CREATE EXTENSION IF NOT EXISTS pgcrypto;"`
> first. The Go backend itself uses `bcrypt` on the Go side for any users it
> creates later — this seed step just needs *some* bcrypt-compatible hash in
> the table to log in with.

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

Log in and grab a token:

```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"journalist@example.com","password":"password123"}'
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

Open http://localhost:3010.

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
