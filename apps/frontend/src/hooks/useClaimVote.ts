"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

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
        setStatus("Vote recorded — waiting on a cross-tag auditor to confirm.");
      }
      setTimeout(() => router.push("/auditor/dashboard"), 1600);
    } catch {
      setTone("alert");
      setStatus("Could not record your vote. It may already be resolved.");
    } finally {
      setSubmitting(false);
    }
  }

  return { stake, setStake, verdict, setVerdict, status, tone, submitting, handleSubmit };
}
