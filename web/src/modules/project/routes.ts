import type { RouteRecordRaw } from "vue-router";

export const routes: RouteRecordRaw[] = [
  { path: "", name: "project.home", component: () => import("./views/ProjectsView.vue") },
  { path: ":id", name: "project.board", component: () => import("./views/ProjectBoardView.vue") },
];
