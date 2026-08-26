"use client";

import { useEffect, useState } from "react";

import { ApiError } from "@/lib/api";
import { createAppeal } from "@/lib/models/appeals";
import { getMyArticles } from "@/lib/models/articles";
import type { Article } from "@/types/domain";

export function useAppeals() {
  const [articles, setArticles] = useState<Article[]>([]);
  const [articleId, setArticleId] = useState("");
  const [stakedPercent, setStakedPercent] = useState("10");
  const [evidenceText, setEvidenceText] = useState("");
  const [evidenceUrl, setEvidenceUrl] = useState("");
  const [status, setStatus] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    getMyArticles().then(setArticles).catch(() => {});
  }, []);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setStatus(null);

    // F-14 is "Proof of Evidence" — the backend rejects an appeal with
    // neither field set, so check client-side rather than making the
    // journalist wait on a round trip to find that out.
    if (!evidenceText.trim() && !evidenceUrl.trim()) {
      setStatus("An appeal needs new evidence — add a short explanation or a link to it.");
      return;
    }

    setSubmitting(true);
    try {
      await createAppeal({
        articleId,
        stakedPercent: Number(stakedPercent),
        evidenceText: evidenceText.trim() || undefined,
        evidenceUrl: evidenceUrl.trim() || undefined,
      });
      setStatus("Appeal filed. The disputed node will show as pending review.");
      setEvidenceText("");
      setEvidenceUrl("");
    } catch (err) {
      // Surface what the server actually said — e.g. "you already have an
      // active appeal on this article" — instead of a blanket message that
      // gives no clue why a second attempt failed (appeals can only ever
      // have one active entry per article; there is currently no way for
      // that to be resolved and re-opened).
      setStatus(
        err instanceof ApiError && err.serverMessage
          ? err.serverMessage
          : "Could not file the appeal.",
      );
    } finally {
      setSubmitting(false);
    }
  }

  const disputable = articles.filter((a) => !a.isRetracted && a.falseClaims > 0);
  const appealable = disputable.filter((a) => !a.hasActiveAppeal);
  const notes = [
    appealable.length > 0
      ? { text: `${appealable.length} stor${appealable.length === 1 ? "y is" : "ies are"} eligible for a new appeal right now.`, tone: "pending" as const }
      : disputable.length > 0
        ? { text: "Every disputed story already has an active appeal awaiting review.", tone: "neutral" as const }
        : { text: "No stories currently carry a false-claim verdict.", tone: "ok" as const },
    { text: "Staking is irreversible — it's deducted whether the appeal succeeds or not.", tone: "neutral" as const },
    { text: "At least one of evidence text or a link is required — an appeal with no new evidence is just a complaint.", tone: "neutral" as const },
    { text: "Only one active appeal is allowed per story at a time.", tone: "neutral" as const },
  ];

  return {
    articles,
    disputable,
    articleId,
    setArticleId,
    stakedPercent,
    setStakedPercent,
    evidenceText,
    setEvidenceText,
    evidenceUrl,
    setEvidenceUrl,
    status,
    submitting,
    handleSubmit,
    notes,
  };
}
