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
  const [tone, setTone] = useState<"ok" | "alert" | "neutral">("neutral");
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

  return (
    <form onSubmit={handleSubmit} className="card app-main--narrow" style={{ padding: "1.5rem" }}>
      <label className="field">
        Reputation to stake
        <Input type="number" min={0.1} step={0.1} value={stake} onChange={(e) => setStake(e.target.value)} required />
      </label>
      <fieldset className="field" style={{ border: "none", padding: 0, margin: 0 }}>
        <legend className="eyebrow" style={{ marginBottom: "0.5rem" }}>
          Verdict
        </legend>
        <label style={{ display: "block", marginBottom: "0.35rem" }}>
          <input type="radio" name="verdict" value="true" checked={verdict === "true"} onChange={() => setVerdict("true")} />
          {" "}Confirm claim
        </label>
        <label style={{ display: "block" }}>
          <input type="radio" name="verdict" value="false" checked={verdict === "false"} onChange={() => setVerdict("false")} />
          {" "}Reject claim (false)
        </label>
      </fieldset>
      <Button type="submit" disabled={submitting} style={{ marginTop: "0.75rem" }}>
        {submitting ? "Submitting…" : "Cast vote"}
      </Button>
      {status && (
        <p className="notice" data-tone={tone === "alert" ? "alert" : undefined} style={{ marginTop: "1rem" }}>
          {status}
        </p>
      )}
    </form>
  );
}
