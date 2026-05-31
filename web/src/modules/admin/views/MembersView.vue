<template>
  <section class="admin-page">
    <PageHeader title="用户管理" sub="本组织成员 · 按部门筛选、批量导入导出、分配角色 / 岗位 / 部门" />
    <MemberManager
      :api="api"
      template-url="/admin/members/template"
      import-url="/admin/members/import"
      write-perm="member:write"
    />
  </section>
</template>

<script setup lang="ts">
import { PageHeader, MemberManager } from "@/shell/components";
import { tenantMemberApi } from "../api/admin";

// Tenant console: members of the current tenant (resolved server-side from the
// tenant context). The shared MemberManager owns the whole UI; this view just
// binds the /admin/members* adapter.
const api = tenantMemberApi();
</script>

<style scoped>
.admin-page { display: flex; flex-direction: column; gap: var(--sp-5); }
</style>
