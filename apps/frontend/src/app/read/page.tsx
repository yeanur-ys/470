"use client";

import { useEffect, useState } from "react";
import Link from "next/link";

import { apiGet } from "@/lib/api";
import { PublicHeader } from "@/components/PublicHeader";
import { PageHeader } from "@/components/PageHeader";
import { Stamp } from "@/components/Stamp";

interface Article {
  id: string;
  title: string;
  readershipVolume: number;
  falseClaims: number;
  isRetracted: boolean;
  createdAt: string;
}

export default function ReadPage() {
  const [articles, setArticles] = useState<Article[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    apiGet<Article[]>("/articles")
      .then(setArticles)
      .catch(() => setError("Could not load stories."));
  }, []);

  return (
    <>
      <PublicHeader />
      <main style={{ maxWidth: 760, margin: "0 auto", padding: "2rem 2rem 4rem" }}>
        <PageHeader
          eyebrow="Open to everyone"
          title="Every story, in order"
          description="No account needed to read. Each story shows the verdict on every claim tagged inside it — nothing here is decided by an editor."
        />
        {error && <p className="notice" data-tone="alert">{error}</p>}
        {!articles && !error && <p className="notice">Loading stories…</p>}
        {articles && articles.length === 0 && <p className="notice">Nothing published yet.</p>}
        {articles?.map((a) => (
          <Link
            key={a.id}
            href={`/read/${a.id}`}
            style={{ textDecoration: "none", color: "inherit", display: "block" }}
          >
            <div className="card">
              <div style={{ display: "flex", justifyContent: "space-between", alignItems: "baseline" }}>
                <h3 style={{ margin: 0 }}>{a.isRetracted ? "[This story was retracted]" : a.title}</h3>
                {a.isRetracted ? (
                  <Stamp tone="alert">Retracted</Stamp>
                ) : a.falseClaims > 0 ? (
                  <Stamp tone="pending">Disputed claim</Stamp>
                ) : (
                  <Stamp tone="ok">Clean record</Stamp>
                )}
              </div>
              <p style={{ color: "var(--ink-soft)", fontSize: "0.85rem", margin: "0.4rem 0 0" }}>
                {a.readershipVolume} reads
              </p>
            </div>
          </Link>
        ))}
      </main>
    </>
  );
}
