import type { RouteRecordRaw } from "vue-router";

export const routes: RouteRecordRaw[] = [
  { path: "", name: "lcform.center", component: () => import("./views/FormCenterView.vue") },
  { path: "design", name: "lcform.design", component: () => import("./views/FormDesignView.vue") },
];
