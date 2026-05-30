import { client } from "@/api/client";

// A mind-map node tree as produced/consumed by simple-mind-map.
export interface MindNode {
  data: { text: string; [k: string]: unknown };
  children: MindNode[];
}

export interface Mindmap {
  id: string;
  created_by: string;
  title: string;
  data?: MindNode; // omitted in list responses, present on get
  created_at: string;
  updated_at: string;
}

const base = "/apps/mindmap";

export async function listMaps(): Promise<Mindmap[]> {
  const r = await client.get(`${base}/maps`);
  return r.data?.data?.maps ?? [];
}

export async function getMap(id: string): Promise<Mindmap> {
  const r = await client.get(`${base}/maps/${id}`);
  return r.data?.data;
}

export async function createMap(title: string, data?: MindNode): Promise<Mindmap> {
  const r = await client.post(`${base}/maps`, { title, data });
  return r.data?.data;
}

export async function updateMap(
  id: string,
  patch: { title?: string; data?: MindNode },
): Promise<Mindmap> {
  const r = await client.put(`${base}/maps/${id}`, patch);
  return r.data?.data;
}

export async function deleteMap(id: string): Promise<void> {
  await client.delete(`${base}/maps/${id}`);
}
