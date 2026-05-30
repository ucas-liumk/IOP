import { client } from "@/api/client";

export interface NewsCategory {
  id: string;
  name: string;
  order_num: number;
  article_count: number;
}

export type ArticleStatus = "draft" | "published";

export interface NewsArticle {
  id: string;
  category_id?: string | null;
  title: string;
  summary: string;
  content: string;
  cover_url: string;
  author: string;
  status: ArticleStatus;
  published_at?: string | null;
  views: number;
  created_by: string;
  created_at: string;
  updated_at: string;
  category_name?: string;
}

export interface Paged<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
}

const base = "/apps/news";

// ---- Categories ----
export async function listCategories(): Promise<NewsCategory[]> {
  const r = await client.get(`${base}/categories`);
  return r.data?.data?.categories ?? [];
}
export async function createCategory(name: string, order_num = 0): Promise<NewsCategory> {
  const r = await client.post(`${base}/categories`, { name, order_num });
  return r.data?.data;
}
export async function updateCategory(id: string, patch: { name: string; order_num?: number }) {
  await client.patch(`${base}/categories/${id}`, patch);
}
export async function deleteCategory(id: string) {
  await client.delete(`${base}/categories/${id}`);
}

// ---- Articles (management) ----
export interface ArticleQuery {
  status?: ArticleStatus;
  category_id?: string;
  keyword?: string;
  page?: number;
  page_size?: number;
}
export async function listArticles(q: ArticleQuery = {}): Promise<Paged<NewsArticle>> {
  const params: Record<string, string | number> = {};
  if (q.status) params.status = q.status;
  if (q.category_id) params.category_id = q.category_id;
  if (q.keyword) params.keyword = q.keyword;
  if (q.page) params.page = q.page;
  if (q.page_size) params.page_size = q.page_size;
  const r = await client.get(`${base}/articles`, { params });
  return r.data?.data ?? { items: [], total: 0, page: 1, page_size: 20 };
}
export async function getArticle(id: string, track = false): Promise<NewsArticle> {
  const r = await client.get(`${base}/articles/${id}`, { params: track ? { track: "1" } : {} });
  return r.data?.data;
}
export async function createArticle(payload: {
  title: string;
  category_id?: string;
  summary?: string;
  content?: string;
  cover_url?: string;
  author?: string;
}): Promise<NewsArticle> {
  const r = await client.post(`${base}/articles`, payload);
  return r.data?.data;
}
export async function updateArticle(
  id: string,
  patch: Partial<{
    title: string;
    summary: string;
    content: string;
    cover_url: string;
    author: string;
    category_id: string; // "" clears
  }>,
): Promise<NewsArticle> {
  const r = await client.patch(`${base}/articles/${id}`, patch);
  return r.data?.data;
}
export async function publishArticle(id: string): Promise<NewsArticle> {
  const r = await client.post(`${base}/articles/${id}/publish`);
  return r.data?.data;
}
export async function unpublishArticle(id: string): Promise<NewsArticle> {
  const r = await client.post(`${base}/articles/${id}/unpublish`);
  return r.data?.data;
}
export async function deleteArticle(id: string) {
  await client.delete(`${base}/articles/${id}`);
}

// ---- Reader feed (published) ----
export async function listFeed(category = "", page = 1, page_size = 10): Promise<Paged<NewsArticle>> {
  const params: Record<string, string | number> = { page, page_size };
  if (category) params.category = category;
  const r = await client.get(`${base}/feed`, { params });
  return r.data?.data ?? { items: [], total: 0, page: 1, page_size };
}
