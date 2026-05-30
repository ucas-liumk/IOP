import { client } from "@/api/client";

export interface Project {
  id: string;
  name: string;
  description: string;
  status: "active" | "archived";
  created_by: string;
  created_at: string;
  updated_at: string;
  columns?: Column[];
  card_count: number;
}

export interface Column {
  id: string;
  project_id: string;
  name: string;
  order_num: number;
  cards?: Card[];
}

export interface Card {
  id: string;
  project_id: string;
  column_id: string;
  title: string;
  description: string;
  assignee_id?: string | null;
  due_date?: string | null;
  priority: number; // 0..3
  order_num: number;
  created_at: string;
  updated_at: string;
}

const base = "/apps/project";

// ---- Projects ----
export async function listProjects(): Promise<Project[]> {
  const r = await client.get(`${base}/projects`);
  return r.data?.data?.projects ?? [];
}
export async function createProject(payload: { name: string; description?: string }): Promise<Project> {
  const r = await client.post(`${base}/projects`, payload);
  return r.data?.data;
}
export async function getBoard(id: string): Promise<Project> {
  const r = await client.get(`${base}/projects/${id}`);
  return r.data?.data;
}
export async function updateProject(
  id: string,
  patch: Partial<{ name: string; description: string; status: "active" | "archived" }>,
): Promise<Project> {
  const r = await client.patch(`${base}/projects/${id}`, patch);
  return r.data?.data;
}
export async function deleteProject(id: string) {
  await client.delete(`${base}/projects/${id}`);
}

// ---- Columns ----
export async function createColumn(projectId: string, name: string): Promise<Column> {
  const r = await client.post(`${base}/projects/${projectId}/columns`, { name });
  return r.data?.data;
}
export async function updateColumn(id: string, patch: Partial<{ name: string; order_num: number }>): Promise<Column> {
  const r = await client.patch(`${base}/columns/${id}`, patch);
  return r.data?.data;
}
export async function deleteColumn(id: string) {
  await client.delete(`${base}/columns/${id}`);
}

// ---- Cards ----
export async function createCard(
  projectId: string,
  payload: {
    column_id: string;
    title: string;
    description?: string;
    assignee_id?: string;
    due_date?: string;
    priority?: number;
  },
): Promise<Card> {
  const r = await client.post(`${base}/projects/${projectId}/cards`, payload);
  return r.data?.data;
}
export async function getCard(id: string): Promise<Card> {
  const r = await client.get(`${base}/cards/${id}`);
  return r.data?.data;
}
export async function updateCard(
  id: string,
  patch: Partial<{
    title: string;
    description: string;
    priority: number;
    assignee_id: string;
    due_date: string;
  }>,
): Promise<Card> {
  const r = await client.patch(`${base}/cards/${id}`, patch);
  return r.data?.data;
}
export async function moveCard(id: string, column_id: string, order_num: number): Promise<Card> {
  const r = await client.post(`${base}/cards/${id}/move`, { column_id, order_num });
  return r.data?.data;
}
export async function deleteCard(id: string) {
  await client.delete(`${base}/cards/${id}`);
}
