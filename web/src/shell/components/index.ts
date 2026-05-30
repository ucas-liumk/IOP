// Public re-exports — modules import from "@/shell/components".
export { default as PageHeader } from "./PageHeader.vue";
export { default as StatCard } from "./StatCard.vue";
export { default as DataTable } from "./DataTable.vue";
export { default as Pagination } from "./Pagination.vue";
export { default as ImportDialog } from "./ImportDialog.vue";
export type { BulkResult, BulkRowError } from "./ImportDialog.vue";
export type { Column } from "./types";
export { default as EmptyState } from "./EmptyState.vue";
export { default as Toast } from "./Toast.vue";
export { default as ConfirmDialog } from "./ConfirmDialog.vue";
export { default as LoadingSpinner } from "./LoadingSpinner.vue";
export { default as SkeletonLoader } from "./SkeletonLoader.vue";
export { default as TreeView } from "./TreeView.vue";
export { default as DeptTreeManager } from "./DeptTreeManager.vue";
export type { DeptApi, DeptRow, DeptTreeRow, CreateDeptPayload, UpdateDeptPatch } from "./DeptTreeManager.vue";
