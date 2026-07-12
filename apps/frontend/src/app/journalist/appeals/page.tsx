"use client";

import { useEffect, useState } from "react";

import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { apiGet, apiPostVoid } from "@/lib/api";

interface Article {
  id: string;
  title: string;
  isRetracted: boolean;
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
      setStatus("Appeal filed. The disputed node will show as pending review (FR-9).");
    } catch {
      setStatus("Could not file the appeal.");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div>
      <p>
        Dispute a ruling by staking a percentage of your own rank score (FR-5). This is
        irreversible if the appeal is rejected — the staked percentage is deducted either way.
      </p>
      <form onSubmit={handleSubmit} style={{ display: "flex", flexDirection: "column", gap: "0.75rem", maxWidth: 480 }}>
        <label>
          Article
          <select value={articleId} onChange={(e) => setArticleId(e.target.value)} required>
            <option value="" disabled>
              Select an article…
            </option>
            {articles.map((a) => (
              <option key={a.id} value={a.id}>
                {a.title} {a.isRetracted ? "(retracted)" : ""}
              </option>
            ))}
          </select>
        </label>
        <label>
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
      </form>
      {status && <p role="status">{status}</p>}
    </div>
  );
}
