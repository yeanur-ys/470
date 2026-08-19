"use client";

import { PageHeader } from "@/components/PageHeader";
import { useAuditorProfile } from "@/hooks/useAuditorProfile";

export default function AuditorProfilePage() {
  const { stats, error } = useAuditorProfile();

  return (
    <>
      <PageHeader eyebrow="Auditor desk" title={stats ? stats.displayName : "Your profile"} description="Your standing as an auditor on the platform." />

      {error && <p className="notice" data-tone="alert">{error}</p>}
      {!stats && !error && <p className="notice">Loading your profile…</p>}

      {stats && (
        <section className="card" style={{ padding: "1.25rem 1.5rem" }}>
          <div style={{ display: "flex", justifyContent: "space-between", alignItems: "baseline", gap: "1rem", flexWrap: "wrap" }}>
            <h2 style={{ margin: 0, fontSize: "1.05rem" }}>Standing</h2>
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
            <Stat
              label="Available to stake"
              value={stats.availableStake.toFixed(1)}
              hint={stats.lockedStake > 0 ? `${stats.lockedStake.toFixed(1)} locked in open votes` : undefined}
            />
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

          <p style={{ marginTop: "1rem", marginBottom: 0, fontSize: "0.85rem", color: "var(--ink-soft)" }}>
            Linked credential: <a href={stats.credentialUrl} target="_blank" rel="noreferrer">{stats.credentialUrl}</a>
          </p>
        </section>
      )}
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
