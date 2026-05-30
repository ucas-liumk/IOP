import { client } from "@/api/client";

export interface TaskList {
  id: string;
  name: string;
  color: string;
  sort_order: number;
  archived: boolean;
  task_count: number;
  done_count: number;
}

export interface Task {
  id: string;
  list_id?: string | null;
  parent_id?: string | null;
  title: string;
  note: string;
  priority: number; // 0..3
  status: "todo" | "done";
  due_date?: string | null;
  completed_at?: string | null;
  tags: string[];
  sort_order: number;
  created_at: string;
  updated_at: string;
  subtasks?: Task[];
}

export type SmartView = "today" | "next7" | "overdue" | "completed" | "all";

const base = "/apps/tasks";

export async function listLists(): Promise<TaskList[]> {
  const r = await client.get(`${base}/lists`);
  return r.data?.data?.lists ?? [];
}
export async function createList(name: string, color: string): Promise<TaskList> {
  const r = await client.post(`${base}/lists`, { name, color });
  return r.data?.data;
}
export async function updateList(id: string, patch: { name: string; color: string; sort_order?: number; archived?: boolean }) {
  await client.patch(`${base}/lists/${id}`, patch);
}
export async function deleteList(id: string) {
  await client.delete(`${base}/lists/${id}`);
}

export interface TaskQuery {
  view?: SmartView;
  list_id?: string;
  status?: "todo" | "done";
  tag?: string;
}
export async function listTasks(q: TaskQuery = {}): Promise<Task[]> {
  const params: Record<string, string> = {};
  if (q.view && q.view !== "all") params.view = q.view;
  if (q.list_id) params.list_id = q.list_id;
  if (q.status) params.status = q.status;
  if (q.tag) params.tag = q.tag;
  const r = await client.get(`${base}/tasks`, { params });
  return r.data?.data?.tasks ?? [];
}
export async function getTask(id: string): Promise<Task> {
  const r = await client.get(`${base}/tasks/${id}`);
  return r.data?.data;
}
export async function getCounts(): Promise<Record<string, number>> {
  const r = await client.get(`${base}/tasks/counts`);
  return r.data?.data?.counts ?? {};
}
export async function createTask(payload: {
  title: string; note?: string; priority?: number;
  list_id?: string; parent_id?: string; due_date?: string; tags?: string[];
}): Promise<Task> {
  const r = await client.post(`${base}/tasks`, payload);
  return r.data?.data;
}
export async function updateTask(id: string, patch: Partial<{
  title: string; note: string; priority: number;
  list_id: string; due_date: string; tags: string[];
}>): Promise<Task> {
  const r = await client.patch(`${base}/tasks/${id}`, patch);
  return r.data?.data;
}
export async function completeTask(id: string): Promise<Task> {
  const r = await client.post(`${base}/tasks/${id}/complete`);
  return r.data?.data;
}
export async function reopenTask(id: string): Promise<Task> {
  const r = await client.post(`${base}/tasks/${id}/reopen`);
  return r.data?.data;
}
export async function deleteTask(id: string) {
  await client.delete(`${base}/tasks/${id}`);
}
