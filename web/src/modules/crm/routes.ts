import type { RouteRecordRaw } from "vue-router";

export const routes: RouteRecordRaw[] = [
  { path: "", name: "crm.home", component: () => import("./views/IndexView.vue") },
];
