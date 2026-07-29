"use client";

import { useEffect, useState } from "react";

import { createAppeal } from "@/lib/models/appeals";
import { getMyArticles } from "@/lib/models/articles";
import type { Article } from "@/types/domain";

export function useAppeals() {
  const [articles, setArticles] = useState<Article[]>([]);
  const [articleId, setArticleId] = useState("");
  const [stakedPercent, setStakedPercent] = useState("10");
  const [status, setStatus] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    getMyArticles().then(setArticles).catch(() => {});
  }, []);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setStatus(null);
    setSubmitting(true);
    try {
      await createAppeal(articleId, Number(stakedPercent));
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

  return { articles, articleId, setArticleId, stakedPercent, setStakedPercent, status, submitting, handleSubmit, notes };
}
