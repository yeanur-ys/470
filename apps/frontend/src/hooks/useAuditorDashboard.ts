"use client";

import { useEffect, useState } from "react";

import { getPendingClaims } from "@/lib/models/claims";
import type { PendingClaim } from "@/types/domain";

export function useAuditorDashboard() {
  const [claims, setClaims] = useState<PendingClaim[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    getPendingClaims()
      .then(setClaims)
      .catch(() => setError("Could not load pending claims."));
  }, []);

  const tagCounts = new Map<string, number>();
  claims?.forEach((c) => tagCounts.set(c.tag, (tagCounts.get(c.tag) ?? 0) + 1));

  const notes = [
    ...(claims && claims.length > 0
      ? [{ text: `${claims.length} claim${claims.length === 1 ? "" : "s"} waiting on a second, non-overlapping tag.`, tone: "pending" as const }]
      : []),
    ...Array.from(tagCounts.entries()).map(([tag, count]) => ({
      text: `${count} claim${count === 1 ? "" : "s"} tagged "${tag}".`,
      tone: "neutral" as const,
    })),
  ];

  return { claims, error, notes };
}
