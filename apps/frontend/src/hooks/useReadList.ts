"use client";

import { useEffect, useState } from "react";

import { getAllArticles } from "@/lib/models/articles";
import type { Article } from "@/types/domain";

export function useReadList() {
  const [articles, setArticles] = useState<Article[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    getAllArticles()
      .then(setArticles)
      .catch(() => setError("Could not load stories."));
  }, []);

  return { articles, error };
}
