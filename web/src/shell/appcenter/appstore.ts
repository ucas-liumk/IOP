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
  color: string;         // CSS color like "var(--cat-goal)"
  category: string;      // "协同办公" | "业务管理" | …
  version: string;
  permissions: Permission[];
  events: string[];
  menus?: MenuNode[];
}
export interface CatalogEntry extends Manifest { installed: boolean }
export interface AppCategory { code: string; name: string; color: string; order: number }

export async function getCatalog(): Promise<CatalogEntry[]> {
  const r = await client.get("/apps/catalog");
  return r.data?.data?.apps ?? [];
}

export async function getAppCategories(): Promise<AppCategory[]> {
  const r = await client.get("/apps/categories");
  return r.data?.data?.categories ?? [];
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

export async function updateAppCategory(code: string, category: string): Promise<Manifest> {
  const r = await client.patch(`/admin/apps/${code}/category`, { category });
  return r.data?.data?.app;
}

export async function getPlatformCatalog(tenantId: string): Promise<CatalogEntry[]> {
  const r = await client.get("/platform/apps/catalog", { params: { tenant_id: tenantId } });
  return r.data?.data?.apps ?? [];
}

export async function getPlatformAppCategories(): Promise<AppCategory[]> {
  const r = await client.get("/platform/apps/categories");
  return r.data?.data?.categories ?? [];
}

export async function installPlatformApp(tenantId: string, code: string) {
  await client.post(`/platform/apps/${code}/install`, { tenant_id: tenantId });
}

export async function uninstallPlatformApp(tenantId: string, code: string) {
  await client.delete(`/platform/apps/${code}/install`, { params: { tenant_id: tenantId } });
}

export async function updatePlatformAppCategory(tenantId: string, code: string, category: string): Promise<Manifest> {
  const r = await client.patch(`/platform/apps/${code}/category`, { tenant_id: tenantId, category });
  return r.data?.data?.app;
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
// Driven by each module's manifest.ts (homeRoute, falling back to routePrefix)
// via the SAME glob the router uses to mount module routes, so the launcher
// target always resolves to a real route and adding a module needs no edit here.
const moduleManifests = import.meta.glob<{ manifest: { code: string; routePrefix?: string; homeRoute?: string } }>(
  "@/modules/*/manifest.ts",
  { eager: true },
);
const HOME_BY_CODE: Record<string, string> = {};
for (const path in moduleManifests) {
  const m = moduleManifests[path]?.manifest;
  if (!m?.code) continue;
  HOME_BY_CODE[m.code] = m.homeRoute || m.routePrefix || `/${m.code}`;
}
export function appHomeRoute(code: string): string {
  return HOME_BY_CODE[code] ?? `/${code}`;
}
