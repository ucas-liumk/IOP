import type { RouteRecordRaw } from "vue-router";

export const routes: RouteRecordRaw[] = [
  { path: "", name: "tasks.home", component: () => import("./views/TasksView.vue") },
];
