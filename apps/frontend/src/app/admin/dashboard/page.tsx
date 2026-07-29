"use client";

import Link from "next/link";

import { PageHeader } from "@/components/PageHeader";
import { MarginLog } from "@/components/MarginLog";
import { Stamp } from "@/components/Stamp";
import { useAdminDashboard } from "@/hooks/useAdminDashboard";

export default function AdminDashboardPage() {
  const { articles, error, notes } = useAdminDashboard();

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
