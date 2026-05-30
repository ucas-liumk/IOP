import type { RouteRecordRaw } from "vue-router";

// Module-owned routes. Auto-mounted by router/index.ts.
// Paths are relative — the prefix `/okr` comes from manifest.routePrefix.
export const routes: RouteRecordRaw[] = [
  { path: "plans",   name: "okr.plans",   component: () => import("./views/PlansView.vue") },
  { path: "reports", name: "okr.reports", component: () => import("./views/ReportsView.vue") },
  { path: "rollup",  name: "okr.rollup",  component: () => import("./views/RollupView.vue") },
];
