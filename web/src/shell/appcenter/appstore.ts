// AppStore client — talks to /apps/catalog + /me/apps + /admin/apps/:code/install
import { client } from "@/api/client";

export interface Permission { resource: string; action: string; label: string }
// MenuNode mirrors server/internal/shared/module.MenuNode — one console nav node.
// Frontend consumption (dynamic nav, role editor tree) is wired in a later layer;
// this is typing only.
export interface MenuNode {
  key: string;
  title: string;
  icon: string;     // SVG path data
  path: string;     // frontend route ("" for dir nodes)
  component?: string;
  parent: string;   // parent key ("" = top level)
  type: string;     // "dir" | "menu" | "button" | "link" | "iframe" | "micro"
  console: string;  // "platform" | "tenant" | "both"
  app: string;      // owning module code ("" for built-in)
  perm: string;     // "resource:action" ("" = login/App only)
  order: number;
  visible?: boolean;
  cacheable?: boolean;
  status?: string;
  built_in?: boolean;
  tenant_enabled?: boolean;
  external_url?: string;
  iframe_url?: string;
  micro_app_code?: string;
  micro_entry?: string;
}
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
  menus?: MenuNode[];
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

// Per-user workspace mutations — any logged-in user curates their own apps.
export async function addMyApp(code: string) {
  await client.post(`/me/apps/${code}`);
}

export async function removeMyApp(code: string) {
  await client.delete(`/me/apps/${code}`);
}

export async function setMyAppOrder(codes: string[]) {
  await client.put("/me/apps/order", { codes });
}

// Tenant-admin install/uninstall — kept for admin surfaces that may reference them.
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
  tasks: "/tasks",
  approval: "/approval",
  crm: "/crm",
};
export function appHomeRoute(code: string): string {
  return ROUTE_BY_CODE[code] ?? `/apps/${code}`;
}
