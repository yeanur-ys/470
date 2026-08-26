"use client";

import { useCallback, useEffect, useMemo, useState } from "react";

import { getMyArticles } from "@/lib/models/articles";
import { getMyClaims, selfCorrectClaim, tagClaim } from "@/lib/models/claims";
import type { Article, JournalistClaim } from "@/types/domain";

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
  const [claims, setClaims] = useState<JournalistClaim[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Which story's claims panel is open — only one at a time, so claims stay
  // scoped per story instead of piling up in one long flat list.
  const [expandedArticleId, setExpandedArticleId] = useState<string | null>(null);

  // Editing an existing (still-pending) claim: a "correction" means fixing
  // what the claim actually says, not just relabelling it, so this doubles as
  // the self-correct action.
  const [editingClaimId, setEditingClaimId] = useState<string | null>(null);
  const [editText, setEditText] = useState("");
  const [editTag, setEditTag] = useState("");
  const [editBusy, setEditBusy] = useState(false);
  const [editStatus, setEditStatus] = useState<string | null>(null);
  const [editTone, setEditTone] = useState<"ok" | "alert">("ok");

  // Tagging a brand-new claim on a story that's already been published — not
  // just in the one-shot window right after publishing it.
  const [newClaimText, setNewClaimText] = useState("");
  const [newClaimTag, setNewClaimTag] = useState("");
  const [addBusy, setAddBusy] = useState(false);
  const [addStatus, setAddStatus] = useState<string | null>(null);

  const loadClaims = useCallback(() => {
    getMyClaims()
      .then(setClaims)
      .catch(() => setError("Could not load your tagged claims."));
  }, []);

  const loadArticles = useCallback(() => {
    getMyArticles()
      .then(setArticles)
      .catch(() => setError("Could not load your articles."));
  }, []);

  useEffect(() => {
    loadArticles();
    loadClaims();
  }, [loadArticles, loadClaims]);

  const claimsByArticle = useMemo(() => {
    const map = new Map<string, JournalistClaim[]>();
    for (const c of claims ?? []) {
      const bucket = map.get(c.articleId);
      if (bucket) bucket.push(c);
      else map.set(c.articleId, [c]);
    }
    return map;
  }, [claims]);

  function toggleStory(articleId: string) {
    const opening = expandedArticleId !== articleId;
    setExpandedArticleId(opening ? articleId : null);
    // Leaving a claim's edit form open across a different story (or a closed
    // panel) reads as a bug the next time it's reopened — reset both forms
    // whenever which story is open changes.
    setEditingClaimId(null);
    setEditStatus(null);
    setNewClaimText("");
    setNewClaimTag("");
    setAddStatus(null);
  }

  function startEdit(claim: JournalistClaim) {
    setEditingClaimId(claim.id);
    setEditText(claim.text);
    setEditTag(claim.tag);
    setEditStatus(null);
  }

  function cancelEdit() {
    setEditingClaimId(null);
    setEditStatus(null);
  }

  // Rewrites the claim's text/tag and marks it self-corrected in one action —
  // the claim stays right where it was, just updated, rather than
  // disappearing from the list.
  async function submitEdit(e: React.FormEvent) {
    e.preventDefault();
    if (!editingClaimId) return;
    setEditBusy(true);
    setEditStatus(null);
    try {
      await selfCorrectClaim(editingClaimId, { text: editText, tag: editTag });
      setEditTone("ok");
      setEditStatus("Saved — marked self-corrected, which counts toward your rank score.");
      setEditingClaimId(null);
      loadClaims();
      loadArticles();
    } catch {
      setEditTone("alert");
      setEditStatus("Couldn't save that change — the claim may already be resolved.");
    } finally {
      setEditBusy(false);
    }
  }

  async function submitNewClaim(e: React.FormEvent, articleId: string) {
    e.preventDefault();
    setAddBusy(true);
    setAddStatus(null);
    try {
      await tagClaim(articleId, newClaimText, newClaimTag);
      setNewClaimText("");
      setNewClaimTag("");
      loadClaims();
    } catch {
      setAddStatus("Couldn't tag that claim — check the fields and try again.");
    } finally {
      setAddBusy(false);
    }
  }

  return {
    articles,
    claims,
    claimsByArticle,
    error,
    notes: buildNotes(articles),
    expandedArticleId,
    toggleStory,
    editingClaimId,
    editText,
    setEditText,
    editTag,
    setEditTag,
    editBusy,
    editStatus,
    editTone,
    startEdit,
    cancelEdit,
    submitEdit,
    newClaimText,
    setNewClaimText,
    newClaimTag,
    setNewClaimTag,
    addBusy,
    addStatus,
    submitNewClaim,
  };
}
