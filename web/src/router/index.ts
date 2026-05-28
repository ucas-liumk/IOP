import { createRouter, createWebHistory } from "vue-router";
import AppLayout from "@/shell/layout/AppLayout.vue";

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: "/",
      component: AppLayout,
      children: [
        {
          path: "",
          name: "workspace",
          component: () => import("@/shell/workspace/WorkspaceHome.vue"),
        },
      ],
    },
  ],
});

export default router;
