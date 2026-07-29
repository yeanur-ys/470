// Central registry of Redis key patterns (Section 5.1: "Redis - Hash cache for
// real-time read counts and Sorted Sets for the global leaderboard").

export const REDIS_KEYS = {
  articleReadCount: (articleId: string) => `article:${articleId}:reads`,
  leaderboard: () => "leaderboard:journalist_rank", // ZSET: member=journalistId, score=R
  auditorTrust: (auditorId: string) => `auditor:${auditorId}:trust_weight`,
};
