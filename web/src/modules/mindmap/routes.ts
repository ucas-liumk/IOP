import type { RouteRecordRaw } from "vue-router";

// Module-owned routes. Auto-mounted by router/index.ts.
// Paths are relative — the prefix `/mindmap` comes from manifest.routePrefix.
export const routes: RouteRecordRaw[] = [
  { path: "", name: "mindmap.home", component: () => import("./views/MindmapListView.vue") },
  { path: ":id", name: "mindmap.edit", component: () => import("./views/MindmapEditView.vue") },
];
