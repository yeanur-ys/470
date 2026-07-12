"use client";

import { useEffect, useState } from "react";
import Link from "next/link";

import { apiGet } from "@/lib/api";
import { PageHeader } from "@/components/PageHeader";
import { MarginLog } from "@/components/MarginLog";
import { Stamp } from "@/components/Stamp";

interface Article {
  id: string;
  title: string;
  journalistId: string;
  isRetracted: boolean;
  falseClaims: number;
  createdAt: string;
}

export default function AdminDashboardPage() {
  const [articles, setArticles] = useState<Article[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    apiGet<Article[]>("/articles")
      .then(setArticles)
      .catch(() => setError("Could not load articles."));
  }, []);

  const flagged = articles?.filter((a) => !a.isRetracted && a.falseClaims > 0) ?? [];
  const retracted = articles?.filter((a) => a.isRetracted) ?? [];

  const notes = [
    ...(flagged.length > 0
      ? [{ text: `${flagged.length} live stor${flagged.length === 1 ? "y carries" : "ies carry"} a false-claim verdict and may need retraction.`, tone: "pending" as const }]
      : []),
    { text: `${retracted.length} stor${retracted.length === 1 ? "y" : "ies"} already tombstoned.`, tone: "neutral" as const },
  ];

  return (
    <>
      <PageHeader
        eyebrow="Admin desk"
        title="Compliance ledger"
        description="Every story on the platform. Retraction is permanent and cannot be reversed."
      />
      <div className="docket">
        <div>
          <p style={{ marginBottom: "1.25rem" }}>
            <Link href="/admin/compliance" className="btn">
              Process a retraction
            </Link>
          </p>
          {error && <p className="notice" data-tone="alert">{error}</p>}
          {!articles && !error && <p className="notice">Loading the ledger…</p>}
          {articles && (
            <table className="ledger">
              <thead>
                <tr>
                  <th>Story</th>
                  <th>ID</th>
                  <th>Status</th>
                </tr>
              </thead>
              <tbody>
                {articles.map((a) => (
                  <tr key={a.id}>
                    <td>{a.title}</td>
                    <td className="mono" style={{ fontSize: "0.78rem", color: "var(--ink-soft)" }}>
                      {a.id}
                    </td>
                    <td>
                      {a.isRetracted ? (
                        <Stamp tone="alert">Retracted</Stamp>
                      ) : a.falseClaims > 0 ? (
                        <Stamp tone="pending">Flagged</Stamp>
                      ) : (
                        <Stamp tone="ok">Live</Stamp>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
        <MarginLog notes={notes} />
      </div>
    </>
  );
}
