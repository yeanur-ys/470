"use client";

import { useEffect, useState } from "react";
import Link from "next/link";

import { apiGet } from "@/lib/api";

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

  return (
    <div>
      <p>
        <Link href="/journalist/publish">Publish a new article</Link> ·{" "}
        <Link href="/journalist/appeals">File an appeal</Link>
      </p>

      {error && <p role="alert">{error}</p>}
      {!articles && !error && <p>Loading…</p>}
      {articles && articles.length === 0 && <p>You haven't published anything yet.</p>}

      {articles && articles.length > 0 && (
        <table style={{ width: "100%", borderCollapse: "collapse" }}>
          <thead>
            <tr>
              <th style={{ textAlign: "left" }}>Title</th>
              <th style={{ textAlign: "right" }}>Reads</th>
              <th style={{ textAlign: "right" }}>Verified</th>
              <th style={{ textAlign: "right" }}>Self-corrected</th>
              <th style={{ textAlign: "right" }}>False</th>
              <th>Status</th>
            </tr>
          </thead>
          <tbody>
            {articles.map((a) => (
              <tr key={a.id} style={{ borderTop: "1px solid #eee" }}>
                <td>{a.title}</td>
                <td style={{ textAlign: "right" }}>{a.readershipVolume}</td>
                <td style={{ textAlign: "right" }}>{a.verifiedClaims}</td>
                <td style={{ textAlign: "right" }}>{a.selfCorrectedClaims}</td>
                <td style={{ textAlign: "right" }}>{a.falseClaims}</td>
                <td>{a.isRetracted ? "Retracted" : "Live"}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
