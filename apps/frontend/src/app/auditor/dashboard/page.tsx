"use client";

import Link from "next/link";

import { PageHeader } from "@/components/PageHeader";
import { MarginLog } from "@/components/MarginLog";
import { useAuditorDashboard } from "@/hooks/useAuditorDashboard";

export default function AuditorDashboardPage() {
  const { claims, stats, error, notes } = useAuditorDashboard();

  return (
    <>
      <PageHeader
        eyebrow="Auditor desk"
        title={stats ? stats.displayName : "Auditor desk"}
        description="Your standing, and the claims awaiting a cross-tag verdict."
      />

      {/* Personal standing. Previously the auditor 'dashboard' was only the
          shared docket — nothing showed the auditor their own verification
          status, reputation, or vote record. */}
      {stats && (
        <>
          {!stats.credentialVerified && (
            <p className="notice" data-tone="alert" style={{ marginBottom: "1rem" }}>
              <strong>Your credentials are awaiting admin verification.</strong> You can review the
              docket below, but your votes won&apos;t be accepted until an admin approves the
              credential you linked at signup (NFR-6, Sybil resistance). This is why a vote is
              rejected right now.
            </p>
          )}

          <section className="card" style={{ padding: "1.25rem 1.5rem", marginBottom: "1.5rem" }}>
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "baseline", gap: "1rem", flexWrap: "wrap" }}>
              <h2 style={{ margin: 0, fontSize: "1.05rem" }}>Your standing</h2>
              <span className="stamp" data-tone={stats.credentialVerified ? "ok" : "pending"}>
                {stats.credentialVerified ? "Verified auditor" : "Pending verification"}
              </span>
            </div>

            <div
              style={{
                display: "grid",
                gridTemplateColumns: "repeat(auto-fit, minmax(130px, 1fr))",
                gap: "1rem",
                marginTop: "1rem",
              }}
            >
              <Stat label="Reputation" value={stats.rankScore.toFixed(1)} />
              <Stat label="Trust weight (Wₐ)" value={stats.trustWeight.toFixed(3)} />
              <Stat label="Available to stake" value={stats.availableStake.toFixed(1)} hint={stats.lockedStake > 0 ? `${stats.lockedStake.toFixed(1)} locked in open votes` : undefined} />
              <Stat label="Votes cast" value={String(stats.votesCast)} />
              <Stat label="Aligned / slashed" value={`${stats.successfulVotes} / ${stats.failedVotes}`} />
            </div>

            {stats.tags.length > 0 && (
              <p style={{ marginTop: "1rem", marginBottom: 0, fontSize: "0.85rem", color: "var(--ink-soft)" }}>
                Expertise: {stats.tags.map((t) => (
                  <span key={t} className="stamp" data-tone="neutral" style={{ marginRight: "0.35rem" }}>{t}</span>
                ))}
              </p>
            )}
          </section>
        </>
      )}

      <div className="docket">
        <div>
          <h2 style={{ fontSize: "1.05rem", marginTop: 0 }}>The docket</h2>
          <p style={{ color: "var(--ink-soft)", fontSize: "0.85rem", marginTop: "-0.25rem" }}>
            A claim resolves only once auditors holding non-overlapping category tags agree on the same verdict.
          </p>
          {error && <p className="notice" data-tone="alert">{error}</p>}
          {!claims && !error && <p className="notice">Loading the docket…</p>}
          {claims && claims.length === 0 && <p className="notice">Nothing pending right now.</p>}
          {claims && claims.length > 0 && (
            <div>
              {claims.map((c) => (
                <div className="card" key={c.id}>
                  <span className="eyebrow">{c.tag}</span>
                  <p style={{ fontFamily: "var(--font-display)", fontStyle: "italic", fontSize: "1.05rem" }}>
                    “{c.text}”
                  </p>
                  <p style={{ color: "var(--ink-soft)", fontSize: "0.85rem", marginBottom: "0.75rem" }}>
                    from “{c.articleTitle}”
                  </p>
                  <Link href={`/auditor/claims/${c.id}`} className="btn btn--ghost">
                    Review and vote →
                  </Link>
                </div>
              ))}
            </div>
          )}
        </div>
        <MarginLog heading="Docket breakdown" notes={notes} />
      </div>
    </>
  );
}

function Stat({ label, value, hint }: { label: string; value: string; hint?: string }) {
  return (
    <div>
      <div style={{ fontSize: "1.5rem", fontFamily: "var(--font-display)", lineHeight: 1.1 }}>{value}</div>
      <div className="eyebrow" style={{ margin: "0.25rem 0 0" }}>{label}</div>
      {hint && <div style={{ fontSize: "0.72rem", color: "var(--ink-soft)", marginTop: "0.15rem" }}>{hint}</div>}
    </div>
  );
}
