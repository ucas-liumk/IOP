import type { RouteRecordRaw } from "vue-router";

// Module-owned routes. Auto-mounted by router/index.ts.
// Paths are relative — the prefix `/books` comes from manifest.routePrefix.
export const routes: RouteRecordRaw[] = [
  { path: "", name: "books.catalog", component: () => import("./views/BooksView.vue") },
  { path: "manage", name: "books.manage", component: () => import("./views/BooksManageView.vue") },
];
