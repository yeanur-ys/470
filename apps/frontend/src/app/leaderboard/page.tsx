"use client";

import Link from "next/link";

import { PublicHeader } from "@/components/PublicHeader";
import { PageHeader } from "@/components/PageHeader";
import { useLeaderboard } from "@/hooks/useLeaderboard";

export default function LeaderboardPage() {
  const { entries, error } = useLeaderboard();

  return (
    <>
      <PublicHeader />
      <main style={{ maxWidth: 680, margin: "0 auto", padding: "2rem 2rem 4rem" }}>
        <PageHeader
          eyebrow="Open to everyone"
          title="Leaderboard"
          description="Ranked by Journalist Rank Score (R) — readership, weighted toward rewarding self-correction over waiting to be caught, weighted against proven false claims."
        />
        {error && <p className="notice" data-tone="alert">{error}</p>}
        {!entries && !error && <p className="notice">Loading…</p>}
        {entries && entries.length === 0 && <p className="notice">No ranked journalists yet.</p>}
        {entries && entries.length > 0 && (
          <table className="ledger">
            <thead>
              <tr>
                <th style={{ width: "3rem" }}>#</th>
                <th>Journalist</th>
                <th className="num">Rank score</th>
              </tr>
            </thead>
            <tbody>
              {entries.map((e, i) => (
                <tr key={e.journalistId}>
                  <td className="mono">{i + 1}</td>
                  <td>
                    <Link href={`/profile/${e.journalistId}`}>{e.displayName || e.journalistId}</Link>
                  </td>
                  <td className="num mono">{e.rankScore.toFixed(2)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </main>
    </>
  );
}
