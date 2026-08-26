import { apiGet, apiPost, apiPostVoid } from "@/lib/api";
import type { JournalistClaim, PaginatedPendingClaims } from "@/types/domain";

export interface SelfCorrectEdits {
  text?: string;
  tag?: string;
}

// Marks a claim self-corrected, optionally rewriting its text/tag in the same
// call -- a "correction" means fixing what the claim actually says, not just
// relabelling it, so editing and self-correcting are one action, not two.
export function selfCorrectClaim(claimId: string, edits: SelfCorrectEdits = {}): Promise<void> {
  return apiPostVoid(`/claims/${claimId}/self-correct`, edits);
}

export function tagClaim(articleId: string, text: string, tag: string): Promise<{ id: string }> {
  return apiPost<{ id: string }>(`/articles/${articleId}/claims`, { text, tag });
}

export interface GetPendingClaimsParams {
  page?: number;
  pageSize?: number;
}

// Paginated, oldest-first (a fair FIFO queue). Without pagination, a large
// backlog hid every claim tagged after the first (hardcoded) 100 from every
// auditor indefinitely — see models/claim.go's PendingClaims doc comment.
export function getPendingClaims(params: GetPendingClaimsParams = {}): Promise<PaginatedPendingClaims> {
  const query = new URLSearchParams();
  if (params.page) query.set("page", String(params.page));
  if (params.pageSize) query.set("pageSize", String(params.pageSize));
  const qs = query.toString();
  return apiGet<PaginatedPendingClaims>(`/claims/pending${qs ? `?${qs}` : ""}`);
}

// Every claim tagged across the journalist's own articles — lets the
// dashboard offer a self-correct button on any of them, not just one tagged
// earlier in the same publish session.
export function getMyClaims(): Promise<JournalistClaim[]> {
  return apiGet<JournalistClaim[]>("/claims/mine");
}
