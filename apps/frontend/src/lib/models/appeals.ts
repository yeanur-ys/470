import { apiPostVoid } from "@/lib/api";

export function createAppeal(articleId: string, stakedPercent: number): Promise<void> {
  return apiPostVoid("/appeals", { articleId, stakedPercent });
}
