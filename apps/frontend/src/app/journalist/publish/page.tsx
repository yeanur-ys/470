"use client";

import Link from "next/link";

import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { PageHeader } from "@/components/PageHeader";
import { MarginLog } from "@/components/MarginLog";
import { usePublish } from "@/hooks/usePublish";

export default function PublishPage() {
  const {
    title, setTitle,
    body, setBody,
    parentArticleId, setParentArticleId,
    myArticles,
    publishedId,
    error,
    submitting,
    handlePublish,
    draftNotes,
    claimText, setClaimText,
    claimTag, setClaimTag,
    claimsAdded,
    claimStatus,
    claimNotes,
    handleAddClaim,
  } = usePublish();

  if (publishedId) {
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
            {claimsAdded.length > 0 && (
              <table className="ledger" style={{ marginTop: "1.25rem" }}>
                <thead>
                  <tr>
                    <th>Claim</th>
                    <th>ID (for self-correction later)</th>
                  </tr>
                </thead>
                <tbody>
                  {claimsAdded.map((c) => (
                    <tr key={c.id}>
                      <td>
                        {c.text} <span style={{ color: "var(--ink-soft)" }}>({c.tag})</span>
                      </td>
                      <td className="mono" style={{ fontSize: "0.78rem" }}>
                        {c.id}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
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
