"use client";

import { useEffect, useState } from "react";
import Link from "next/link";

import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { PageHeader } from "@/components/PageHeader";
import { MarginLog } from "@/components/MarginLog";
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
  const [claimsAdded, setClaimsAdded] = useState<string[]>([]);
  const [claimStatus, setClaimStatus] = useState<string | null>(null);

  useEffect(() => {
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
      setClaimsAdded((prev) => [...prev, `${claimText} — ${claimTag}`]);
      setClaimStatus(null);
      setClaimText("");
      setClaimTag("");
    } catch {
      setClaimStatus("Could not save that claim.");
    }
  }

  const draftNotes = [
    { text: title ? "Headline set." : "Headline still blank.", tone: title ? ("ok" as const) : ("pending" as const) },
    { text: body ? "Body drafted." : "Body still empty.", tone: body ? ("ok" as const) : ("pending" as const) },
    parentArticleId
      ? { text: "Chained to a parent story.", tone: "ok" as const }
      : { text: "Standalone — no parent story picked.", tone: "neutral" as const },
  ];

  if (publishedId) {
    const claimNotes =
      claimsAdded.length > 0
        ? claimsAdded.map((c) => ({ text: c, tone: "ok" as const }))
        : [{ text: "No claims tagged yet — auditors have nothing to verify on this story.", tone: "pending" as const }];

    return (
      <>
        <PageHeader
          eyebrow="Journalist desk"
          title="Tag the record"
          description={`Published. Now mark the specific #Claim statements auditors should verify.`}
        />
        <div className="docket">
          <div className="card app-main--narrow" style={{ padding: "1.5rem" }}>
            <form onSubmit={handleAddClaim}>
              <label className="field">
                Claim text
                <Input value={claimText} onChange={(e) => setClaimText(e.target.value)} required />
              </label>
              <label className="field">
                Category tag (e.g. "Economic Analyst")
                <Input value={claimTag} onChange={(e) => setClaimTag(e.target.value)} required />
              </label>
              <Button type="submit">Tag this claim</Button>
            </form>
            {claimStatus && <p className="notice" data-tone="alert" style={{ marginTop: "1rem" }}>{claimStatus}</p>}
            <p style={{ marginTop: "1.5rem" }}>
              <Link href="/journalist/dashboard">← Back to your byline</Link>
            </p>
          </div>
          <MarginLog heading="Tagged so far" notes={claimNotes} />
        </div>
      </>
    );
  }

  return (
    <>
      <PageHeader eyebrow="Journalist desk" title="File a story" description="Every word here is signed the moment you publish." />
      <div className="docket">
        <form onSubmit={handlePublish} className="card app-main--narrow" style={{ padding: "1.5rem" }}>
          <label className="field">
            Headline
            <Input value={title} onChange={(e) => setTitle(e.target.value)} required />
          </label>
          <label className="field">
            Body
            <textarea
              className="field-input"
              value={body}
              onChange={(e) => setBody(e.target.value)}
              required
              rows={10}
            />
          </label>
          <label className="field">
            Parent story (optional — Sequence Stitching)
            <select className="field-input" value={parentArticleId} onChange={(e) => setParentArticleId(e.target.value)}>
              <option value="">None</option>
              {myArticles.map((a) => (
                <option key={a.id} value={a.id}>
                  {a.title}
                </option>
              ))}
            </select>
          </label>
          {error && <p className="notice" data-tone="alert">{error}</p>}
          <Button type="submit" disabled={submitting} style={{ marginTop: "0.25rem" }}>
            {submitting ? "Signing and filing…" : "Publish"}
          </Button>
        </form>
        <MarginLog heading="Draft status" notes={draftNotes} />
      </div>
    </>
  );
}
