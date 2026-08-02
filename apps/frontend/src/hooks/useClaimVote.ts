"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

import { ApiError } from "@/lib/api";
import { castVote } from "@/lib/models/votes";

export function useClaimVote(claimId: string) {
  const router = useRouter();
  const [stake, setStake] = useState("1");
  const [verdict, setVerdict] = useState<"true" | "false">("true");
  const [status, setStatus] = useState<string | null>(null);
  const [tone, setTone] = useState<"ok" | "alert" | "neutral">("neutral");
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setStatus(null);
    setSubmitting(true);
    try {
      const res = await castVote(claimId, Number(stake), verdict === "true");
      if (res.resolved) {
        setTone(res.verdict ? "ok" : "alert");
        setStatus(`Consensus reached: claim marked ${res.verdict ? "verified" : "false"}.`);
      } else {
        setTone("neutral");
        setStatus("Vote recorded. It counts once an auditor with a non-overlapping tag also votes — you'll see it resolve then.");
      }
      setTimeout(() => router.push("/auditor/dashboard"), 1800);
    } catch (err) {
      setTone("alert");
      // Surface what the server actually said instead of guessing "already
      // resolved". The most common real cause is a 403 for an auditor whose
      // credentials an admin hasn't approved yet (NFR-6) — the old blanket
      // message hid exactly that, so a brand-new auditor thought voting was
      // broken rather than pending verification. Others: 422 insufficient
      // reputation to stake, 409 already resolved, 400 bad stake.
      setStatus(
        err instanceof ApiError && err.serverMessage
          ? err.serverMessage
          : "Could not record your vote — please try again.",
      );
    } finally {
      setSubmitting(false);
    }
  }

  return { stake, setStake, verdict, setVerdict, status, tone, submitting, handleSubmit };
}
