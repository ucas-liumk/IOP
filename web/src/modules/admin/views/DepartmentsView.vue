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
  getDeptTree, listDepts, createDept, updateDept, setDeptStatus, deleteDept, moveDept,
  downloadDeptsCsv,
} from "../api/admin";

// Tenant-console dept adapter — drives the shared DeptTreeManager against this
// org's own /admin/depts* endpoints (tenant context resolved server-side).
const deptApi: DeptApi = {
  fetchTree: (p) => getDeptTree(p),
  fetchFlat: (p) => listDepts(p),
  create: (p) => createDept(p),
  update: (id, patch) => updateDept(id, patch),
  setStatus: (id, status, cascade) => setDeptStatus(id, status, cascade),
  remove: (id) => deleteDept(id),
  move: (id, parentId) => moveDept(id, parentId),
  exportCsv: (p) => downloadDeptsCsv(p),
};
</script>

<style scoped>
.admin-page { display: flex; flex-direction: column; gap: var(--sp-5); }
</style>
