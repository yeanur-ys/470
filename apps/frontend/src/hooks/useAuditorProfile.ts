"use client";

import { useEffect, useState } from "react";

import { getAuditorStats } from "@/lib/models/auditors";
import type { AuditorStats } from "@/types/domain";

export function useAuditorProfile() {
  const [stats, setStats] = useState<AuditorStats | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    getAuditorStats()
      .then(setStats)
      .catch(() => setError("Could not load your profile."));
  }, []);

  return { stats, error };
}
