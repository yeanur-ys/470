"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";

import { apiGet, apiPostVoid } from "@/lib/api";
import { PublicHeader } from "@/components/PublicHeader";
import { Stamp } from "@/components/Stamp";

interface ArticleClaim {
  id: string;
  text: string;
  tag: string;
  status: "pending" | "verified" | "self_corrected" | "false";
}

interface ArticleDetail {
  id: string;
  journalistId: string;
  title: string;
  body: string;
  readershipVolume: number;
  isRetracted: boolean;
  createdAt: string;
  claims: ArticleClaim[];
}

const CLAIM_STAMP: Record<ArticleClaim["status"], { tone: "ok" | "alert" | "pending" | "neutral"; label: string }> = {
  pending: { tone: "neutral", label: "Awaiting auditors" },
  verified: { tone: "ok", label: "Verified" },
  self_corrected: { tone: "pending", label: "Self-corrected" },
  false: { tone: "alert", label: "False" },
};

export default function ReadArticlePage() {
  const params = useParams<{ articleId: string }>();
  const [article, setArticle] = useState<ArticleDetail | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    apiGet<ArticleDetail>(`/articles/${params.articleId}`)
      .then((data) => {
        setArticle(data);
        // Fire-and-forget: counts toward readership volume (FR-12) and the
        // journalist's rank score input, but a failed increment shouldn't
        // block the reader from seeing the story.
        apiPostVoid(`/articles/${params.articleId}/read`, {}).catch(() => {});
      })
      .catch(() => setError("Could not load this story."));
  }, [params.articleId]);

  return (
    <>
      <PublicHeader />
      <main style={{ maxWidth: 680, margin: "0 auto", padding: "2rem 2rem 4rem" }}>
        <p style={{ marginBottom: "1.5rem" }}>
          <Link href="/read">← All stories</Link>
        </p>

        {error && <p className="notice" data-tone="alert">{error}</p>}
        {!article && !error && <p className="notice">Loading…</p>}

        {article && (
          <article>
            {article.isRetracted && (
              <p className="notice" data-tone="alert" style={{ marginBottom: "1.5rem" }}>
                This story was retracted following a valid legal request. It stays listed,
                greyed out, so the record isn't erased — see its{" "}
                <Link href={`/profile/${article.journalistId}`}>lineage graph</Link>.
              </p>
            )}

            <h1>{article.title}</h1>
            <p className="eyebrow">
              {article.readershipVolume} reads ·{" "}
              <Link href={`/profile/${article.journalistId}`}>view journalist's record ↗</Link>
            </p>

            <div style={{ marginTop: "1.5rem", whiteSpace: "pre-wrap", lineHeight: 1.7 }}>{article.body}</div>

            <h2 style={{ marginTop: "2.5rem" }}>Tagged claims</h2>
            {article.claims.length === 0 && (
              <p className="notice">No claims were tagged in this story.</p>
            )}
            {article.claims.map((c) => {
              const stamp = CLAIM_STAMP[c.status];
              return (
                <div className="card" key={c.id}>
                  <div style={{ display: "flex", justifyContent: "space-between", alignItems: "baseline", gap: "1rem" }}>
                    <p style={{ fontFamily: "var(--font-display)", fontStyle: "italic", margin: 0 }}>
                      “{c.text}”
                    </p>
                    <Stamp tone={stamp.tone}>{stamp.label}</Stamp>
                  </div>
                  <p style={{ color: "var(--ink-soft)", fontSize: "0.8rem", margin: "0.4rem 0 0" }}>{c.tag}</p>
                </div>
              );
            })}
          </article>
        )}
      </main>
    </>
  );
}
