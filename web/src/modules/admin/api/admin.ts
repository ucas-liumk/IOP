import { client } from "@/api/client";
import type {
  MemberApi, MemberRow, MemberListParams, MemberPage,
  MemberDeptRow, MemberDeptTreeRow,
} from "@/shell/components";

// === Shared paging / bulk shapes ===
export interface Paged<T> {
  data: T[];
  total: number;
  page: number;
  pageSize: number;
}
export interface BulkRowError { row: number; key?: string; message: string }
export interface BulkResult { total: number; succeeded: number; failed: number; errors: BulkRowError[] }
export interface PolicyChange { resource: string; action: string }

// Triggers a browser download for a blob fetched from `path` (responseType
// blob). Falls back to a sensible default filename when none is provided.
async function downloadFile(path: string, filename: string, params?: Record<string, string | number | boolean | undefined>): Promise<void> {
  const res = await client.get(path, { responseType: "blob", params });
  const url = URL.createObjectURL(res.data as Blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}

// POSTs a single file as multipart/form-data (field "file") and returns the
// BulkResult envelope.
async function uploadCsv(path: string, file: File): Promise<BulkResult> {
  const fd = new FormData();
  fd.append("file", file);
  const r = await client.post(path, fd);
  return (r.data?.data ?? r.data) as BulkResult;
}

export interface MeAdmin {
  is_tenant_admin: boolean;
  is_platform_admin: boolean;
  has_platform_access: boolean;
}
export async function getMyAdminFlags(): Promise<MeAdmin> {
  try {
    const r = await client.get("/me/admin");
    const d = r.data?.data ?? {};
    return {
      is_tenant_admin: !!d.is_tenant_admin,
      is_platform_admin: !!d.is_platform_admin,
      has_platform_access: !!(d.has_platform_access ?? d.is_platform_admin),
    };
  } catch { return { is_tenant_admin: false, is_platform_admin: false, has_platform_access: false }; }
}

export interface MemberPost { post_id: string; code: string; name: string }
export interface Member {
  member_id: string; platform_user_id: string;
  // username is the primary login identity (joined from platform_user); email is
  // optional and often NULL — never use it as the primary display identifier.
  username: string;
  display_name: string; email?: string;
  department: string; dept_id?: string | null; title: string; phone: string;
  status: string; joined_at: string;
  posts: MemberPost[];
}
export interface ListMembersParams {
  search?: string;
  dept_id?: string;
  subtree?: boolean;
}
export async function listMembers(params: ListMembersParams | string = {}): Promise<Member[]> {
  // Back-compat: a bare string is treated as the search term.
  const p: ListMembersParams = typeof params === "string" ? { search: params } : params;
  const query: Record<string, string> = {};
  if (p.search) query.search = p.search;
  if (p.dept_id) query.dept_id = p.dept_id;
  if (p.subtree) query.subtree = "true";
  const r = await client.get("/admin/members", { params: query });
  return r.data?.data?.members ?? [];
}

export interface ListMembersPagedParams {
  page?: number;
  pageSize?: number;
  search?: string;
  deptId?: string | null;
  subtree?: boolean;
}
// Server-side paginated member listing. Returns the typed page envelope; the
// server caps page_size at 100.
export async function listMembersPaged(p: ListMembersPagedParams = {}): Promise<Paged<Member>> {
  const query: Record<string, string | number> = {
    page: p.page ?? 1,
    page_size: p.pageSize ?? 20,
  };
  if (p.search) query.search = p.search;
  if (p.deptId) query.dept_id = p.deptId;
  if (p.subtree) query.subtree = "true";
  const r = await client.get("/admin/members", { params: query });
  const d = r.data?.data ?? {};
  return {
    data: d.data ?? d.members ?? [],
    total: d.total ?? 0,
    page: d.page ?? query.page as number,
    pageSize: d.page_size ?? query.page_size as number,
  };
}

// Member CSV export / template download + import (member:write gated server-side).
export function downloadMembersCsv(): Promise<void> {
  return downloadFile("/admin/members/export", "members.csv");
}
export function downloadMembersTemplate(): Promise<void> {
  return downloadFile("/admin/members/template", "members_template.csv");
}
export function importMembers(file: File): Promise<BulkResult> {
  return uploadCsv("/admin/members/import", file);
}
// updateMember: dept_id is tri-state on the wire — omit to leave unchanged,
// "" / null to clear, an id to assign. Pass `dept_id` key only when changing it.
export async function updateMember(
  id: string,
  patch: Partial<Pick<Member, "display_name" | "department" | "title" | "phone">> & { dept_id?: string | null },
) {
  await client.patch(`/admin/members/${id}`, patch);
}
export async function setMemberDept(id: string, deptId: string | null) {
  await client.patch(`/admin/members/${id}`, { dept_id: deptId ?? "" });
}
export async function setMemberDisabled(id: string, disabled: boolean) {
  await client.post(`/admin/members/${id}/${disabled ? 'disable' : 'enable'}`);
}
export async function assignMemberPost(memberId: string, postId: string) {
  await client.post(`/admin/members/${memberId}/posts`, { post_id: postId });
}
export async function removeMemberPost(memberId: string, postId: string) {
  await client.delete(`/admin/members/${memberId}/posts/${postId}`);
}

// === Departments (部门) ===
export interface Dept {
  id: string;
  tenant_id: string;
  name: string;
  org_code: string;
  parent_id?: string | null;
  org_type: string;
  order_num: number;
  leader?: string;
  leader_account?: string;
  phone?: string;
  email?: string;
  status: string;
  remark?: string;
  path?: string;
  is_root?: boolean;
  created_at: string;
}
export interface DeptTreeNode extends Dept {
  children?: DeptTreeNode[];
}
export interface DeptQuery {
  search?: string;
  status?: string;
  [key: string]: string | undefined;
}
export async function listDepts(params: DeptQuery = {}): Promise<Dept[]> {
  const r = await client.get("/admin/depts", { params });
  return r.data?.data?.depts ?? [];
}
export async function getDeptTree(params: DeptQuery = {}): Promise<DeptTreeNode[]> {
  const r = await client.get("/admin/depts/tree", { params });
  return r.data?.data?.tree ?? [];
}
export async function createDept(payload: {
  name: string; org_code: string; parent_id?: string | null; org_type?: string; order_num?: number;
  leader?: string; leader_account?: string; phone?: string; email?: string; status?: string; remark?: string;
}): Promise<Dept> {
  const r = await client.post("/admin/depts", payload);
  return r.data?.data as Dept;
}
export async function updateDept(id: string, patch: Partial<Pick<Dept, "name" | "org_code" | "parent_id" | "org_type" | "order_num" | "leader" | "leader_account" | "phone" | "email" | "status" | "remark">>) {
  await client.patch(`/admin/depts/${id}`, patch);
}
export async function setDeptStatus(id: string, status: string, cascade = false) {
  await client.post(`/admin/depts/${id}/status`, { status, cascade });
}
export async function deleteDept(id: string) {
  await client.delete(`/admin/depts/${id}`);
}
export async function moveDept(id: string, parentId: string | null) {
  await client.post(`/admin/depts/${id}/move`, { parent_id: parentId ?? "" });
}

// Department spreadsheet export / template download + import (dept:write gated server-side).
export function downloadDeptsCsv(params: DeptQuery = {}): Promise<void> {
  return downloadFile("/admin/depts/export", "departments.xlsx", params);
}
export function downloadDeptsTemplate(): Promise<void> {
  return downloadFile("/admin/depts/template", "departments_template.xlsx");
}
export function importDepts(file: File): Promise<BulkResult> {
  return uploadCsv("/admin/depts/import", file);
}

// === Posts (岗位) ===
export interface Post {
  id: string;
  code: string;
  name: string;
  order_num: number;
  status: string;
  created_at: string;
}
export async function listPosts(): Promise<Post[]> {
  const r = await client.get("/admin/posts");
  return r.data?.data?.posts ?? [];
}
export async function createPost(payload: { code: string; name: string; order_num?: number }): Promise<Post> {
  const r = await client.post("/admin/posts", payload);
  return r.data?.data as Post;
}
export async function updatePost(id: string, patch: Partial<Pick<Post, "name" | "order_num" | "status">>) {
  await client.patch(`/admin/posts/${id}`, patch);
}
export async function deletePost(id: string) {
  await client.delete(`/admin/posts/${id}`);
}

// === Menu catalog (角色编辑器勾选用) ===
export interface MenuNode {
  key: string;
  title: string;
  icon: string;
  path: string;
  parent: string;
  type: string;     // "dir" | "menu" | "button"
  console: string;
  app: string;
  perm: string;     // "resource:action"; "" = no perm required
  order: number;
  children?: MenuNode[];
}
export async function getTenantMenuTree(): Promise<MenuNode[]> {
  const r = await client.get("/admin/menus");
  return r.data?.data?.menus ?? [];
}

export interface PolicyRule { resource: string; action: string; effect: string }
export type DataScope = "all" | "dept" | "dept_and_sub" | "self" | "custom";
export interface Role {
  id: string; tenant_id?: string; code: string; name: string;
  built_in: boolean; member_count: number;
  data_scope: DataScope;
  dept_ids?: string[];
  policies: PolicyRule[] | null;
}
export async function listRoles(): Promise<Role[]> {
  const r = await client.get("/admin/roles");
  return r.data?.data?.roles ?? [];
}
export async function createRole(payload: {
  code: string; name: string; data_scope?: DataScope; dept_ids?: string[];
}): Promise<Role> {
  const r = await client.post("/admin/roles", payload);
  return r.data?.data as Role;
}
// updateRole patches name / data_scope / dept_ids (code is immutable for builtin
// roles server-side). Omit a field to leave it unchanged.
export async function updateRole(id: string, patch: {
  name?: string; code?: string; data_scope?: DataScope; dept_ids?: string[];
}) {
  await client.patch(`/admin/roles/${id}`, patch);
}
export async function deleteRole(id: string) { await client.delete(`/admin/roles/${id}`); }
export async function addPolicy(roleId: string, resource: string, action: string) {
  await client.post(`/admin/roles/${roleId}/policies`, { resource, action });
}
export async function removePolicy(roleId: string, resource: string, action: string) {
  await client.delete(`/admin/roles/${roleId}/policies`, { params: { resource, action } });
}
// Atomically add/remove a set of policies on a tenant role in one request.
export async function batchPolicy(
  roleId: string,
  changes: { add?: PolicyChange[]; remove?: PolicyChange[] },
) {
  await client.post(`/admin/roles/${roleId}/policies/batch`, {
    add: changes.add ?? [],
    remove: changes.remove ?? [],
  });
}
export async function grantRoleToMember(memberId: string, code: string) {
  await client.post(`/admin/members/${memberId}/roles`, { code });
}
export async function revokeRoleFromMember(memberId: string, roleId: string) {
  await client.delete(`/admin/members/${memberId}/roles/${roleId}`);
}
export async function getMemberRoles(memberId: string): Promise<Role[]> {
  const r = await client.get(`/admin/members/${memberId}/roles`);
  return r.data?.data?.roles ?? [];
}

export interface AuditEntry {
  id: string; occurred_at: string; actor: string;
  action: string; resource: string; resource_id: string;
  trace_id: string; detail: any;
}
export async function listAudit(): Promise<AuditEntry[]> {
  const r = await client.get("/audit/logs");
  return r.data?.data?.entries ?? [];
}

export interface TenantInfo {
  tenant: {
    id: string; slug: string; name: string;
    schema_name: string; status: string; created_at: string;
  };
  member_count: number;
}
export async function getTenant(): Promise<TenantInfo> {
  const r = await client.get("/admin/tenant");
  return r.data?.data;
}
export async function updateTenantName(name: string) {
  await client.patch("/admin/tenant", { name });
}

// Platform admin: list all tenants
export interface PlatformTenant {
  id: string; slug: string; name: string;
  schema_name: string; status: string; created_at: string;
}
export async function listAllTenants(): Promise<PlatformTenant[]> {
  const r = await client.get("/tenants");
  return r.data?.data?.tenants ?? [];
}
export async function suspendTenant(id: string, reason = "") {
  await client.post(`/tenants/${id}/suspend`, null, { params: reason ? { reason } : {} });
}
export async function resumeTenant(id: string) {
  await client.post(`/tenants/${id}/resume`);
}

// Dictionary admin
export interface DictTypeItems {
  type_code: string;
  items: Array<{ type_code: string; code: string; name: string; sort_order: number; active: boolean }>;
  overrides: Record<string, { name?: string; sort_order?: number; active?: boolean }>;
}
export async function listDictTypes(): Promise<string[]> {
  const r = await client.get("/admin/dict/types");
  return r.data?.data?.types ?? [];
}
export async function getDictType(typeCode: string): Promise<DictTypeItems> {
  const r = await client.get(`/admin/dict/${typeCode}/items`);
  return r.data?.data;
}
export async function setDictOverride(typeCode: string, code: string, override: { name: string; sort_order: number; active: boolean }) {
  await client.put(`/admin/dict/${typeCode}/items/${code}/override`, override);
}
export async function clearDictOverride(typeCode: string, code: string) {
  await client.delete(`/admin/dict/${typeCode}/items/${code}/override`);
}

// === Notices (通知公告) ===
export interface Notice {
  id: string;
  title: string;
  content: string;
  type: string;       // "notice" | "announcement" | ...
  status: string;     // "draft" | "published"
  created_by?: string | null;
  created_at: string;
}
export type NoticeStatus = "draft" | "published" | "";
export async function listNotices(status: NoticeStatus = "", page = 1, pageSize = 50): Promise<Notice[]> {
  const params: Record<string, string | number> = { page, page_size: pageSize };
  if (status) params.status = status;
  const r = await client.get("/admin/notices", { params });
  return r.data?.data?.notices ?? [];
}
export async function createNotice(payload: { title: string; content?: string; type?: string }): Promise<Notice> {
  const r = await client.post("/admin/notices", payload);
  return r.data?.data as Notice;
}
export async function updateNotice(id: string, patch: { title?: string; content?: string; type?: string }) {
  await client.patch(`/admin/notices/${id}`, patch);
}
export async function deleteNotice(id: string) {
  await client.delete(`/admin/notices/${id}`);
}
export async function publishNotice(id: string) {
  await client.post(`/admin/notices/${id}/publish`);
}
export async function withdrawNotice(id: string) {
  await client.post(`/admin/notices/${id}/withdraw`);
}

// === Logs (操作日志 / 登录日志) ===
// Both endpoints return the tenant audit_log shape (AuditEntry). They are paged
// (page / page_size) and return a bare array — there is no total count, so the
// UI infers "has next page" from a full page.
export interface LogFilter {
  actor?: string;
  action?: string;
  from?: string;   // YYYY-MM-DD
  to?: string;     // YYYY-MM-DD
  page?: number;
  pageSize?: number;
}
function logParams(f: LogFilter): Record<string, string | number> {
  const p: Record<string, string | number> = {
    page: f.page ?? 1,
    page_size: f.pageSize ?? 20,
  };
  if (f.actor) p.actor = f.actor;
  if (f.action) p.action = f.action;
  if (f.from) p.from = f.from;
  if (f.to) p.to = f.to;
  return p;
}
export async function listOperLogs(f: LogFilter = {}): Promise<AuditEntry[]> {
  const r = await client.get("/admin/operlogs", { params: logParams(f) });
  return r.data?.data?.entries ?? [];
}
export async function listLoginLogs(f: LogFilter = {}): Promise<AuditEntry[]> {
  // The login-log endpoint ignores `action` (it's pinned to login topics).
  const { action, ...rest } = f;
  void action;
  const r = await client.get("/admin/loginlogs", { params: logParams(rest) });
  return r.data?.data?.entries ?? [];
}

// === Online users (在线用户) ===
export interface OnlineSession {
  session_id: string;
  member_id?: string;
  display_name: string;
  ip_address?: string;
  issued_at: string;
  expires_at: string;
}
export async function listOnlineSessions(): Promise<OnlineSession[]> {
  const r = await client.get("/admin/online");
  return r.data?.data?.sessions ?? [];
}
export async function kickSession(sid: string) {
  await client.post(`/admin/online/${sid}/kick`);
}

// Personal
export async function changePassword(oldPw: string, newPw: string) {
  await client.post("/me/password", { old: oldPw, new: newPw });
}
export interface Session {
  id: string; issued_at: string; expires_at: string;
  ip_address: string; user_agent: string;
  current: boolean; revoked: boolean;
}
export async function listSessions(): Promise<Session[]> {
  const r = await client.get("/me/sessions");
  return r.data?.data?.sessions ?? [];
}
export async function revokeSession(id: string) {
  await client.post(`/me/sessions/${id}/revoke`);
}

// Platform users (cross-tenant) — platform_admin only
export interface PlatformUser {
  id: string;
  username?: string;
  phone?: string;
  email?: string;
  status: string;
  last_login_at?: string;
  created_at: string;
}
export async function listPlatformUsers(search = ""): Promise<PlatformUser[]> {
  const r = await client.get("/platform/users", { params: search ? { search } : {} });
  return r.data?.data?.users ?? [];
}
export interface ListPlatformUsersPagedParams {
  page?: number;
  pageSize?: number;
  search?: string;
}
// Server-side paginated platform-user listing (platform admin only).
export async function listPlatformUsersPaged(p: ListPlatformUsersPagedParams = {}): Promise<Paged<PlatformUser>> {
  const query: Record<string, string | number> = {
    page: p.page ?? 1,
    page_size: p.pageSize ?? 20,
  };
  if (p.search) query.search = p.search;
  const r = await client.get("/platform/users", { params: query });
  const d = r.data?.data ?? {};
  return {
    data: d.data ?? d.users ?? [],
    total: d.total ?? 0,
    page: d.page ?? query.page as number,
    pageSize: d.page_size ?? query.page_size as number,
  };
}
export async function createPlatformUser(payload: {
  username: string; real_name: string; phone?: string; password: string;
  organization_id: string; role: "tenant_member" | "tenant_admin";
}): Promise<PlatformUser> {
  const r = await client.post("/platform/users", payload);
  return r.data?.data;
}
export async function disablePlatformUser(id: string) {
  await client.post(`/platform/users/${id}/disable`);
}
export async function enablePlatformUser(id: string) {
  await client.post(`/platform/users/${id}/enable`);
}
export async function resetPlatformUserPassword(id: string, newPassword: string) {
  await client.post(`/platform/users/${id}/reset-password`, { new_password: newPassword });
}

// Platform overview stats (platform admin)
export interface PlatformStats { organizations: number; users: number; pending_registrations: number }
export async function getPlatformStats(): Promise<PlatformStats> {
  const r = await client.get("/platform/stats");
  return r.data?.data ?? { organizations: 0, users: 0, pending_registrations: 0 };
}

// Registration applications
export interface RegistrationApplication {
  id: string;
  username: string;
  real_name: string;
  organization: string;
  phone?: string;
  status: "pending" | "approved" | "rejected";
  applied_at: string;
  reviewed_by?: string;
  reviewed_at?: string;
  reject_reason?: string;
  target_tenant_id?: string;
  granted_role?: string;
}
// Registration review is available in two scopes:
//   - "platform" → /platform/registrations (ALL tenants; platform admin)
//   - "tenant"   → /admin/registrations    (own tenant only; tenant admin)
export type RegScope = "platform" | "tenant";
function regBase(scope: RegScope): string {
  return scope === "platform" ? "/platform/registrations" : "/admin/registrations";
}
export async function listRegistrations(scope: RegScope, status: "pending" | "approved" | "rejected" | "all" = "pending"): Promise<RegistrationApplication[]> {
  const r = await client.get(regBase(scope), { params: { status } });
  return r.data?.data?.applications ?? [];
}
export async function approveRegistration(scope: RegScope, id: string, role: "tenant_member" | "tenant_admin") {
  await client.post(`${regBase(scope)}/${id}/approve`, { role });
}
export async function rejectRegistration(scope: RegScope, id: string, reason: string) {
  await client.post(`${regBase(scope)}/${id}/reject`, { reason });
}

// === Tenant-console MemberApi adapter (binds to <MemberManager>) ===
// Wraps the existing /admin/members* funcs into the MemberApi shape the shared
// MemberManager consumes. Reset-password routes through the platform-user
// endpoint (members are platform users), keyed by platform_user_id.
export function tenantMemberApi(): MemberApi {
  return {
    listMembers: async (p: MemberListParams): Promise<MemberPage> => {
      const res = await listMembersPaged({
        page: p.page, pageSize: p.pageSize, search: p.search,
        deptId: p.deptId, subtree: p.subtree,
      });
      return { data: res.data as unknown as MemberRow[], total: res.total, page: res.page, pageSize: res.pageSize };
    },
    fetchDeptTree: () => getDeptTree() as unknown as Promise<MemberDeptTreeRow[]>,
    fetchDeptFlat: () => listDepts() as unknown as Promise<MemberDeptRow[]>,
    setDept: (memberId, deptId) => setMemberDept(memberId, deptId),
    assignPost: (memberId, postId) => assignMemberPost(memberId, postId),
    removePost: (memberId, postId) => removeMemberPost(memberId, postId),
    listRoles: () => listRoles(),
    memberRoles: (m: MemberRow) => getMemberRoles(m.member_id),
    grantRole: (m: MemberRow, code) => grantRoleToMember(m.member_id, code),
    revokeRole: (m: MemberRow, roleId) => revokeRoleFromMember(m.member_id, roleId),
    listPosts: () => listPosts(),
    exportCsv: () => downloadMembersCsv(),
    setDisabled: (m: MemberRow, disabled) => setMemberDisabled(m.member_id, disabled),
    resetPassword: (m: MemberRow, pw) => resetPlatformUserPassword(m.platform_user_id, pw),
  };
}
