# SRS Feature Status — F-01 through F-22, EX-01 through EX-03

Cross-referenced against `470_Project_Group5.pdf` (the fixed-feature-ID SRS).
Status reflects what's actually wired end-to-end and runnable, not just
scaffolded. "Partial" means something real exists but doesn't fully satisfy
the acceptance criterion in Section 7 of the SRS.

| ID | Feature | Status | Where | Gap, if any |
|---|---|---|---|---|
| F-01 | Individual Account Directed Graphs | **Done** | `GET /journalists/{id}/graph`, `LineageGraph.tsx` | — |
| F-02 | Transparent Article Lineage | **Done** | `SEQUENCE_OF` edges, `parentArticleId` | — |
| F-03 | Dynamic Node Sizing | **Done** | `readershipToSize()` in `sigma-config.ts` | — |
| F-04 | WebGL Epistemic Graphs | **Done** | Sigma.js + ForceAtlas2 (this round) | — |
| F-05 | Semantic Zooming | **Done** | `useSemanticZoom.ts` | — |
| F-06 | Color Intensity Grading | **Done** | `corruptionToColor()` | — |
| F-07 | Community-Detection Grouping | **Done** | Python worker, Louvain, `clusterId` | — |
| F-08 | Time-Based Clustering | **Missing** | — | No chronological bucketing exists anywhere; nodes aren't organized by time period at all. |
| F-09 | Claim Extraction Tool | **Done** | `POST /articles/{id}/claims`, publish flow UI | — |
| F-10 | Reputation-Weighted Decentralized Consensus | **Done** | `consensus/voting.go` | — |
| F-11 | Cross-Tag Validation Matrix | **Done** | `TryResolve()` non-overlapping-tag check | — |
| F-12 | Auditor Reputation Staking | **Done** | `votes.stake` | — |
| F-13 | Slashing Protocol | **Done** | `ApplySlashing()` | — |
| F-14 | Proof of Evidence Appeals Protocol | **Partial** | `POST /appeals` | Stakes rank score, but there's no field or mechanism to actually submit new evidence — just an article ID and a stake percentage. |
| F-15 | Under Dispute Visual State | **Missing** | — | `appeals` rows exist and are queryable, but nothing renders the pulsing amber/orange node state anywhere — not in `LineageGraph.tsx`, not as a graph node attribute. |
| F-16 | Journalist Rank Score (R) | **Done** | `ranking.JournalistRankScore()` | — |
| F-17 | Logarithmic Volume Dampener | **Done** | `math.Log10` in the same formula | — |
| F-18 | Integrity-Incentivized Self-Correction | **Done** | `POST /claims/{id}/self-correct`, `w2 > w1` | — |
| F-19 | Instant Global Leaderboards | **Partial** | `GET /leaderboard` (Redis sorted set) | Backend is real and fast; there's no frontend page that displays it anywhere. |
| F-20 | Immutability Lock | **Missing** | `articles.immutability_locked` column exists, unused | No hard-delete endpoint exists at all (so nothing to lock against yet), and no logic ever sets this column based on the readership threshold. |
| F-21 | Retracted State Compliance | **Done** | `compliance/tombstone.go` | — |
| F-22 | Cryptographic Author Signatures | **Partial** | `lib/crypto.ts` signs client-side, `signature` column stores it | The backend accepts and stores whatever signature string it's given — it never verifies the signature against the journalist's public key. Right now this is closer to "signature-shaped field" than a verified crypto guarantee. |
| EX-01 | System Integration | **Mostly done** | `docker-compose.yml` wires Postgres/Kafka/Debezium/Neo4j/Redis/Go/Python | Frontend isn't containerized yet — runs via `pnpm dev` alongside the rest. |
| EX-02 | Testing and Acceptance Validation | **Missing** | — | No automated test suite exists — no Go unit/integration tests, no frontend test runner. Everything verified so far has been manual (`curl` against a live stack, `tsc`/`eslint`/`build`), which is real but isn't the same as a checked-in, repeatable test suite. |
| EX-03 | Final Documentation and Deployment Package | **Partial** | `README.md`, `MIGRATION_NOTES.md` | Solid technical docs exist; presentation-ready screenshots and a packaged deployment bundle are still on you. |

## Suggested priority order for what's next

Given the size of what's left, roughly cheapest-to-most-involved:

1. **F-15 (Under Dispute Visual State)** — the data already exists
   (`appeals.status`); this is almost entirely a `LineageGraph.tsx` change
   (a node attribute + a CSS/reducer-driven pulse animation) plus one new
   field on the graph API response.
2. **F-19 frontend** — a `/leaderboard` page is a straightforward read-only
   list against an endpoint that already works.
3. **F-08 (Time-Based Clustering)** — needs a decision on bucketing rule
   (by month? by "recent" vs "archive"?) before it's an hour of work rather
   than a guess.
4. **F-14 evidence submission** — needs a schema decision (a text field? a
   file/URL? both?) before implementation.
5. **F-20 (Immutability Lock)** — there's no hard-delete endpoint to guard
   yet, so this is really "decide whether hard delete should exist at all,"
   which is a product question as much as a code one.
6. **F-22 real signature verification** — the most involved: needs a
   keypair-registration step at signup, storing the public key, and the
   backend actually verifying each submission against it.
7. **EX-02 (tests)** — valuable regardless of the above; a real Go test
   suite for `ranking`/`consensus` (the math has zero test coverage right
   now, despite being the most formula-heavy part of the system) would be
   the highest-leverage place to start.
