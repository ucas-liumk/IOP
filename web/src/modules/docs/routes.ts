import type { RouteRecordRaw } from "vue-router";

export const routes: RouteRecordRaw[] = [
  { path: "", name: "docs.home", component: () => import("./views/DocsView.vue") },
];
