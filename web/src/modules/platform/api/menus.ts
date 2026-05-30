import { client } from "@/api/client";
import type { MenuNode } from "@/shell/appcenter/appstore";

// MenuTreeNode = a server MenuNode with its assembled children (the catalog
// endpoints return the tree already nested by parent).
export interface MenuTreeNode extends MenuNode {
  children?: MenuTreeNode[];
}

// getPlatformMenuTree returns the COMPLETE platform-console menu/permission tree
// (unfiltered) — used by the platform role editor (checkbox tree) and the menu
// catalog (governance/inspection) view. GET /platform/menus.
export async function getPlatformMenuTree(): Promise<MenuTreeNode[]> {
  const r = await client.get("/platform/menus");
  return (r.data?.data?.menus ?? []) as MenuTreeNode[];
}

// getTenantMenuTree returns the COMPLETE tenant-console menu tree. This endpoint
// (GET /admin/menus) is tenant-scoped (TenantAdminRequired) — a platform-only
// admin without a tenant context will get 403, so callers must tolerate failure.
export async function getTenantMenuTree(): Promise<MenuTreeNode[]> {
  const r = await client.get("/admin/menus");
  return (r.data?.data?.menus ?? []) as MenuTreeNode[];
}
