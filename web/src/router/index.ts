import { createRouter, createWebHistory } from "vue-router";
import AppLayout from "@/shell/layout/AppLayout.vue";
import AdminLayout from "@/modules/admin/AdminLayout.vue";
import { requireAuth } from "@/shell/auth/guard";

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
        // OKR business module
        { path: "okr/plans", name: "okr.plans",     component: () => import("@/modules/okr/views/PlansView.vue") },
        { path: "okr/reports", name: "okr.reports", component: () => import("@/modules/okr/views/ReportsView.vue") },
        { path: "okr/rollup", name: "okr.rollup",   component: () => import("@/modules/okr/views/RollupView.vue") },

        // Admin module (mounted under AppLayout but uses its own sidebar)
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

        // Personal settings (also rendered inside AdminLayout for the unified nav)
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

export default router;
