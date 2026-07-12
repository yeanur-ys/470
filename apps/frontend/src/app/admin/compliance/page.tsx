"use client";

import { useState } from "react";

import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { apiPost } from "@/lib/api";

interface RetractResponse {
  status: string;
  tombstoneHash: string;
}

export default function CompliancePage() {
  const [articleId, setArticleId] = useState("");
  const [result, setResult] = useState<RetractResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setResult(null);
    setSubmitting(true);
    try {
      const res = await apiPost<RetractResponse>(`/admin/articles/${articleId}/retract`, {});
      setResult(res);
    } catch {
      setError("Retraction failed. Confirm the article ID is correct.");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div>
      <p>
        Apply a GDPR/DMCA retraction (FR-14): identifying content is replaced with a cryptographic
        tombstone, the node stays visible (greyed-out) for historical continuity, and the
        author's rank score takes a permanent deduction (FR-15). This cannot be undone.
      </p>
      <form onSubmit={handleSubmit} style={{ display: "flex", gap: "0.5rem", maxWidth: 480 }}>
        <Input
          placeholder="Article ID"
          value={articleId}
          onChange={(e) => setArticleId(e.target.value)}
          required
          style={{ flex: 1 }}
        />
        <Button type="submit" disabled={submitting}>
          {submitting ? "Retracting…" : "Retract"}
        </Button>
      </form>
      {error && <p role="alert">{error}</p>}
      {result && (
        <p role="status">
          Retracted. Tombstone hash: <code>{result.tombstoneHash}</code>
        </p>
      )}
    </div>
  );
}
