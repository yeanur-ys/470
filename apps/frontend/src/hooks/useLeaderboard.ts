"use client";

import { useEffect, useState } from "react";

import { getLeaderboard } from "@/lib/models/leaderboard";
import type { LeaderboardEntry } from "@/types/domain";

// F-19: Instant Global Leaderboards. The backend reads this straight from a
// Redis sorted set (NFR-3: near-instant regardless of historical node count)
// — this hook just fetches whatever it returns.
export function useLeaderboard() {
  const [entries, setEntries] = useState<LeaderboardEntry[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    getLeaderboard()
      .then(setEntries)
      .catch(() => setError("Could not load the leaderboard."));
  }, []);

  return { entries, error };
}
