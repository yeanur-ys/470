"use client";

import Link from "next/link";

import { PageHeader } from "@/components/PageHeader";
import { MarginLog } from "@/components/MarginLog";
import { useAuditorDashboard } from "@/hooks/useAuditorDashboard";

export default function AuditorDashboardPage() {
  const { claims, error, notes } = useAuditorDashboard();

  return (
    <>
      <PageHeader
        eyebrow="Auditor desk"
        title="The docket"
        description="A claim resolves only once auditors holding non-overlapping category tags agree on the same verdict."
      />
      <div className="docket">
        <div>
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
