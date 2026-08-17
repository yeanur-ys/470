"use client";

import Link from "next/link";

import { PageHeader } from "@/components/PageHeader";
import { getUserId } from "@/lib/auth";

// Admins don't have a "me" endpoint the way auditors do (no reputation/stats
// to show) — this is an identity card plus quick links into the sections an
// admin actually works in, not a data-heavy dashboard.
export default function AdminProfilePage() {
  const userId = getUserId();

  return (
    <>
      <PageHeader eyebrow="Admin desk" title="Your profile" description="Platform administrator account." />

      <section className="card" style={{ padding: "1.25rem 1.5rem", marginBottom: "1.5rem" }}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "baseline", gap: "1rem", flexWrap: "wrap" }}>
          <h2 style={{ margin: 0, fontSize: "1.05rem" }}>Identity</h2>
          <span className="stamp" data-tone="ok">Admin</span>
        </div>
        <p style={{ marginTop: "1rem", marginBottom: 0, fontSize: "0.85rem", color: "var(--ink-soft)" }}>
          User ID: <span className="mono">{userId}</span>
        </p>
      </section>

      <section className="card" style={{ padding: "1.25rem 1.5rem" }}>
        <h2 style={{ margin: 0, fontSize: "1.05rem", marginBottom: "0.75rem" }}>Go to</h2>
        <div style={{ display: "flex", flexWrap: "wrap", gap: "0.75rem" }}>
          <Link href="/admin/dashboard" className="btn btn--ghost">Dashboard</Link>
          <Link href="/admin/auditors" className="btn btn--ghost">Auditors</Link>
          <Link href="/admin/compliance" className="btn btn--ghost">Compliance</Link>
        </div>
      </section>
    </>
  );
}