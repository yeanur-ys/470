"use client";

import Link from "next/link";

import { PublicHeader } from "@/components/PublicHeader";
import { PageHeader } from "@/components/PageHeader";
import { Stamp } from "@/components/Stamp";
import { useReadList } from "@/hooks/useReadList";

export default function ReadPage() {
  const { articles, error, page, setPage, totalPages } = useReadList();

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
        {articles && articles.length > 0 && (
          <div
            style={{
              display: "flex",
              justifyContent: "center",
              alignItems: "center",
              gap: "1rem",
              marginTop: "1.5rem",
            }}
          >
            <button
              type="button"
              className="btn btn--ghost"
              onClick={() => setPage((p) => p - 1)}
              disabled={page <= 1}
            >
              ← Previous
            </button>
            <span style={{ color: "var(--ink-soft)", fontSize: "0.85rem" }}>
              Page {page} of {totalPages}
            </span>
            <button
              type="button"
              className="btn btn--ghost"
              onClick={() => setPage((p) => p + 1)}
              disabled={page >= totalPages}
            >
              Next →
            </button>
          </div>
        )}
      </main>
    </>
  );
}
