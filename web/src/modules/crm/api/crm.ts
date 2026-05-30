import { client } from "@/api/client";

export interface CrmItem {
  id: string;
  title: string;
  body: string;
  created_at: string;
}

export async function listItems(): Promise<CrmItem[]> {
  const r = await client.get("/apps/crm/items");
  return r.data?.data?.items ?? [];
}

export async function createItem(title: string, body: string): Promise<CrmItem> {
  const r = await client.post("/apps/crm/items", { title, body });
  return r.data?.data;
}
