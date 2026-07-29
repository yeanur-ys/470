import { apiGet } from "@/lib/api";
import type { LeaderboardEntry } from "@/types/domain";

export function getLeaderboard(): Promise<LeaderboardEntry[]> {
  return apiGet<LeaderboardEntry[]>("/leaderboard");
}
