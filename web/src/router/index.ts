import { createRouter, createWebHistory } from "vue-router";
import AppLayout from "@/shell/layout/AppLayout.vue";
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
        {
          path: "okr/plans",
          name: "okr.plans",
          component: () => import("@/modules/okr/views/PlansView.vue"),
        },
        {
          path: "okr/reports",
          name: "okr.reports",
          component: () => import("@/modules/okr/views/ReportsView.vue"),
        },
        {
          path: "okr/rollup",
          name: "okr.rollup",
          component: () => import("@/modules/okr/views/RollupView.vue"),
        },
      ],
    },
  ],
});

export default router;
