import { createRouter, createWebHistory, type RouteRecordRaw } from "vue-router";
import AppLayout from "@/shell/layout/AppLayout.vue";
import AdminLayout from "@/modules/admin/AdminLayout.vue";
import PlatformLayout from "@/modules/platform/PlatformLayout.vue";
import MeLayout from "@/modules/me/MeLayout.vue";
import { requireAuth, requirePlatformAdmin, requireTenantAdmin } from "@/shell/auth/guard";
import { useAuthStore } from "@/shell/auth/auth.store";
import { bumpRecent } from "@/shell/workspace/widgets/recentApps";

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
      path: "/register",
      name: "register",
      component: () => import("@/shell/auth/RegisterView.vue"),
    },
    {
      path: "/403",
      name: "access-denied",
      component: () => import("@/shell/views/AccessDenied.vue"),
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

        // Tenant console (组织控制台) — tenant_admin only
        {
          path: "admin",
          component: AdminLayout,
          beforeEnter: requireTenantAdmin,
          children: [
            { path: "",              name: "admin.home",          component: () => import("@/modules/admin/views/AdminHome.vue") },
            { path: "members",       name: "admin.members",       component: () => import("@/modules/admin/views/MembersView.vue") },
            { path: "registrations", name: "admin.registrations", component: () => import("@/modules/admin/views/RegistrationsView.vue") },
            { path: "roles",         name: "admin.roles",         component: () => import("@/modules/admin/views/RolesView.vue") },
            { path: "departments",   name: "admin.depts",         component: () => import("@/modules/admin/views/DepartmentsView.vue") },
            { path: "posts",         name: "admin.posts",         component: () => import("@/modules/admin/views/PostsView.vue") },
            { path: "settings",      name: "admin.settings",      component: () => import("@/modules/admin/views/SettingsView.vue") },
            { path: "apps",          name: "admin.apps",          component: () => import("@/modules/admin/views/AppsView.vue") },
            { path: "audit",         name: "admin.audit",         component: () => import("@/modules/admin/views/AuditView.vue") },
            { path: "dict",          name: "admin.dict",          component: () => import("@/modules/admin/views/DictView.vue") },
            { path: "notices",       name: "admin.notices",       component: () => import("@/modules/admin/views/NoticeView.vue") },
            { path: "logs",          name: "admin.logs",          component: () => import("@/modules/admin/views/LogsView.vue") },
            { path: "online",        name: "admin.online",        component: () => import("@/modules/admin/views/OnlineUsersView.vue") },
            // Platform-level pages moved to the Platform Console (/platform/*).
            { path: "users",            redirect: "/platform/users" },
            { path: "organizations",    redirect: "/platform/organizations" },
            { path: "platform/tenants", redirect: "/platform/organizations" },
          ],
        },

        // Platform Console — a left-rail module inside the app shell (NOT a
        // standalone page). Global platform_admin only; no tenant context needed.
        {
          path: "platform",
          component: PlatformLayout,
          beforeEnter: requirePlatformAdmin,
          children: [
            { path: "",              name: "platform.home",          component: () => import("@/modules/platform/views/PlatformHome.vue") },
            { path: "organizations", name: "platform.organizations", component: () => import("@/modules/admin/views/PlatformTenantsView.vue") },
            { path: "users",         name: "platform.users",         component: () => import("@/modules/admin/views/UsersView.vue") },
            { path: "registrations", name: "platform.registrations", component: () => import("@/modules/admin/views/RegistrationsView.vue"), props: { scope: "platform" } },
            { path: "rbac",  name: "platform.rbac",  component: () => import("@/modules/platform/views/RbacView.vue") },
            { path: "menus", name: "platform.menus", component: () => import("@/modules/platform/views/MenuCatalogView.vue") },
            // System pages (P3): dict / params / notices / logs / online / monitor / cron
            { path: "dict",    name: "platform.dict",    component: () => import("@/modules/platform/views/PlatformDictView.vue") },
            { path: "params",  name: "platform.params",  component: () => import("@/modules/platform/views/ParamsView.vue") },
            { path: "notices", name: "platform.notices", component: () => import("@/modules/platform/views/PlatformNoticeView.vue") },
            { path: "logs",    name: "platform.logs",    component: () => import("@/modules/platform/views/PlatformLogsView.vue") },
            { path: "online",  name: "platform.online",  component: () => import("@/modules/platform/views/PlatformOnlineView.vue") },
            { path: "monitor", name: "platform.monitor", component: () => import("@/modules/platform/views/MonitorView.vue") },
            { path: "cron",    name: "platform.cron",    component: () => import("@/modules/platform/views/CronView.vue") },
          ],
        },

        // Personal center — its own layout, separate from admin
        {
          path: "me",
          component: MeLayout,
          children: [
            { path: "",         redirect: "/me/profile" },
            { path: "profile",  name: "me.profile",  component: () => import("@/modules/me/views/ProfileView.vue") },
            { path: "security", name: "me.security", component: () => import("@/modules/me/views/SecurityView.vue") },
            { path: "sessions", name: "me.sessions", component: () => import("@/modules/me/views/SessionsView.vue") },
            { path: "tenants",  name: "me.tenants",  component: () => import("@/modules/me/views/TenantsView.vue") },
            // Back-compat
            { path: "settings", redirect: "/me/profile" },
          ],
        },
      ],
    },
    // Catch-all 404 — MUST stay last.
    {
      path: "/:pathMatch(.*)*",
      name: "not-found",
      component: () => import("@/shell/views/NotFound.vue"),
    },
  ],
});

// Global guard: force a must-change user to the security page on EVERY navigation
// (the parent route's beforeEnter only fires on subtree entry, not between siblings).
// Runs restore() first so a hard reload also enforces it. Backend PasswordChangeGate
// is the authoritative block; this is the UX redirect.
router.beforeEach(async (to) => {
  const auth = useAuthStore();
  if (auth.loggedIn && !auth.restored) await auth.restore();
  if (
    auth.loggedIn &&
    auth.user?.password_must_change &&
    !to.path.startsWith("/me/security") &&
    to.path !== "/login" &&
    to.path !== "/register"
  ) {
    return { path: "/me/security", query: { forced: "1" } };
  }
  return true;
});

// Track recently-opened apps: any nav into /<moduleCode>/... bumps the stack.
const moduleCodes = new Set(
  Object.values(moduleManifests)
    .map((m) => m.manifest.code)
    .filter((c) => c !== "admin"),
);
router.afterEach((to) => {
  const first = to.path.split("/").filter(Boolean)[0];
  if (first && moduleCodes.has(first)) bumpRecent(first);
});

if (import.meta.env.DEV) {
  console.info("[router] auto-mounted business modules:", businessRoutes.map((r) => r.path));
}

export default router;
