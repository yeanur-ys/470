"use client";

import { useEffect, useState } from "react";
import Link from "next/link";

import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { apiGet, apiPost, apiPostVoid } from "@/lib/api";
import { signArticle } from "@/lib/crypto";

interface Article {
  id: string;
  title: string;
}

interface CreateArticleResponse {
  id: string;
}

export default function PublishPage() {
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [parentArticleId, setParentArticleId] = useState("");
  const [myArticles, setMyArticles] = useState<Article[]>([]);
  const [publishedId, setPublishedId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const [claimText, setClaimText] = useState("");
  const [claimTag, setClaimTag] = useState("");
  const [claimStatus, setClaimStatus] = useState<string | null>(null);

  useEffect(() => {
    // Populated for Sequence Stitching (FR-4): pick a parent to chain this
    // story onto an existing one.
    apiGet<Article[]>("/articles/mine").then(setMyArticles).catch(() => {});
  }, []);

  async function handlePublish(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setSubmitting(true);
    try {
      const signature = await signArticle(title, body);
      const res = await apiPost<CreateArticleResponse>("/articles", {
        title,
        body,
        signature,
        parentArticleId: parentArticleId || undefined,
      });
      setPublishedId(res.id);
    } catch {
      setError("Could not publish. Check your connection and try again.");
    } finally {
      setSubmitting(false);
    }
  }

  async function handleAddClaim(e: React.FormEvent) {
    e.preventDefault();
    if (!publishedId) return;
    setClaimStatus(null);
    try {
      await apiPostVoid(`/articles/${publishedId}/claims`, { text: claimText, tag: claimTag });
      setClaimStatus(`Tagged: "${claimText}" (${claimTag})`);
      setClaimText("");
      setClaimTag("");
    } catch {
      setClaimStatus("Could not save that claim.");
    }
  }

  if (publishedId) {
    return (
      <div>
        <p role="status">Published. Now tag any #Claim statements auditors should verify (FR-3).</p>
        <form onSubmit={handleAddClaim} style={{ display: "flex", flexDirection: "column", gap: "0.5rem", maxWidth: 480 }}>
          <label>
            Claim text
            <Input value={claimText} onChange={(e) => setClaimText(e.target.value)} required />
          </label>
          <label>
            Category tag (e.g. "Economic Analyst")
            <Input value={claimTag} onChange={(e) => setClaimTag(e.target.value)} required />
          </label>
          <Button type="submit">Add claim</Button>
        </form>
        {claimStatus && <p>{claimStatus}</p>}
        <p>
          <Link href="/journalist/dashboard">Back to dashboard</Link>
        </p>
      </div>
    );
  }

  return (
    <form onSubmit={handlePublish} style={{ display: "flex", flexDirection: "column", gap: "0.75rem", maxWidth: 640 }}>
      <label>
        Title
        <Input value={title} onChange={(e) => setTitle(e.target.value)} required />
      </label>
      <label>
        Body
        <textarea
          value={body}
          onChange={(e) => setBody(e.target.value)}
          required
          rows={10}
          style={{ width: "100%" }}
        />
      </label>
      <label>
        Parent article (optional — Sequence Stitching)
        <select value={parentArticleId} onChange={(e) => setParentArticleId(e.target.value)}>
          <option value="">None</option>
          {myArticles.map((a) => (
            <option key={a.id} value={a.id}>
              {a.title}
            </option>
          ))}
        </select>
      </label>
      {error && <p role="alert">{error}</p>}
      <Button type="submit" disabled={submitting}>
        {submitting ? "Signing and publishing…" : "Publish"}
      </Button>
    </form>
  );
}
