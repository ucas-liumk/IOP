import { client } from "@/api/client";

export interface MeAdmin { is_tenant_admin: boolean; is_platform_admin: boolean }
export async function getMyAdminFlags(): Promise<MeAdmin> {
  try {
    const r = await client.get("/me/admin");
    return r.data?.data ?? { is_tenant_admin: false, is_platform_admin: false };
  } catch { return { is_tenant_admin: false, is_platform_admin: false }; }
}

export interface Member {
  member_id: string; platform_user_id: string;
  display_name: string; email: string;
  department: string; title: string; phone: string;
  status: string; joined_at: string;
}
export async function listMembers(search = ""): Promise<Member[]> {
  const r = await client.get("/admin/members", { params: search ? { search } : {} });
  return r.data?.data?.members ?? [];
}
export async function updateMember(id: string, patch: Partial<Pick<Member,'display_name'|'department'|'title'|'phone'>>) {
  await client.patch(`/admin/members/${id}`, patch);
}
export async function setMemberDisabled(id: string, disabled: boolean) {
  await client.post(`/admin/members/${id}/${disabled ? 'disable' : 'enable'}`);
}

export interface PolicyRule { resource: string; action: string; effect: string }
export interface Role {
  id: string; tenant_id?: string; code: string; name: string;
  built_in: boolean; member_count: number;
  policies: PolicyRule[] | null;
}
export async function listRoles(): Promise<Role[]> {
  const r = await client.get("/admin/roles");
  return r.data?.data?.roles ?? [];
}
export async function createRole(code: string, name: string) {
  const r = await client.post("/admin/roles", { code, name });
  return r.data?.data as Role;
}
export async function deleteRole(id: string) { await client.delete(`/admin/roles/${id}`); }
export async function addPolicy(roleId: string, resource: string, action: string) {
  await client.post(`/admin/roles/${roleId}/policies`, { resource, action });
}
export async function removePolicy(roleId: string, resource: string, action: string) {
  await client.delete(`/admin/roles/${roleId}/policies`, { params: { resource, action } });
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
