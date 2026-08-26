import { apiGet, apiPost, apiPostVoid } from "@/lib/api";
import type { Article, ArticleDetail, PaginatedArticles } from "@/types/domain";

export function getMyArticles(): Promise<Article[]> {
  return apiGet<Article[]>("/articles/mine");
}

export interface GetAllArticlesParams {
  page?: number;
  pageSize?: number;
}

// Paginated, newest-first (F-list pagination on /read: 20 at a time). Omit
// pageSize to get the backend's "everything in one page" default (100) — the
// admin compliance ledger uses that, since it wants the whole ledger visible
// at once rather than paged.
export function getAllArticles(params: GetAllArticlesParams = {}): Promise<PaginatedArticles> {
  const query = new URLSearchParams();
  if (params.page) query.set("page", String(params.page));
  if (params.pageSize) query.set("pageSize", String(params.pageSize));
  const qs = query.toString();
  return apiGet<PaginatedArticles>(`/articles${qs ? `?${qs}` : ""}`);
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
