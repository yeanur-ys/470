"use client";

import { useEffect, useState } from "react";

import { getAllArticles } from "@/lib/models/articles";
import type { Article } from "@/types/domain";

const PAGE_SIZE = 20;

export function useReadList() {
  const [articles, setArticles] = useState<Article[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);

  useEffect(() => {
    setArticles(null);
    getAllArticles({ page, pageSize: PAGE_SIZE })
      .then((res) => {
        setArticles(res.articles);
        setTotal(res.total);
      })
      .catch(() => setError("Could not load stories."));
  }, [page]);

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  return { articles, error, page, setPage, totalPages };
}
