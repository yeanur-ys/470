// Shared domain types (the frontend's "Model" layer, alongside lib/models/).
// Mirrors the Go backend's response shapes 1:1 (see apps/go-backend/internal
// /models and /views) — one definition per entity instead of the ad hoc,
// slightly-different Article/PendingClaim/etc. interfaces that used to be
// redeclared in 4+ page files.

import type { Role } from "@/lib/auth";

export type { Role };

export interface AuthSession {
  token: string;
  role: Role;
  userId: string;
}

export interface Article {
  id: string;
  journalistId: string;
  parentArticleId?: string;
  title: string;
  body: string;
  signature: string;
  readershipVolume: number;
  verifiedClaims: number;
  selfCorrectedClaims: number;
  falseClaims: number;
  isRetracted: boolean;
  // Only populated on /articles/mine — an article can carry at most one
  // active appeal at a time (see hooks/useAppeals.ts), so this is what lets
  // the appeals form warn before a doomed second attempt on the same story.
  hasActiveAppeal?: boolean;
  createdAt: string;
}

export type ClaimStatus = "pending" | "verified" | "self_corrected" | "false";

export interface ArticleClaim {
  id: string;
  text: string;
  tag: string;
  status: ClaimStatus;
}

export interface ArticleDetail extends Article {
  claims: ArticleClaim[];
}

export interface PaginatedArticles {
  articles: Article[];
  total: number;
  page: number;
  pageSize: number;
}

export interface PendingClaim {
  id: string;
  articleId: string;
  articleTitle: string;
  text: string;
  tag: string;
}

export interface PaginatedPendingClaims {
  claims: PendingClaim[];
  total: number;
  page: number;
  pageSize: number;
}

// Every claim tagged across a journalist's own articles — the dashboard's
// source for self-correcting a claim on any article, not just the one most
// recently published (see hooks/useJournalistDashboard.ts).
export interface JournalistClaim {
  id: string;
  articleId: string;
  articleTitle: string;
  text: string;
  tag: string;
  status: ClaimStatus;
}

export interface VoteResult {
  resolved?: boolean;
  verdict?: boolean;
}

export interface AppealStakeResult {
  stakedRankScore: number;
}

export interface PendingAuditor {
  id: string;
  email: string;
  displayName: string;
  credentialUrl: string;
  tags: string[];
}

// The requesting auditor's own standing (their personal dashboard), distinct
// from PendingAuditor (an admin's view of others). Mirrors models.AuditorStats.
export interface AuditorStats {
  id: string;
  displayName: string;
  credentialVerified: boolean;
  credentialUrl: string;
  tags: string[];
  rankScore: number;
  trustWeight: number;
  successfulVotes: number;
  failedVotes: number;
  votesCast: number;
  lockedStake: number;
  availableStake: number;
}

export interface LeaderboardEntry {
  journalistId: string;
  displayName: string;
  rankScore: number;
}

export interface RetractionResult {
  status: string;
  tombstoneHash?: string;
  detail?: string;
}

export interface GraphNode {
  id: string;
  title: string;
  journalistId?: string;
  journalistName?: string;
  readershipVolume: number;
  corruptionFactor: number;
  clusterId?: number;
  clusterLabel?: string;
  tags: string[];
  isRetracted: boolean;
  hasActiveAppeal: boolean;
  createdAt?: string;
}

export interface GraphEdge {
  source: string;
  target: string;
  kind: "sequence" | "topic";
}

export interface ClusterSummary {
  id: number;
  label: string;
  size: number;
}

export interface GraphResponse {
  nodes: GraphNode[];
  edges: GraphEdge[];
  clusters: ClusterSummary[];
  truncated: boolean;
}
