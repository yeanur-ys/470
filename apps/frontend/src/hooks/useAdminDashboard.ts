"use client";

import { useEffect, useState } from "react";

import { getAllArticles } from "@/lib/models/articles";
import type { Article } from "@/types/domain";

export function useAdminDashboard() {
  const [articles, setArticles] = useState<Article[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    getAllArticles()
      .then(setArticles)
      .catch(() => setError("Could not load articles."));
  }, []);

  const flagged = articles?.filter((a) => !a.isRetracted && a.falseClaims > 0) ?? [];
  const retracted = articles?.filter((a) => a.isRetracted) ?? [];

  const notes = [
    ...(flagged.length > 0
      ? [{ text: `${flagged.length} live stor${flagged.length === 1 ? "y carries" : "ies carry"} a false-claim verdict and may need retraction.`, tone: "pending" as const }]
      : []),
    { text: `${retracted.length} stor${retracted.length === 1 ? "y" : "ies"} already tombstoned.`, tone: "neutral" as const },
  ];

  return { articles, error, notes };
}
