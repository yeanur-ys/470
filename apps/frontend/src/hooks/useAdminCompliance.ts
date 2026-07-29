"use client";

import { useState } from "react";

import { retractArticle } from "@/lib/models/compliance";
import type { RetractionResult } from "@/types/domain";

export function useAdminCompliance() {
  const [articleId, setArticleId] = useState("");
  const [result, setResult] = useState<RetractionResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setResult(null);
    setSubmitting(true);
    try {
      const res = await retractArticle(articleId);
      setResult(res);
    } catch {
      setError("Retraction failed. Confirm the article ID is correct.");
    } finally {
      setSubmitting(false);
    }
  }

  return { articleId, setArticleId, result, error, submitting, handleSubmit };
}
