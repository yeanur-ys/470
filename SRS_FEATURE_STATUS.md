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
| F-04 | WebGL Epistemic Graphs | **Done** | Sigma.js + ForceAtlas2, hover-highlight, search, cluster legend (matches sigmajs.org/demo's interaction pattern) | — |
| F-05 | Semantic Zooming | **Done** | `useSemanticZoom.ts` | — |
| F-06 | Color Intensity Grading | **Done** | `corruptionToColor()` | — |
| F-07 | Community-Detection Grouping | **Done** | Python worker, Louvain, `clusterId` | — |
| F-08 | Time-Based Clustering | **Done** | `sigma-config.ts` (`articleEra`), `LineageGraph.tsx` era legend | Buckets: Last 30 days / Last 12 months / Older — a fixed, simple rule rather than the SRS specifying an exact one. |
| F-09 | Claim Extraction Tool | **Done** | `POST /articles/{id}/claims`, publish flow UI | — |
| F-10 | Reputation-Weighted Decentralized Consensus | **Done** | `consensus/voting.go` | — |
| F-11 | Cross-Tag Validation Matrix | **Done** | `TryResolve()` non-overlapping-tag check | — |
| F-12 | Auditor Reputation Staking | **Done** | `votes.stake` | — |
| F-13 | Slashing Protocol | **Done** | `ApplySlashing()` | — |
| F-14 | Proof of Evidence Appeals Protocol | **Partial** | `POST /appeals` | Stakes rank score, but there's no field or mechanism to actually submit new evidence — just an article ID and a stake percentage. |
| F-15 | Under Dispute Visual State | **Done** | `graph/handler.go` (`hasActiveAppeal`), `LineageGraph.tsx` (`.dispute-pulse`) | — |
| F-16 | Journalist Rank Score (R) | **Done** | `ranking.JournalistRankScore()` | — |
| F-17 | Logarithmic Volume Dampener | **Done** | `math.Log10` in the same formula | — |
| F-18 | Integrity-Incentivized Self-Correction | **Done** | `POST /claims/{id}/self-correct`, `w2 > w1` | — |
| F-19 | Instant Global Leaderboards | **Done** | `GET /leaderboard`, `/leaderboard` page | — |
| F-20 | Immutability Lock | **Missing** | `articles.immutability_locked` column exists, unused | No hard-delete endpoint exists at all (so nothing to lock against yet), and no logic ever sets this column based on the readership threshold. |
| F-21 | Retracted State Compliance | **Done** | `compliance/tombstone.go` | — |
| F-22 | Cryptographic Author Signatures | **Partial** | `lib/crypto.ts` signs client-side, `signature` column stores it | The backend accepts and stores whatever signature string it's given — it never verifies the signature against the journalist's public key. Right now this is closer to "signature-shaped field" than a verified crypto guarantee. |
| EX-01 | System Integration | **Mostly done** | `docker-compose.yml` wires Postgres/Kafka/Debezium/Neo4j/Redis/Go/Python | Frontend isn't containerized yet — runs via `pnpm dev` alongside the rest. |
| EX-02 | Testing and Acceptance Validation | **Partial** | `go test ./...` — `ranking`, `consensus`, `auth` | Real, checked-in unit tests now cover the three formulas (F-16/17/18), F-11 cross-tag consensus (including the "same tag ≠ cross-tag" edge case), and JWT issue/parse/tamper/expiry. Every other package (`articles`, `claims`, `compliance`, `graph`, `auditors`, `leaderboard`) still has zero automated coverage — those are mostly thin HTTP handlers over SQL, which need either a real test database or mocking to test meaningfully, neither of which exists yet. The frontend has no test runner configured at all. |
| EX-03 | Final Documentation and Deployment Package | **Partial** | `README.md`, `MIGRATION_NOTES.md` | Solid technical docs exist; presentation-ready screenshots and a packaged deployment bundle are still on you. |

## Suggested priority order for what's next

F-15 and F-19 are done as of this round. What's left, roughly
cheapest-to-most-involved:

1. **F-08 (Time-Based Clustering)** — needs a decision on bucketing rule
   (by month? by "recent" vs "archive"?) before it's an hour of work rather
   than a guess.
2. **F-14 evidence submission** — needs a schema decision (a text field? a
   file/URL? both?) before implementation.
3. **F-20 (Immutability Lock)** — there's no hard-delete endpoint to guard
   yet, so this is really "decide whether hard delete should exist at all,"
   which is a product question as much as a code one.
4. **F-22 real signature verification** — the most involved: needs a
   keypair-registration step at signup, storing the public key, and the
   backend actually verifying each submission against it.
5. **EX-02 (tests)** — valuable regardless of the above; a real Go test
   suite for `ranking`/`consensus` (the math has zero test coverage right
   now, despite being the most formula-heavy part of the system) would be
   the highest-leverage place to start.
