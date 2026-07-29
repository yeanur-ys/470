import { apiGet } from "@/lib/api";
import type { GraphResponse } from "@/types/domain";

export function getJournalistGraph(journalistId: string): Promise<GraphResponse> {
  return apiGet<GraphResponse>(`/journalists/${journalistId}/graph`);
}

export function getGlobalGraph(): Promise<GraphResponse> {
  return apiGet<GraphResponse>("/graph");
}
