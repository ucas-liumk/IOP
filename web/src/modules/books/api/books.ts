import { client } from "@/api/client";

export interface Book {
  id: string;
  isbn: string;
  title: string;
  author: string;
  publisher: string;
  category: string;
  total: number;
  available: number;
  cover_url: string;
  location: string;
  created_at: string;
  updated_at: string;
}

export type BorrowStatus = "borrowed" | "returned" | "overdue";

export interface Borrow {
  id: string;
  book_id: string;
  member_id: string;
  borrowed_at: string;
  due_at: string;
  returned_at?: string | null;
  status: BorrowStatus;
  book_title?: string;
  book_author?: string;
  book_isbn?: string;
}

export interface BookPage {
  books: Book[];
  total: number;
  page: number;
  page_size: number;
}

export interface BookInput {
  isbn?: string;
  title: string;
  author?: string;
  publisher?: string;
  category?: string;
  total?: number;
  cover_url?: string;
  location?: string;
}

const base = "/apps/books";

export async function listBooks(q: { search?: string; category?: string; page?: number; page_size?: number } = {}): Promise<BookPage> {
  const params: Record<string, string> = {};
  if (q.search) params.search = q.search;
  if (q.category) params.category = q.category;
  if (q.page) params.page = String(q.page);
  if (q.page_size) params.page_size = String(q.page_size);
  const r = await client.get(`${base}/books`, { params });
  return r.data?.data ?? { books: [], total: 0, page: 1, page_size: 20 };
}

export async function getBook(id: string): Promise<Book> {
  const r = await client.get(`${base}/books/${id}`);
  return r.data?.data;
}

export async function createBook(payload: BookInput): Promise<Book> {
  const r = await client.post(`${base}/books`, payload);
  return r.data?.data;
}

export async function updateBook(id: string, payload: BookInput): Promise<Book> {
  const r = await client.patch(`${base}/books/${id}`, payload);
  return r.data?.data;
}

export async function deleteBook(id: string): Promise<void> {
  await client.delete(`${base}/books/${id}`);
}

export async function borrowBook(id: string): Promise<Borrow> {
  const r = await client.post(`${base}/books/${id}/borrow`);
  return r.data?.data;
}

export async function returnBorrow(borrowId: string): Promise<void> {
  await client.post(`${base}/borrows/${borrowId}/return`);
}

export async function listBorrows(scope: "mine" | "all" = "mine", status?: BorrowStatus): Promise<Borrow[]> {
  const params: Record<string, string> = { scope };
  if (status) params.status = status;
  const r = await client.get(`${base}/borrows`, { params });
  return r.data?.data?.borrows ?? [];
}
