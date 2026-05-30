import type { RouteRecordRaw } from "vue-router";

export const routes: RouteRecordRaw[] = [
  { path: "", name: "news.feed", component: () => import("./views/NewsFeedView.vue") },
  { path: "manage", name: "news.manage", component: () => import("./views/NewsManageView.vue") },
];
