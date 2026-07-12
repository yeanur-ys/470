"use client";

import { useEffect, useState } from "react";
import Link from "next/link";

import { apiGet } from "@/lib/api";
import { PageHeader } from "@/components/PageHeader";
import { MarginLog } from "@/components/MarginLog";
import { Stamp } from "@/components/Stamp";

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

  useEffect(() => {
    apiGet<Article[]>("/articles/mine")
      .then(setArticles)
      .catch(() => setError("Could not load your articles."));
  }, []);

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
