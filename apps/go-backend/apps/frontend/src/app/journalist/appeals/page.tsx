"use client";

import { useEffect, useState } from "react";

import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { PageHeader } from "@/components/PageHeader";
import { MarginLog } from "@/components/MarginLog";
import { apiGet, apiPostVoid } from "@/lib/api";

interface Article {
  id: string;
  title: string;
  isRetracted: boolean;
  falseClaims: number;
}

export default function AppealsPage() {
  const [articles, setArticles] = useState<Article[]>([]);
  const [articleId, setArticleId] = useState("");
  const [stakedPercent, setStakedPercent] = useState("10");
  const [status, setStatus] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    apiGet<Article[]>("/articles/mine").then(setArticles).catch(() => {});
  }, []);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setStatus(null);
    setSubmitting(true);
    try {
      await apiPostVoid("/appeals", { articleId, stakedPercent: Number(stakedPercent) });
      setStatus("Appeal filed. The disputed node will show as pending review.");
    } catch {
      setStatus("Could not file the appeal.");
    } finally {
      setSubmitting(false);
    }
  }

  const disputable = articles.filter((a) => !a.isRetracted && a.falseClaims > 0);
  const notes = [
    disputable.length > 0
      ? { text: `${disputable.length} stor${disputable.length === 1 ? "y is" : "ies are"} eligible for appeal right now.`, tone: "pending" as const }
      : { text: "No stories currently carry a false-claim verdict.", tone: "ok" as const },
    { text: "Staking is irreversible — it's deducted whether the appeal succeeds or not.", tone: "neutral" as const },
  ];

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
