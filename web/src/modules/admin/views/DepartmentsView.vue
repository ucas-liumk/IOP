<template>
  <section class="admin-page">
    <PageHeader title="组织机构" sub="本组织的部门 / 处室层级结构" />
    <DeptTreeManager
      :api="deptApi"
      template-url="/admin/depts/template"
      import-url="/admin/depts/import"
      write-perm="dept:write"
    />
  </section>
</template>

<script setup lang="ts">
import { PageHeader, DeptTreeManager, type DeptApi } from "@/shell/components";
import {
  getDeptTree, listDepts, createDept, updateDept, deleteDept, moveDept,
  downloadDeptsCsv,
} from "../api/admin";

// Tenant-console dept adapter — drives the shared DeptTreeManager against this
// org's own /admin/depts* endpoints (tenant context resolved server-side).
const deptApi: DeptApi = {
  fetchTree: () => getDeptTree(),
  fetchFlat: () => listDepts(),
  create: (p) => createDept(p),
  update: (id, patch) => updateDept(id, patch),
  remove: (id) => deleteDept(id),
  move: (id, parentId) => moveDept(id, parentId),
  exportCsv: () => downloadDeptsCsv(),
};
</script>

<style scoped>
.admin-page { display: flex; flex-direction: column; gap: var(--sp-5); }
</style>
