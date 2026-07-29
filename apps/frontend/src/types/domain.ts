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

export interface PendingClaim {
  id: string;
  articleId: string;
  articleTitle: string;
  text: string;
  tag: string;
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
