"use client";

import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { PageHeader } from "@/components/PageHeader";
import { MarginLog } from "@/components/MarginLog";
import { useAdminCompliance } from "@/hooks/useAdminCompliance";

export default function CompliancePage() {
  const { articleId, setArticleId, result, error, submitting, handleSubmit } = useAdminCompliance();

  return (
    <>
      <PageHeader
        eyebrow="Admin desk"
        title="Retract a story"
        description="A GDPR/DMCA retraction replaces the content with a cryptographic tombstone, greys out the node, and permanently deducts the author's rank score."
      />
      <div className="docket">
        <form onSubmit={handleSubmit} className="card app-main--narrow" style={{ padding: "1.5rem" }}>
          <label className="field">
            Article ID
            <Input value={articleId} onChange={(e) => setArticleId(e.target.value)} required />
          </label>
          <Button type="submit" disabled={submitting}>
            {submitting ? "Retracting…" : "Retract"}
          </Button>
          {error && <p className="notice" data-tone="alert" style={{ marginTop: "1rem" }}>{error}</p>}
          {result && (
            <p className="notice" style={{ marginTop: "1rem" }}>
              Retracted. Tombstone hash: <code>{result.tombstoneHash}</code>
            </p>
          )}
        </form>
        <MarginLog
          notes={[
            { text: "This action cannot be undone once submitted.", tone: "alert" },
            { text: "The node stays visible, greyed-out, to preserve graph continuity.", tone: "neutral" },
          ]}
        />
      </div>
    </>
  );
}
