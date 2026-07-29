import { apiGet, apiPost, apiPostVoid } from "@/lib/api";
import type { Article, ArticleDetail } from "@/types/domain";

export function getMyArticles(): Promise<Article[]> {
  return apiGet<Article[]>("/articles/mine");
}

export function getAllArticles(): Promise<Article[]> {
  return apiGet<Article[]>("/articles");
}

export interface CreateArticleInput {
  title: string;
  body: string;
  signature: string;
  parentArticleId?: string;
}

export function createArticle(input: CreateArticleInput): Promise<{ id: string }> {
  return apiPost<{ id: string }>("/articles", input);
}

export function getArticleDetail(articleId: string): Promise<ArticleDetail> {
  return apiGet<ArticleDetail>(`/articles/${articleId}`);
}

// Fire-and-forget: counts toward readership volume (FR-12) and the
// journalist's rank score input, but a failed increment shouldn't block the
// reader from seeing the story — callers should not await this.
export function recordArticleRead(articleId: string): Promise<void> {
  return apiPostVoid(`/articles/${articleId}/read`, {});
}
