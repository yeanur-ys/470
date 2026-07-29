import { apiPost } from "@/lib/api";
import type { RetractionResult } from "@/types/domain";

export function retractArticle(articleId: string): Promise<RetractionResult> {
  return apiPost<RetractionResult>(`/admin/articles/${articleId}/retract`, {});
}
