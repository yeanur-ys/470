import { apiGet, apiPost, apiPostVoid } from "@/lib/api";
import type { PendingClaim } from "@/types/domain";

export function selfCorrectClaim(claimId: string): Promise<void> {
  return apiPostVoid(`/claims/${claimId}/self-correct`, {});
}

export function tagClaim(articleId: string, text: string, tag: string): Promise<{ id: string }> {
  return apiPost<{ id: string }>(`/articles/${articleId}/claims`, { text, tag });
}

export function getPendingClaims(): Promise<PendingClaim[]> {
  return apiGet<PendingClaim[]>("/claims/pending");
}
