"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { apiPost } from "@/lib/api";

interface VoteResponse {
  resolved?: boolean;
  verdict?: boolean;
}

export function ClaimVoteForm({ claimId }: { claimId: string }) {
  const router = useRouter();
  const [stake, setStake] = useState("1");
  const [verdict, setVerdict] = useState<"true" | "false">("true");
  const [status, setStatus] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setStatus(null);
    setSubmitting(true);
    try {
      const res = await apiPost<VoteResponse>(`/claims/${claimId}/votes`, {
        stake: Number(stake),
        verdict: verdict === "true",
      });
      if (res.resolved) {
        setStatus(`Consensus reached: claim marked ${res.verdict ? "verified" : "false"}.`);
      } else {
        setStatus("Vote recorded — waiting on a cross-tag auditor to confirm (FR-7).");
      }
    } catch {
      setStatus("Could not record your vote. It may already be resolved.");
    } finally {
      setSubmitting(false);
      setTimeout(() => router.push("/auditor/dashboard"), 1500);
    }
  }

  return (
    <form onSubmit={handleSubmit} style={{ display: "flex", flexDirection: "column", gap: "0.75rem", maxWidth: 420 }}>
      <label>
        Reputation to stake (FR-6)
        <Input type="number" min={0.1} step={0.1} value={stake} onChange={(e) => setStake(e.target.value)} required />
      </label>
      <fieldset>
        <legend>Verdict</legend>
        <label>
          <input type="radio" name="verdict" value="true" checked={verdict === "true"} onChange={() => setVerdict("true")} />
          {" "}Confirm claim
        </label>
        <br />
        <label>
          <input type="radio" name="verdict" value="false" checked={verdict === "false"} onChange={() => setVerdict("false")} />
          {" "}Reject claim (false)
        </label>
      </fieldset>
      <Button type="submit" disabled={submitting}>
        {submitting ? "Submitting…" : "Cast vote"}
      </Button>
      {status && <p role="status">{status}</p>}
    </form>
  );
}
