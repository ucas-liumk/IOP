import { client } from "@/api/client";

export type NodeType = "folder" | "doc";

export interface DocNode {
  id: string;
  parent_id?: string | null;
  title: string;
  type: NodeType;
  content?: string;
  order_num: number;
  created_by?: string;
  updated_by?: string;
  created_at: string;
  updated_at: string;
  children?: DocNode[];
}

const base = "/apps/docs";

export async function getTree(): Promise<DocNode[]> {
  const r = await client.get(`${base}/tree`);
  return r.data?.data?.tree ?? [];
}

export async function getDoc(id: string): Promise<DocNode> {
  const r = await client.get(`${base}/docs/${id}`);
  return r.data?.data;
}

export async function createNode(payload: {
  title: string;
  type: NodeType;
  parent_id?: string;
  content?: string;
}): Promise<DocNode> {
  const r = await client.post(`${base}/docs`, payload);
  return r.data?.data;
}

// Save a doc's content (and optionally rename it in the same call).
export async function saveDoc(id: string, content: string, title?: string): Promise<DocNode> {
  const r = await client.put(`${base}/docs/${id}`, { content, ...(title !== undefined ? { title } : {}) });
  return r.data?.data;
}

// Rename a node (folder or doc) without touching content.
export async function renameNode(id: string, title: string): Promise<DocNode> {
  const r = await client.put(`${base}/docs/${id}`, { title });
  return r.data?.data;
}

export async function moveNode(id: string, parentId: string | null, orderNum = 0): Promise<void> {
  await client.post(`${base}/docs/${id}/move`, { parent_id: parentId ?? "", order_num: orderNum });
}

export async function deleteNode(id: string): Promise<void> {
  await client.delete(`${base}/docs/${id}`);
}
