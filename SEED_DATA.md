# Seed Data

Two files populate enough realistic data to actually see the graph features
(cluster legend, era legend, corruption-factor coloring, the pulsing
under-dispute state) without publishing everything by hand first.

- `packages/database/postgres/seed.sql` — users, articles, claims, one vote,
  one active appeal.
- `packages/database/neo4j/seed.cypher` — the same articles mirrored directly
  into the graph (with `clusterId`/`createdAt` precomputed), so the graph
  renders immediately without needing Kafka + Debezium + the Python worker
  running first. In a real deployment this data flows in automatically via
  the CDC pipeline the moment an article is created in Postgres — this file
  exists purely so you can see the graph without also standing up the whole
  pipeline just to try it.

## Load it

```bash
# Postgres
docker exec -i ngj-postgres psql -U ngj -d nextgenjournalism < packages/database/postgres/seed.sql

# Neo4j (adjust host/port/credentials if you changed them)
docker exec -i ngj-neo4j cypher-shell -u neo4j -p ngj_dev_password < packages/database/neo4j/seed.cypher
```

Both are safe to re-run — articles/claims are upserted by fixed ID and the
Postgres side skips rows that already exist.

> **Note on the Neo4j seed**: this sandbox couldn't install Neo4j to actually
> run it (no route to Neo4j's own package repo from here), so
> `seed.cypher` is checked by hand against the same `MERGE`/`SET` patterns
> already used in `packages/database/neo4j/queries.ts` and
> `internal/kafka/consumer.go`, not verified by executing it. If anything in
> it errors when you run it for real, let me know exactly what Neo4j says
> and I'll fix it properly rather than guessing again.

## What you'll see

Sign in as `journalist1@example.com` / `password123` (Amara Osei) and open
her public profile graph:

- **Two clusters**: a 4-story "budget" chain and a 2-story "transit" chain,
  toggleable in the cluster legend.
- **A pulsing amber node** — "Officials Deny Pension Mismanagement" has an
  active appeal filed against it.
- **Color variation**: that same node and its neighbors show the Corruption
  Factor gradient from neutral toward red; the rest stay clean.
- **Size variation**: readership spans 800 to 10,000 across the two chains.
- **Era spread**: the budget chain's oldest story is 140 days back ("Older"),
  its most recent 2 days back ("Last 30 days") — the era legend should show
  more than one bucket.

Sign in as `journalist2@example.com` / `password123` (Devon Cole) to see a
third cluster (a layoffs story with heavier false-claim concentration and a
self-correction), plus one retracted article rendered greyed-out.

Sign in as `auditor1@example.com` (verified, tag: Economic Analyst) or
`auditor2@example.com` (verified, tag: Geopolitical Analyst) to see resolved
claims on their dashboard's history, or `auditor3@example.com` (**not yet
verified** — sign in as `admin@example.com` to approve her credentials first,
under **Auditors** in the admin nav) to see the credential-verification gate
firsthand: she can sign in immediately but can't vote until approved.

Two claims are seeded `pending` with no votes yet — sign in as a verified
auditor and check `/auditor/dashboard` to actually cast votes and watch
cross-tag consensus resolve one live.
