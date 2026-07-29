"use client";

import { useEffect, useState } from "react";

import { getArticleDetail, recordArticleRead } from "@/lib/models/articles";
import type { ArticleDetail } from "@/types/domain";

export function useReadArticle(articleId: string) {
  const [article, setArticle] = useState<ArticleDetail | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    getArticleDetail(articleId)
      .then((data) => {
        setArticle(data);
        // Fire-and-forget: see recordArticleRead's own comment — a failed
        // increment shouldn't block the reader from seeing the story.
        recordArticleRead(articleId).catch(() => {});
      })
      .catch(() => setError("Could not load this story."));
  }, [articleId]);

  return { article, error };
}
