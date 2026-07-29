"use client";

import { useEffect, useState } from "react";

import { signArticle } from "@/lib/crypto";
import { createArticle, getMyArticles } from "@/lib/models/articles";
import { tagClaim } from "@/lib/models/claims";
import type { Article } from "@/types/domain";

interface TaggedClaim {
  id: string;
  text: string;
  tag: string;
}

export function usePublish() {
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [parentArticleId, setParentArticleId] = useState("");
  const [myArticles, setMyArticles] = useState<Article[]>([]);
  const [publishedId, setPublishedId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const [claimText, setClaimText] = useState("");
  const [claimTag, setClaimTag] = useState("");
  const [claimsAdded, setClaimsAdded] = useState<TaggedClaim[]>([]);
  const [claimStatus, setClaimStatus] = useState<string | null>(null);

  useEffect(() => {
    getMyArticles().then(setMyArticles).catch(() => {});
  }, []);

  async function handlePublish(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setSubmitting(true);
    try {
      const signature = await signArticle(title, body);
      const res = await createArticle({ title, body, signature, parentArticleId: parentArticleId || undefined });
      setPublishedId(res.id);
    } catch {
      setError("Could not publish. Check your connection and try again.");
    } finally {
      setSubmitting(false);
    }
  }

  async function handleAddClaim(e: React.FormEvent) {
    e.preventDefault();
    if (!publishedId) return;
    setClaimStatus(null);
    try {
      const res = await tagClaim(publishedId, claimText, claimTag);
      setClaimsAdded((prev) => [...prev, { id: res.id, text: claimText, tag: claimTag }]);
      setClaimText("");
      setClaimTag("");
    } catch {
      setClaimStatus("Could not save that claim.");
    }
  }

  const draftNotes = [
    { text: title ? "Headline set." : "Headline still blank.", tone: title ? ("ok" as const) : ("pending" as const) },
    { text: body ? "Body drafted." : "Body still empty.", tone: body ? ("ok" as const) : ("pending" as const) },
    parentArticleId
      ? { text: "Chained to a parent story.", tone: "ok" as const }
      : { text: "Standalone — no parent story picked.", tone: "neutral" as const },
  ];

  const claimNotes =
    claimsAdded.length > 0
      ? claimsAdded.map((c) => ({ text: `${c.text} — ${c.tag}`, tone: "ok" as const }))
      : [{ text: "No claims tagged yet — auditors have nothing to verify on this story.", tone: "pending" as const }];

  return {
    title, setTitle,
    body, setBody,
    parentArticleId, setParentArticleId,
    myArticles,
    publishedId,
    error,
    submitting,
    handlePublish,
    draftNotes,
    claimText, setClaimText,
    claimTag, setClaimTag,
    claimsAdded,
    claimStatus,
    claimNotes,
    handleAddClaim,
  };
}
