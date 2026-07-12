"use client";

import { useEffect, useState } from "react";
import Link from "next/link";

import { apiGet } from "@/lib/api";
import { PageHeader } from "@/components/PageHeader";
import { MarginLog } from "@/components/MarginLog";

interface PendingClaim {
  id: string;
  articleId: string;
  articleTitle: string;
  text: string;
  tag: string;
}

export default function AuditorDashboardPage() {
  const [claims, setClaims] = useState<PendingClaim[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    apiGet<PendingClaim[]>("/claims/pending")
      .then(setClaims)
      .catch(() => setError("Could not load pending claims."));
  }, []);

  const tagCounts = new Map<string, number>();
  claims?.forEach((c) => tagCounts.set(c.tag, (tagCounts.get(c.tag) ?? 0) + 1));

  const notes = [
    ...(claims && claims.length > 0
      ? [{ text: `${claims.length} claim${claims.length === 1 ? "" : "s"} waiting on a second, non-overlapping tag.`, tone: "pending" as const }]
      : []),
    ...Array.from(tagCounts.entries()).map(([tag, count]) => ({
      text: `${count} claim${count === 1 ? "" : "s"} tagged "${tag}".`,
      tone: "neutral" as const,
    })),
  ];

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
