import { createRouter, createWebHistory, type RouteRecordRaw } from "vue-router";
import AppLayout from "@/shell/layout/AppLayout.vue";
import AdminLayout from "@/modules/admin/AdminLayout.vue";
import { requireAuth } from "@/shell/auth/guard";

// === Auto-discover business modules ===
// Each module under @/modules/<code>/ exposes a manifest.ts + routes.ts.
// Adding a new module = drop a folder; no router edits needed.
//
// "admin" module is hand-wired because it uses its own AdminLayout.
const moduleManifests = import.meta.glob<{ manifest: { code: string; routePrefix: string } }>(
  "@/modules/*/manifest.ts", { eager: true }
);
const moduleRoutes = import.meta.glob<{ routes: RouteRecordRaw[] }>(
  "@/modules/*/routes.ts", { eager: true }
);

const businessRoutes: RouteRecordRaw[] = [];
for (const path in moduleManifests) {
  const m = moduleManifests[path].manifest;
  if (m.code === "admin") continue;
  const folder = path.replace("/manifest.ts", "");
  const routesEntry = moduleRoutes[`${folder}/routes.ts`];
  if (!routesEntry) continue;
  for (const r of routesEntry.routes) {
    businessRoutes.push({
      ...r,
      path: m.routePrefix.replace(/^\//, "") + (r.path ? "/" + r.path : ""),
    });
  }
}

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: "/login",
      name: "login",
      component: () => import("@/shell/auth/LoginView.vue"),
    },
    {
      path: "/",
      component: AppLayout,
      beforeEnter: requireAuth,
      children: [
        {
          path: "",
          name: "workspace",
          component: () => import("@/shell/workspace/WorkspaceHome.vue"),
        },

        // Auto-mounted business module routes
        ...businessRoutes,

        // Admin module
        {
          path: "admin",
          component: AdminLayout,
          children: [
            { path: "",            name: "admin.home",     component: () => import("@/modules/admin/views/AdminHome.vue") },
            { path: "members",     name: "admin.members",  component: () => import("@/modules/admin/views/MembersView.vue") },
            { path: "roles",       name: "admin.roles",    component: () => import("@/modules/admin/views/RolesView.vue") },
            { path: "departments", name: "admin.depts",    component: () => import("@/modules/admin/views/DepartmentsView.vue") },
            { path: "settings",    name: "admin.settings", component: () => import("@/modules/admin/views/SettingsView.vue") },
            { path: "audit",       name: "admin.audit",    component: () => import("@/modules/admin/views/AuditView.vue") },
            { path: "dict",        name: "admin.dict",     component: () => import("@/modules/admin/views/DictView.vue") },
            { path: "platform/tenants", name: "admin.platform.tenants", component: () => import("@/modules/admin/views/PlatformTenantsView.vue") },
          ],
        },

        // Personal settings (uses AdminLayout for unified sidebar)
        {
          path: "me",
          component: AdminLayout,
          children: [
            { path: "settings", name: "me.settings", component: () => import("@/modules/admin/views/MeSettingsView.vue") },
          ],
        },
      ],
    },
  ],
});

if (import.meta.env.DEV) {
  console.info("[router] auto-mounted business modules:", businessRoutes.map((r) => r.path));
}

export default router;
