"use client";

import { useEffect, useState } from "react";
import Link from "next/link";

import { apiGet, apiPostVoid } from "@/lib/api";
import { PageHeader } from "@/components/PageHeader";
import { MarginLog } from "@/components/MarginLog";
import { Stamp } from "@/components/Stamp";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";

interface Article {
  id: string;
  title: string;
  readershipVolume: number;
  verifiedClaims: number;
  selfCorrectedClaims: number;
  falseClaims: number;
  isRetracted: boolean;
  createdAt: string;
}

export default function JournalistDashboardPage() {
  const [articles, setArticles] = useState<Article[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  const [claimId, setClaimId] = useState("");
  const [selfCorrectStatus, setSelfCorrectStatus] = useState<string | null>(null);
  const [selfCorrectTone, setSelfCorrectTone] = useState<"ok" | "alert">("ok");
  const [submittingSelfCorrect, setSubmittingSelfCorrect] = useState(false);

  useEffect(() => {
    apiGet<Article[]>("/articles/mine")
      .then(setArticles)
      .catch(() => setError("Could not load your articles."));
  }, []);

  async function handleSelfCorrect(e: React.FormEvent) {
    e.preventDefault();
    setSelfCorrectStatus(null);
    setSubmittingSelfCorrect(true);
    try {
      await apiPostVoid(`/claims/${claimId}/self-correct`, {});
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

  const notes = buildNotes(articles);

  return (
    <>
      <PageHeader
        eyebrow="Journalist desk"
        title="Your byline"
        description="Every story you've filed, its readership, and where its claims stand."
      />

      <div className="docket">
        <div>
          <p style={{ marginBottom: "1.25rem" }}>
            <Link href="/journalist/publish" className="btn">
              File a new story
            </Link>{" "}
            <Link href="/journalist/appeals" className="btn btn--ghost">
              Dispute a ruling
            </Link>
          </p>

          {error && <p className="notice" data-tone="alert">{error}</p>}
          {!articles && !error && <p className="notice">Loading your byline…</p>}
          {articles && articles.length === 0 && (
            <p className="notice">Nothing filed yet — your first story starts the ledger.</p>
          )}

          {articles && articles.length > 0 && (
            <table className="ledger">
              <thead>
                <tr>
                  <th>Story</th>
                  <th className="num">Reads</th>
                  <th className="num">Verified</th>
                  <th className="num">Self-corrected</th>
                  <th className="num">False</th>
                  <th>Status</th>
                </tr>
              </thead>
              <tbody>
                {articles.map((a) => (
                  <tr key={a.id}>
                    <td>{a.title}</td>
                    <td className="num mono">{a.readershipVolume}</td>
                    <td className="num mono">{a.verifiedClaims}</td>
                    <td className="num mono">{a.selfCorrectedClaims}</td>
                    <td className="num mono">{a.falseClaims}</td>
                    <td>
                      {a.isRetracted ? (
                        <Stamp tone="alert">Retracted</Stamp>
                      ) : a.falseClaims > 0 ? (
                        <Stamp tone="pending">Disputed</Stamp>
                      ) : (
                        <Stamp tone="ok">Live</Stamp>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}

          <div className="card" style={{ marginTop: "1.5rem" }}>
            <span className="eyebrow">Get ahead of a mistake</span>
            <p style={{ color: "var(--ink-soft)", marginBottom: "1rem" }}>
              Mark one of your own claims self-corrected before an auditor resolves it — the
              record rewards catching your own mistakes over waiting to be caught (formula
              weighs self-correction above baseline verification). Paste a claim ID from the
              publish flow.
            </p>
            <form onSubmit={handleSelfCorrect} style={{ display: "flex", gap: "0.5rem" }}>
              <Input
                placeholder="Claim ID"
                value={claimId}
                onChange={(e) => setClaimId(e.target.value)}
                required
                style={{ flex: 1 }}
              />
              <Button type="submit" disabled={submittingSelfCorrect}>
                {submittingSelfCorrect ? "Submitting…" : "Self-correct"}
              </Button>
            </form>
            {selfCorrectStatus && (
              <p className="notice" data-tone={selfCorrectTone === "alert" ? "alert" : undefined} style={{ marginTop: "1rem" }}>
                {selfCorrectStatus}
              </p>
            )}
          </div>
        </div>

        <MarginLog notes={notes} />
      </div>
    </>
  );
}

function buildNotes(articles: Article[] | null): { text: string; tone?: "ok" | "alert" | "pending" | "neutral" }[] {
  if (!articles) return [];
  const notes: { text: string; tone?: "ok" | "alert" | "pending" | "neutral" }[] = [];

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
