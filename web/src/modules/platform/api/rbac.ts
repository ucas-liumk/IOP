import { client } from "@/api/client";

export interface PlatformPermission {
  resource: string; action: string; domain: string; label: string; is_high_risk: boolean;
}
export interface PolicyRule { resource: string; action: string; effect: string }
export interface PlatformRole {
  id: string; code: string; name: string; built_in: boolean;
  member_count: number; policies: PolicyRule[] | null; members: string[];
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
export async function listPlatformRoles(): Promise<PlatformRole[]> {
  const r = await client.get("/platform/rbac/roles");
  return r.data?.data?.roles ?? [];
}
export async function createPlatformRole(code: string, name: string) {
  await client.post("/platform/rbac/roles", { code, name });
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
export async function grantPlatformRole(id: string, platform_user_id: string) {
  await client.post(`/platform/rbac/roles/${id}/members`, { platform_user_id });
}
export async function revokePlatformRole(id: string, uid: string) {
  await client.delete(`/platform/rbac/roles/${id}/members/${uid}`);
}
