import { client } from "@/api/client";
import type { MenuNode } from "@/shell/appcenter/appstore";

// MenuTreeNode = a server MenuNode with its assembled children (the catalog
// endpoints return the tree already nested by parent).
export interface MenuTreeNode extends MenuNode {
  children?: MenuTreeNode[];
}

export interface MenuPayload {
  key: string;
  title: string;
  type: string;
  parent?: string;
  path?: string;
  component?: string;
  permission_code?: string;
  perm?: string;
  icon?: string;
  order?: number;
  visible?: boolean;
  cacheable?: boolean;
  status?: string;
  app_code?: string;
  app?: string;
  console?: string;
  external_url?: string;
  iframe_url?: string;
  micro_app_code?: string;
  micro_entry?: string;
}

// getPlatformMenuTree returns the COMPLETE platform-console menu/permission tree
// (unfiltered) — used by the platform role editor (checkbox tree) and the menu
// catalog (governance/inspection) view. GET /platform/menus.
export async function getPlatformMenuTree(): Promise<MenuTreeNode[]> {
  const r = await client.get("/platform/menus");
  return (r.data?.data?.menus ?? []) as MenuTreeNode[];
}

export async function createPlatformMenu(payload: MenuPayload): Promise<MenuTreeNode> {
  const r = await client.post("/platform/menus", payload);
  return r.data?.data as MenuTreeNode;
}
export async function updatePlatformMenu(key: string, payload: Partial<MenuPayload>): Promise<void> {
  await client.patch(`/platform/menus/${encodeURIComponent(key)}`, payload);
}
export async function deletePlatformMenu(key: string): Promise<void> {
  await client.delete(`/platform/menus/${encodeURIComponent(key)}`);
}
export async function batchPlatformMenus(menus: MenuPayload[]): Promise<void> {
  await client.post("/platform/menus/batch", { menus });
}

// getTenantMenuTree returns the COMPLETE tenant-console menu tree. This endpoint
// (GET /admin/menus) is tenant-scoped (TenantAdminRequired) — a platform-only
// admin without a tenant context will get 403, so callers must tolerate failure.
export async function getTenantMenuTree(): Promise<MenuTreeNode[]> {
  const r = await client.get("/admin/menus");
  return (r.data?.data?.menus ?? []) as MenuTreeNode[];
}

export async function getTenantMenuConfigTree(): Promise<MenuTreeNode[]> {
  const r = await client.get("/admin/menus/config");
  return (r.data?.data?.menus ?? []) as MenuTreeNode[];
}

export async function updateTenantMenuConfig(key: string, payload: { enabled: boolean; order?: number }): Promise<void> {
  await client.patch(`/admin/menus/${encodeURIComponent(key)}/config`, payload);
}
