import type { RouteRecordRaw } from "vue-router";

export const routes: RouteRecordRaw[] = [
  { path: "mine", name: "approval.mine", component: () => import("./views/MyApprovalsView.vue") },
  { path: "new", name: "approval.new", component: () => import("./views/NewApprovalView.vue") },
  { path: "forms", name: "approval.forms", component: () => import("./views/FormsView.vue") },
  { path: "instances/:id", name: "approval.detail", component: () => import("./views/ApprovalDetailView.vue") },
];
