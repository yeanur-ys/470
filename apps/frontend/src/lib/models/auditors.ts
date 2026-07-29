import { apiGet, apiPostVoid } from "@/lib/api";
import type { PendingAuditor } from "@/types/domain";

export function getPendingAuditors(): Promise<PendingAuditor[]> {
  return apiGet<PendingAuditor[]>("/admin/auditors/pending");
}

export function verifyAuditor(id: string): Promise<void> {
  return apiPostVoid(`/admin/auditors/${id}/verify`, {});
}
