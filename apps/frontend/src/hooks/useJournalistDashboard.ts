"use client";

import { useEffect, useState } from "react";

import { getMyArticles } from "@/lib/models/articles";
import { selfCorrectClaim } from "@/lib/models/claims";
import type { Article } from "@/types/domain";

interface MarginNote {
  text: string;
  tone?: "ok" | "alert" | "pending" | "neutral";
}

// buildNotes is a business rule (what's worth flagging about a journalist's
// byline), not presentation — it belongs here, not in the page component.
function buildNotes(articles: Article[] | null): MarginNote[] {
  if (!articles) return [];
  const notes: MarginNote[] = [];

  const retracted = articles.filter((a) => a.isRetracted);
  if (retracted.length > 0) {
    notes.push({
      text: `${retracted.length} stor${retracted.length === 1 ? "y" : "ies"} tombstoned by compliance.`,
      tone: "alert",
    });
  }

  const disputed = articles.filter((a) => !a.isRetracted && a.falseClaims > 0);
  if (disputed.length > 0) {
    notes.push({
      text: `${disputed.length} stor${disputed.length === 1 ? "y carries" : "ies carry"} at least one false claim — consider an appeal.`,
      tone: "pending",
    });
  }

  const untagged = articles.filter((a) => a.verifiedClaims + a.selfCorrectedClaims + a.falseClaims === 0);
  if (untagged.length > 0) {
    notes.push({
      text: `${untagged.length} stor${untagged.length === 1 ? "y has" : "ies have"} no tagged claims yet — nothing for an auditor to verify.`,
      tone: "neutral",
    });
  }

  if (notes.length === 0 && articles.length > 0) {
    notes.push({ text: "Clean ledger. Every story stands unchallenged.", tone: "ok" });
  }

  return notes;
}

export function useJournalistDashboard() {
  const [articles, setArticles] = useState<Article[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  const [claimId, setClaimId] = useState("");
  const [selfCorrectStatus, setSelfCorrectStatus] = useState<string | null>(null);
  const [selfCorrectTone, setSelfCorrectTone] = useState<"ok" | "alert">("ok");
  const [submittingSelfCorrect, setSubmittingSelfCorrect] = useState(false);

  useEffect(() => {
    getMyArticles()
      .then(setArticles)
      .catch(() => setError("Could not load your articles."));
  }, []);

  async function handleSelfCorrect(e: React.FormEvent) {
    e.preventDefault();
    setSelfCorrectStatus(null);
    setSubmittingSelfCorrect(true);
    try {
      await selfCorrectClaim(claimId);
      setSelfCorrectTone("ok");
      setSelfCorrectStatus("Marked self-corrected — this counts toward your rank score.");
      setClaimId("");
    } catch {
      setSelfCorrectTone("alert");
      setSelfCorrectStatus("Couldn't self-correct that claim — it may not be yours, or already resolved.");
    } finally {
      setSubmittingSelfCorrect(false);
    }
  }

  return {
    articles,
    error,
    notes: buildNotes(articles),
    claimId,
    setClaimId,
    selfCorrectStatus,
    selfCorrectTone,
    submittingSelfCorrect,
    handleSelfCorrect,
  };
}
