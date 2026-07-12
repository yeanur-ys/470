"use client";

import { useEffect, useState } from "react";
import Link from "next/link";

import { apiGet } from "@/lib/api";

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

  return (
    <div>
      <p>
        Claims awaiting cross-tag consensus (FR-7): a claim only resolves once auditors holding
        non-overlapping category tags agree on the same verdict.
      </p>
      {error && <p role="alert">{error}</p>}
      {!claims && !error && <p>Loading…</p>}
      {claims && claims.length === 0 && <p>Nothing pending right now.</p>}
      {claims && claims.length > 0 && (
        <ul>
          {claims.map((c) => (
            <li key={c.id} style={{ marginBottom: "0.75rem" }}>
              <Link href={`/auditor/claims/${c.id}`}>
                <strong>{c.tag}</strong>: {c.text}
              </Link>
              <br />
              <small>from "{c.articleTitle}"</small>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
