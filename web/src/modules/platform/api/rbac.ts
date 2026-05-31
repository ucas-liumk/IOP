import { client } from "@/api/client";

export interface PlatformPermission {
  resource: string; action: string; domain: string; label: string; is_high_risk: boolean;
}
export interface PolicyRule { resource: string; action: string; effect: string }
export type RoleStatus = "active" | "disabled";
export type RoleType = "platform" | "tenant";
export interface PlatformRole {
  id: string; code: string; name: string; built_in: boolean;
  tenant_id?: string;
  role_type?: RoleType;
  status: RoleStatus;
  order_num: number;
  remark?: string;
  data_scope?: string;
  member_count: number; policies: PolicyRule[] | null; members: string[];
}
export interface RoleSummary {
  id: string; tenant_id?: string; role_type: RoleType; code: string; name: string;
  status: RoleStatus; order_num: number; remark?: string; built_in: boolean;
  data_scope: string; member_count: number; policies: PolicyRule[] | null;
}
export interface RbacMe {
  roles: string[]; permissions: string[]; is_super_admin: boolean;
}

export async function getRbacMe(): Promise<RbacMe> {
  const r = await client.get("/platform/rbac/me");
  return r.data?.data ?? { roles: [], permissions: [], is_super_admin: false };
}
export async function listPlatformPermissions(): Promise<PlatformPermission[]> {
  const r = await client.get("/platform/rbac/permissions");
  return r.data?.data?.permissions ?? [];
}
export async function listPlatformRoles(params?: { q?: string; status?: string }): Promise<PlatformRole[]> {
  const r = await client.get("/platform/rbac/roles", { params });
  return r.data?.data?.roles ?? [];
}
export async function listAllRoles(params?: {
  q?: string; status?: string; role_type?: string; tenant_id?: string;
}): Promise<RoleSummary[]> {
  const r = await client.get("/platform/rbac/roles/all", { params });
  return r.data?.data?.roles ?? [];
}
export async function createPlatformRole(payload: {
  code: string; name: string; status?: RoleStatus; order_num?: number; remark?: string;
}): Promise<PlatformRole> {
  const r = await client.post("/platform/rbac/roles", payload);
  return r.data?.data as PlatformRole;
}
export async function updatePlatformRole(id: string, patch: {
  code?: string; name?: string; status?: RoleStatus; order_num?: number; remark?: string;
}) {
  await client.patch(`/platform/rbac/roles/${id}`, patch);
}
export async function deletePlatformRole(id: string) {
  await client.delete(`/platform/rbac/roles/${id}`);
}
export async function addPlatformPolicy(id: string, resource: string, action: string) {
  await client.post(`/platform/rbac/roles/${id}/policies`, { resource, action });
}
export async function removePlatformPolicy(id: string, resource: string, action: string) {
  await client.delete(`/platform/rbac/roles/${id}/policies`, { params: { resource, action } });
}
export interface PolicyChange { resource: string; action: string }
// Atomically add/remove a set of policies on a platform role in one request.
export async function platformBatchPolicy(
  id: string,
  changes: { add?: PolicyChange[]; remove?: PolicyChange[] },
) {
  await client.post(`/platform/rbac/roles/${id}/policies/batch`, {
    add: changes.add ?? [],
    remove: changes.remove ?? [],
  });
}
export async function grantPlatformRole(id: string, platform_user_id: string) {
  await client.post(`/platform/rbac/roles/${id}/members`, { platform_user_id });
}
export async function revokePlatformRole(id: string, uid: string) {
  await client.delete(`/platform/rbac/roles/${id}/members/${uid}`);
}
