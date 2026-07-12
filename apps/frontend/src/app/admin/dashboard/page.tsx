"use client";

import { useEffect, useState } from "react";
import Link from "next/link";

import { apiGet } from "@/lib/api";

interface Article {
  id: string;
  title: string;
  journalistId: string;
  isRetracted: boolean;
  createdAt: string;
}

export default function AdminDashboardPage() {
  const [articles, setArticles] = useState<Article[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    apiGet<Article[]>("/articles")
      .then(setArticles)
      .catch(() => setError("Could not load articles."));
  }, []);

  return (
    <div>
      <p>
        <Link href="/admin/compliance">Process a retraction request</Link>
      </p>
      {error && <p role="alert">{error}</p>}
      {!articles && !error && <p>Loading…</p>}
      {articles && (
        <table style={{ width: "100%", borderCollapse: "collapse" }}>
          <thead>
            <tr>
              <th style={{ textAlign: "left" }}>Title</th>
              <th style={{ textAlign: "left" }}>Article ID</th>
              <th>Status</th>
            </tr>
          </thead>
          <tbody>
            {articles.map((a) => (
              <tr key={a.id} style={{ borderTop: "1px solid #eee" }}>
                <td>{a.title}</td>
                <td><code>{a.id}</code></td>
                <td>{a.isRetracted ? "Retracted" : "Live"}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
