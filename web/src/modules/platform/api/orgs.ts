import { client } from "@/api/client";
import type {
  DeptApi, DeptRow, DeptTreeRow, CreateDeptPayload, UpdateDeptPatch,
  MemberApi, MemberPage, MemberListParams,
  MemberDeptRow, MemberDeptTreeRow, RoleRow, PostRow, MemberRow,
} from "@/shell/components";
import { resetPlatformUserPassword } from "@/modules/admin/api/admin";

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

// === Platform-console member management for a specific org ===
// Mirrors the tenant console's /admin/members* exactly, only the org's tenant id
// is carried in the path (/platform/orgs/:tid/members*). Gated server-side by the
// platform RBAC user:read / user:write. Username (not email) is the identity.

const mbase = (tid: string) => `/platform/orgs/${tid}/members`;

export async function listOrgMembers(tid: string, p: MemberListParams): Promise<MemberPage> {
  const query: Record<string, string | number> = {
    page: p.page ?? 1,
    page_size: p.pageSize ?? 20,
  };
  if (p.search) query.search = p.search;
  if (p.deptId) query.dept_id = p.deptId;
  if (p.subtree) query.subtree = "true";
  const r = await client.get(mbase(tid), { params: query });
  const d = r.data?.data ?? {};
  return {
    data: d.data ?? d.members ?? [],
    total: d.total ?? 0,
    page: d.page ?? (query.page as number),
    pageSize: d.page_size ?? (query.page_size as number),
  };
}

export async function setOrgMemberDept(tid: string, memberId: string, deptId: string | null): Promise<void> {
  await client.patch(`${mbase(tid)}/${memberId}`, { dept_id: deptId ?? "" });
}
export async function assignOrgMemberPost(tid: string, memberId: string, postId: string): Promise<void> {
  await client.post(`${mbase(tid)}/${memberId}/posts`, { post_id: postId });
}
export async function removeOrgMemberPost(tid: string, memberId: string, postId: string): Promise<void> {
  await client.delete(`${mbase(tid)}/${memberId}/posts/${postId}`);
}
export async function getOrgMemberRoles(tid: string, memberId: string): Promise<RoleRow[]> {
  const r = await client.get(`${mbase(tid)}/${memberId}/roles`);
  return r.data?.data?.roles ?? [];
}
export async function grantOrgMemberRole(tid: string, memberId: string, code: string): Promise<void> {
  await client.post(`${mbase(tid)}/${memberId}/roles`, { code });
}
export async function revokeOrgMemberRole(tid: string, memberId: string, roleId: string): Promise<void> {
  await client.delete(`${mbase(tid)}/${memberId}/roles/${roleId}`);
}
export async function listOrgRoles(tid: string): Promise<RoleRow[]> {
  const r = await client.get(`/platform/orgs/${tid}/roles`);
  return r.data?.data?.roles ?? [];
}
export async function listOrgPosts(tid: string): Promise<PostRow[]> {
  const r = await client.get(`/platform/orgs/${tid}/posts`);
  return r.data?.data?.posts ?? [];
}
export async function downloadOrgMembersCsv(tid: string): Promise<void> {
  const res = await client.get(`${mbase(tid)}/export`, { responseType: "blob" });
  const url = URL.createObjectURL(res.data as Blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = "members.csv";
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}

// URL helpers for ImportDialog (it owns the multipart POST + template GET).
export const orgMemberTemplateUrl = (tid: string) => `${mbase(tid)}/template`;
export const orgMemberImportUrl = (tid: string) => `${mbase(tid)}/import`;

// Build a MemberApi adapter for a specific org, ready to bind to <MemberManager>.
// Reset-password routes through the platform-user endpoint (keyed by
// platform_user_id) since members are platform users; status toggle has no
// per-org route, so it is omitted (the button hides) — disable/enable a user
// from the 全局用户 list instead.
export function orgMemberApi(tid: string): MemberApi {
  return {
    listMembers: (p) => listOrgMembers(tid, p),
    fetchDeptTree: () => getOrgDeptTree(tid) as Promise<MemberDeptTreeRow[]>,
    fetchDeptFlat: () => listOrgDepts(tid) as Promise<MemberDeptRow[]>,
    setDept: (memberId, deptId) => setOrgMemberDept(tid, memberId, deptId),
    assignPost: (memberId, postId) => assignOrgMemberPost(tid, memberId, postId),
    removePost: (memberId, postId) => removeOrgMemberPost(tid, memberId, postId),
    listRoles: () => listOrgRoles(tid),
    memberRoles: (m: MemberRow) => getOrgMemberRoles(tid, m.member_id),
    grantRole: (m: MemberRow, code) => grantOrgMemberRole(tid, m.member_id, code),
    revokeRole: (m: MemberRow, roleId) => revokeOrgMemberRole(tid, m.member_id, roleId),
    listPosts: () => listOrgPosts(tid),
    exportCsv: () => downloadOrgMembersCsv(tid),
    resetPassword: (m: MemberRow, pw) => resetPlatformUserPassword(m.platform_user_id, pw),
  };
}
