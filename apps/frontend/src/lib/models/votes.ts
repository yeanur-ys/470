import { apiPost } from "@/lib/api";
import type { VoteResult } from "@/types/domain";

export function castVote(claimId: string, stake: number, verdict: boolean): Promise<VoteResult> {
  return apiPost<VoteResult>(`/claims/${claimId}/votes`, { stake, verdict });
}
