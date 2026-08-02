import { apiGet, apiPostVoid } from "@/lib/api";
import type { AuditorStats, PendingAuditor } from "@/types/domain";

export function getPendingAuditors(): Promise<PendingAuditor[]> {
  return apiGet<PendingAuditor[]>("/admin/auditors/pending");
}

// The requesting auditor's own standing, for their personal dashboard.
export function getAuditorStats(): Promise<AuditorStats> {
  return apiGet<AuditorStats>("/auditor/me");
}

export function verifyAuditor(id: string): Promise<void> {
  return apiPostVoid(`/admin/auditors/${id}/verify`, {});
}
