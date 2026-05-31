import { client } from "@/api/client";

export type FieldType =
  | "text"
  | "textarea"
  | "number"
  | "date"
  | "select"
  | "checkbox"
  | "money"
  | "phone";

export interface FormField {
  key: string;
  label: string;
  type: FieldType;
  required: boolean;
  options?: string[];
}

export interface FormDef {
  id: string;
  code: string;
  name: string;
  icon: string;
  fields: FormField[];
  status: "active" | "archived";
  created_by: string;
  created_at: string;
  updated_at: string;
  entry_count: number;
}

export interface FormEntry {
  id: string;
  form_id: string;
  data: Record<string, any>;
  submitted_by: string;
  created_at: string;
}

export interface Page<T> {
  data: T[];
  total: number;
  page: number;
  page_size: number;
}

export interface SaveFormPayload {
  code?: string;
  name: string;
  icon?: string;
  status?: "active" | "archived";
  fields: FormField[];
}

const base = "/apps/lcform";

// ---- Form definitions ----
export async function listForms(includeArchived = false): Promise<FormDef[]> {
  const params = includeArchived ? { include_archived: "1" } : {};
  const r = await client.get(`${base}/forms`, { params });
  return r.data?.data?.forms ?? [];
}

export async function getForm(id: string): Promise<FormDef> {
  const r = await client.get(`${base}/forms/${id}`);
  return r.data?.data;
}

export async function createForm(payload: SaveFormPayload): Promise<FormDef> {
  const r = await client.post(`${base}/forms`, payload);
  return r.data?.data;
}

export async function updateForm(id: string, payload: SaveFormPayload): Promise<FormDef> {
  const r = await client.put(`${base}/forms/${id}`, payload);
  return r.data?.data;
}

export async function deleteForm(id: string): Promise<void> {
  await client.delete(`${base}/forms/${id}`);
}

// ---- Entries ----
export async function submitEntry(formId: string, data: Record<string, any>): Promise<FormEntry> {
  const r = await client.post(`${base}/forms/${formId}/entries`, { data });
  return r.data?.data;
}

export async function listEntries(
  formId: string,
  opts: { page?: number; page_size?: number; search?: string } = {},
): Promise<Page<FormEntry>> {
  const params: Record<string, any> = {
    page: opts.page ?? 1,
    page_size: opts.page_size ?? 20,
  };
  if (opts.search) params.search = opts.search;
  const r = await client.get(`${base}/forms/${formId}/entries`, { params });
  return r.data?.data ?? { data: [], total: 0, page: 1, page_size: 20 };
}

// exportEntries downloads the CSV as a file (browser save dialog).
export async function exportEntries(formId: string, filename: string): Promise<void> {
  const r = await client.get(`${base}/forms/${formId}/entries/export`, { responseType: "blob" });
  const url = URL.createObjectURL(new Blob([r.data], { type: "text/csv;charset=utf-8" }));
  const a = document.createElement("a");
  a.href = url;
  a.download = filename || "entries.csv";
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}
