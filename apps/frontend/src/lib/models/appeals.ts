import { apiPostVoid } from "@/lib/api";

export interface CreateAppealInput {
  articleId: string;
  stakedPercent: number;
  // F-14 is "Proof of Evidence": the backend rejects an appeal that carries
  // neither — at least one must be non-empty.
  evidenceText?: string;
  evidenceUrl?: string;
}

export function createAppeal(input: CreateAppealInput): Promise<void> {
  return apiPostVoid("/appeals", input);
}
