// AppStore client — talks to /apps/catalog + /me/apps + /admin/apps/:code/install
import { client } from "@/api/client";

export interface Permission { resource: string; action: string; label: string }
export interface Manifest {
  code: string;
  name: string;
  description: string;
  icon: string;          // SVG path (single d= string from manifest)
  color: string;         // CSS color like "var(--cat-collab)"
  category: string;      // "协同办公" | "业务管理" | …
  version: string;
  permissions: Permission[];
  events: string[];
}
export interface CatalogEntry extends Manifest { installed: boolean }

export async function getCatalog(): Promise<CatalogEntry[]> {
  const r = await client.get("/apps/catalog");
  return r.data?.data?.apps ?? [];
}

export async function getMyApps(): Promise<Manifest[]> {
  const r = await client.get("/me/apps");
  return r.data?.data?.apps ?? [];
}

export async function installApp(code: string) {
  await client.post(`/admin/apps/${code}/install`);
}

export async function uninstallApp(code: string) {
  await client.delete(`/admin/apps/${code}/install`);
}

// Permissions registry — used by RolesView dropdowns
export interface PermissionRegistry {
  permissions: Permission[];
  by_resource: Record<string, Permission[]>;
}
export async function getPermissionRegistry(): Promise<PermissionRegistry> {
  const r = await client.get("/admin/permissions");
  return r.data?.data ?? { permissions: [], by_resource: {} };
}

// Per-app default frontend route — for "open this app" from rail / appcenter.
// Modules that need different routing can override via manifest.config later.
const ROUTE_BY_CODE: Record<string, string> = {
  okr: "/okr/plans",
  approval: "/approval",
  crm: "/crm",
};
export function appHomeRoute(code: string): string {
  return ROUTE_BY_CODE[code] ?? `/apps/${code}`;
}
