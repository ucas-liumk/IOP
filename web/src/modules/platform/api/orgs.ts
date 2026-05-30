import { client } from "@/api/client";
import type {
  DeptApi, DeptRow, DeptTreeRow, CreateDeptPayload, UpdateDeptPatch,
} from "@/shell/components";

// Platform-console organization (tenant) department management. These reuse the
// SAME dept service the tenant console uses; the only difference is the org's
// tenant id is carried in the path (/platform/orgs/:tid/depts*) instead of via
// tenant context. Gated server-side by the platform RBAC org:read / org:write.

const base = (tid: string) => `/platform/orgs/${tid}/depts`;

export async function getOrgDeptTree(tid: string): Promise<DeptTreeRow[]> {
  const r = await client.get(`${base(tid)}/tree`);
  return r.data?.data?.tree ?? [];
}
export async function listOrgDepts(tid: string): Promise<DeptRow[]> {
  const r = await client.get(base(tid));
  return r.data?.data?.depts ?? [];
}
export async function createOrgDept(tid: string, payload: CreateDeptPayload): Promise<DeptRow> {
  const r = await client.post(base(tid), payload);
  return r.data?.data as DeptRow;
}
export async function updateOrgDept(tid: string, id: string, patch: UpdateDeptPatch): Promise<void> {
  await client.patch(`${base(tid)}/${id}`, patch);
}
export async function deleteOrgDept(tid: string, id: string): Promise<void> {
  await client.delete(`${base(tid)}/${id}`);
}
export async function moveOrgDept(tid: string, id: string, parentId: string | null): Promise<void> {
  await client.post(`${base(tid)}/${id}/move`, { parent_id: parentId ?? "" });
}

// CSV export → blob download.
export async function downloadOrgDeptsCsv(tid: string): Promise<void> {
  const res = await client.get(`${base(tid)}/export`, { responseType: "blob" });
  const url = URL.createObjectURL(res.data as Blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = "departments.csv";
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}

// URL helpers for ImportDialog (it owns the multipart POST + template GET).
export const orgDeptTemplateUrl = (tid: string) => `${base(tid)}/template`;
export const orgDeptImportUrl = (tid: string) => `${base(tid)}/import`;

// Build a DeptApi adapter for a specific org, ready to bind to <DeptTreeManager>.
export function orgDeptApi(tid: string): DeptApi {
  return {
    fetchTree: () => getOrgDeptTree(tid),
    fetchFlat: () => listOrgDepts(tid),
    create: (p) => createOrgDept(tid, p),
    update: (id, patch) => updateOrgDept(tid, id, patch),
    remove: (id) => deleteOrgDept(tid, id),
    move: (id, parentId) => moveOrgDept(tid, id, parentId),
    exportCsv: () => downloadOrgDeptsCsv(tid),
  };
}
