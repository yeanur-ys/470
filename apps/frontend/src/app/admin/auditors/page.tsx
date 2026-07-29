"use client";

import { PageHeader } from "@/components/PageHeader";
import { MarginLog } from "@/components/MarginLog";
import { Button } from "@/components/ui/Button";
import { useAdminAuditors } from "@/hooks/useAdminAuditors";

export default function AdminAuditorsPage() {
  const { auditors, error, busyId, handleVerify, notes } = useAdminAuditors();

  return (
    <>
      <PageHeader
        eyebrow="Admin desk"
        title="Credential review"
        description="An auditor can sign in immediately, but can't cast a vote until their linked credentials are approved here (NFR-6)."
      />
      <div className="docket">
        <div>
          {error && <p className="notice" data-tone="alert">{error}</p>}
          {!auditors && !error && <p className="notice">Loading…</p>}
          {auditors && auditors.length === 0 && <p className="notice">Nothing waiting on review.</p>}
          {auditors?.map((a) => (
            <div className="card" key={a.id}>
              <span className="eyebrow">{a.tags.join(" · ") || "No tags provided"}</span>
              <p style={{ fontFamily: "var(--font-display)", fontSize: "1.05rem", marginBottom: "0.25rem" }}>
                {a.displayName}
              </p>
              <p style={{ color: "var(--ink-soft)", fontSize: "0.85rem" }}>{a.email}</p>
              <p style={{ marginBottom: "0.75rem" }}>
                <a href={a.credentialUrl} target="_blank" rel="noreferrer">
                  {a.credentialUrl}
                </a>
              </p>
              <Button onClick={() => handleVerify(a.id)} disabled={busyId === a.id}>
                {busyId === a.id ? "Verifying…" : "Approve credentials"}
              </Button>
            </div>
          ))}
        </div>
        <MarginLog notes={notes} />
      </div>
    </>
  );
}
