"use client";

import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { useClaimVote } from "@/hooks/useClaimVote";

export function ClaimVoteForm({ claimId }: { claimId: string }) {
  const { stake, setStake, verdict, setVerdict, status, tone, submitting, handleSubmit } = useClaimVote(claimId);

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
