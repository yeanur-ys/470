"use client";

import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { PageHeader } from "@/components/PageHeader";
import { MarginLog } from "@/components/MarginLog";
import { useAppeals } from "@/hooks/useAppeals";

export default function AppealsPage() {
  const { articles, articleId, setArticleId, stakedPercent, setStakedPercent, status, submitting, handleSubmit, notes } =
    useAppeals();

  return (
    <>
      <PageHeader
        eyebrow="Journalist desk"
        title="Dispute a ruling"
        description="Stake a percentage of your rank score to challenge a false-claim verdict."
      />
      <div className="docket">
        <form onSubmit={handleSubmit} className="card app-main--narrow" style={{ padding: "1.5rem" }}>
          <label className="field">
            Story
            <select className="field-input" value={articleId} onChange={(e) => setArticleId(e.target.value)} required>
              <option value="" disabled>
                Select a story…
              </option>
              {articles.map((a) => (
                <option key={a.id} value={a.id}>
                  {a.title} {a.isRetracted ? "(retracted)" : ""}
                </option>
              ))}
            </select>
          </label>
          <label className="field">
            Rank score to stake (%)
            <Input
              type="number"
              min={1}
              max={100}
              value={stakedPercent}
              onChange={(e) => setStakedPercent(e.target.value)}
              required
            />
          </label>
          <Button type="submit" disabled={submitting || !articleId}>
            {submitting ? "Filing…" : "File appeal"}
          </Button>
          {status && <p className="notice" style={{ marginTop: "1rem" }}>{status}</p>}
        </form>
        <MarginLog notes={notes} />
      </div>
    </>
  );
}
