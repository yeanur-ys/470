# SRS Feature Status — F-01 through F-22, EX-01 through EX-03

Cross-referenced against `470_Project_Group5 (4).pdf` (the fixed-feature-ID SRS).
Status reflects what is **verified working end-to-end against the running
stack**, not what is scaffolded. "Partial" means something real exists but does
not fully satisfy the acceptance criterion in Section 7.

> **Revised July 2026.** The previous version of this file marked five features
> **Done** that did not meet their own Section 7 acceptance criteria: F-10,
> F-11, F-12, F-13 and (in substance) F-07. Details and root causes are in
> [`AUDIT_2026-07.md`](AUDIT_2026-07.md). Those five are now genuinely
> implemented and covered by tests.

| ID | Feature | Status | Where | Notes / remaining gap |
|---|---|---|---|---|
| F-01 | Individual Account Directed Graphs | **Done** | `GET /journalists/{id}/graph`, `LineageGraph.tsx` | — |
| F-02 | Transparent Article Lineage | **Done** | `SEQUENCE_OF` edges via CDC | Sync now reconciles the edge instead of only ever adding one, and MERGEs the parent so out-of-order CDC can't silently drop lineage. |
| F-03 | Dynamic Node Sizing | **Done** | `readershipToSize()` | log10-scaled, so one viral story can't swallow the layout. |
| F-04 | WebGL Epistemic Graphs | **Done** | Sigma.js + ForceAtlas2 (Barnes-Hut above 300 nodes) | Verified rendering 900 nodes / 9,858 edges. |
| F-05 | Semantic Zooming | **Done** | `useSemanticZoom.ts` | Now progressive — visible members per cluster grow with zoom, instead of a binary jump from ~8 nodes to all of them in one scroll step. |
| F-06 | Color Intensity Grading | **Done** | `corruptionToColor()` | Default fill encoding, per FR-10. |
| F-07 | Community-Detection Grouping | **Done** | `louvain.py`, `graph_store.py` | Previously **Partial in substance**: Louvain ran over `SEQUENCE_OF` only — a *forest*, where community detection is meaningless — and silently omitted every article with no parent. Now clusters over lineage + co-tag edges with isolated nodes included. 44 communities on the seeded corpus. |
| F-08 | Time-Based Clustering | **Done** | `articleEra()`, era legend | Buckets: Last 30 days / Last 12 months / Older. The SRS doesn't specify a rule; this is our choice. |
| F-09 | Claim Extraction Tool | **Done** | `POST /articles/{id}/claims` | Ownership is now enforced — previously any journalist could tag claims on anyone's article. |
| F-10 | Reputation-Weighted Consensus | **Done** | `EvaluateCrossTagConsensus()` | Was **failing**: no weighting of any kind — neither stake nor trust weight entered the decision. Now `sum(stake × (1 + trustWeight))` decides the verdict. |
| F-11 | Cross-Tag Validation Matrix | **Done** | `hasNonOverlappingPair()` | Was **partial**: only `users.tags[1]` was compared, so overlapping multi-tag auditors counted as a valid cross-tag pair. Now tests true set-disjointness. |
| F-12 | Auditor Reputation Staking | **Done** | `consensus/handler.go` | Was **failing**: `votes.stake` was written and never read — nothing was deducted or locked, so the same reputation backed unlimited simultaneous votes. Now locked against an available balance before the vote is accepted. |
| F-13 | Slashing Protocol | **Done** | `ApplySlashingTx()` | Was **failing**: it set a boolean and deducted nothing; `AuditorTrustWeight` (formula 3) was dead code. Misaligned voters now forfeit stake, aligned voters earn `0.5 ×` stake, and `V_s`/`V_f`/`W_a` are maintained. |
| F-14 | Proof of Evidence Appeals | **Done** | `POST /appeals` | Evidence (`evidenceText`/`evidenceUrl`) is now required and stored, the stake is actually deducted, and ownership is enforced. **Caveat:** no endpoint *resolves* an appeal — see below. |
| F-15 | Under Dispute Visual State | **Done** | `hasActiveAppeal`, `.dispute-pulse` | One active appeal per article, enforced by a partial unique index. |
| F-16 | Journalist Rank Score (R) | **Done** | `JournalistRankScore()` | — |
| F-17 | Logarithmic Volume Dampener | **Done** | `math.Log10` | — |
| F-18 | Integrity-Incentivized Self-Correction | **Done** | `w2 (1.5) > w1 (1.0)` | — |
| F-19 | Instant Global Leaderboards | **Done** | `GET /leaderboard` | Redis is now warmed from Postgres at boot. Previously a restarted Redis or a SQL-seeded DB served an empty leaderboard forever, with no code path able to reconcile it. |
| F-20 | Immutability Lock | **Done** | migration 0003 triggers | Was **Missing**. Auto-locks past 1,000 reads; DELETE of a locked article is refused at the storage layer, so the guarantee doesn't depend on every endpoint remembering to check. |
| F-21 | Retracted State Compliance | **Done** | `compliance/tombstone.go` | Now idempotent — retracting twice previously applied the FR-15 penalty twice, permanently. Penalty is reach-scaled rather than a flat `-2`. |
| F-22 | Cryptographic Author Signatures | **Partial** | `lib/crypto.ts`, `articles.signature` | **Unchanged and still the biggest gap.** The backend stores whatever signature string it is given and never verifies it. The frontend generates a throwaway keypair *per browser session*, so it isn't verifiable even in principle. Needs keypair registration at signup + a public key column + server-side verification. |
| EX-01 | System Integration | **Done** | `docker-compose.yml` | All 8 services verified running together; CDC verified Postgres → Kafka → Neo4j with counts matching exactly (900 articles, 676 lineage edges). Frontend still runs via `pnpm dev` rather than a container. |
| EX-02 | Testing and Acceptance Validation | **Partial** | `go test ./...` | `ranking`, `consensus`, `auth` pass, including new regression tests for consensus determinism and tag-disjointness. Handlers/SQL remain untested (needs a test database); frontend has no test runner. |
| EX-03 | Final Documentation | **Partial** | `README.md`, `AUDIT_2026-07.md`, `MIGRATION_NOTES.md` | Screenshots and a packaged deployment bundle are still outstanding. |

## Known gaps, in priority order

1. **F-22 signature verification** — the only core feature not substantively
   implemented. Needs a keypair registered at signup, a `users.public_key`
   column, and verification on every article write.
2. **Appeal resolution** — appeals can be filed, staked and displayed, but
   nothing resolves one, so staked score is never returned or forfeited. The
   SRS does not say who adjudicates an appeal or on what basis, so
   implementing it means inventing the rule. Needs a product decision first.
3. **NFR-5 (HTTPS/TLS)** — everything is plaintext HTTP.
4. **CI-3 (WebSocket/SSE)** — no realtime transport; the UI refreshes on
   navigation.
5. **Login rate limiting** — unlimited password attempts.
6. **Handler/SQL test coverage** — the formula layer is tested; the HTTP and
   persistence layers are not.
