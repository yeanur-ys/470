"use client";

import { useEffect, useState } from "react";

import { getPendingClaims } from "@/lib/models/claims";
import { getAuditorStats } from "@/lib/models/auditors";
import type { AuditorStats, PendingClaim } from "@/types/domain";

const PAGE_SIZE = 20;

export function useAuditorDashboard() {
  const [claims, setClaims] = useState<PendingClaim[] | null>(null);
  const [stats, setStats] = useState<AuditorStats | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);

  useEffect(() => {
    getAuditorStats().then(setStats).catch(() => setStats(null));
  }, []);

  useEffect(() => {
    setClaims(null);
    getPendingClaims({ page, pageSize: PAGE_SIZE })
      .then((res) => {
        setClaims(res.claims);
        setTotal(res.total);
      })
      .catch(() => setError("Could not load pending claims."));
  }, [page]);

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  const tagCounts = new Map<string, number>();
  claims?.forEach((c) => tagCounts.set(c.tag, (tagCounts.get(c.tag) ?? 0) + 1));

  const notes = [
    ...(total > 0
      ? [{ text: `${total} claim${total === 1 ? "" : "s"} in the queue, oldest first — showing page ${page} of ${totalPages}.`, tone: "pending" as const }]
      : []),
    ...Array.from(tagCounts.entries()).map(([tag, count]) => ({
      text: `${count} claim${count === 1 ? "" : "s"} on this page tagged "${tag}".`,
      tone: "neutral" as const,
    })),
  ];

  return { claims, stats, error, notes, page, setPage, totalPages };
}
