import { client } from "@/api/client";

export type FieldType = "text" | "textarea" | "number" | "date" | "select" | "radio";

export interface Field {
  key: string;
  label: string;
  type: FieldType;
  required: boolean;
  options: string[];
}

export type NodeType = "approve" | "cc";
export type AssigneeType = "user" | "role" | "dept_leader";
export type FlowMode = "and" | "or";

export interface FlowNode {
  type: NodeType;
  assignee_type: AssigneeType;
  assignee_id: string; // member id (assignee_type=user)
  role_code: string; // role code (assignee_type=role)
  mode: FlowMode;
}

export interface Form {
  id: string;
  code: string;
  name: string;
  icon: string;
  description: string;
  fields: Field[];
  flow: FlowNode[];
  status: "active" | "disabled";
  created_by: string;
  created_at: string;
  updated_at: string;
}

export type InstanceStatus = "pending" | "approved" | "rejected" | "canceled";
export type TaskStatus = "pending" | "approved" | "rejected" | "read" | "canceled";

export interface Task {
  id: string;
  instance_id: string;
  node_index: number;
  assignee_id: string;
  type: NodeType;
  mode: FlowMode;
  status: TaskStatus;
  comment: string;
  acted_at?: string | null;
  created_at: string;
  assignee_name?: string;
  form_name?: string;
  instance_status?: InstanceStatus;
  mine?: boolean;
}

export interface Instance {
  id: string;
  form_id: string;
  form_name: string;
  fields: Field[];
  data: Record<string, unknown>;
  flow: FlowNode[];
  initiator_id: string;
  status: InstanceStatus;
  current_node: number;
  created_at: string;
  finished_at?: string | null;
  initiator_name?: string;
  tasks?: Task[];
  can_cancel?: boolean;
}

export interface MemberRef {
  id: string;
  name: string;
  department: string;
}

export type InboxType = "todo" | "done" | "initiated" | "cc";

const base = "/apps/approval";

// ---- directory ----
export async function listMembers(): Promise<MemberRef[]> {
  const r = await client.get(`${base}/members`);
  return r.data?.data?.members ?? [];
}

// ---- forms ----
export async function listForms(includeDisabled = false): Promise<Form[]> {
  const r = await client.get(`${base}/forms`, { params: includeDisabled ? { all: "1" } : {} });
  return r.data?.data?.forms ?? [];
}
export async function getForm(id: string): Promise<Form> {
  const r = await client.get(`${base}/forms/${id}`);
  return r.data?.data;
}
export interface FormPayload {
  code?: string;
  name: string;
  icon?: string;
  description?: string;
  fields: Field[];
  flow: FlowNode[];
  status?: "active" | "disabled";
}
export async function createForm(p: FormPayload): Promise<Form> {
  const r = await client.post(`${base}/forms`, p);
  return r.data?.data;
}
export async function updateForm(id: string, p: FormPayload): Promise<Form> {
  const r = await client.put(`${base}/forms/${id}`, p);
  return r.data?.data;
}
export async function deleteForm(id: string): Promise<void> {
  await client.delete(`${base}/forms/${id}`);
}

// ---- instances ----
export async function submit(formId: string, data: Record<string, unknown>): Promise<Instance> {
  const r = await client.post(`${base}/instances`, { form_id: formId, data });
  return r.data?.data;
}
export async function inbox(type: InboxType): Promise<{ type: InboxType; items: Task[] | Instance[] }> {
  const r = await client.get(`${base}/instances`, { params: { type } });
  return { type, items: r.data?.data?.items ?? [] };
}
export async function getInstance(id: string): Promise<Instance> {
  const r = await client.get(`${base}/instances/${id}`);
  return r.data?.data;
}
export async function cancelInstance(id: string): Promise<void> {
  await client.post(`${base}/instances/${id}/cancel`);
}

// ---- tasks ----
export async function act(taskId: string, action: "approve" | "reject" | "read", comment = ""): Promise<void> {
  await client.post(`${base}/tasks/${taskId}/act`, { action, comment });
}
