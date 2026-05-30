<template>
  <div class="status-page">
    <EmptyState title="无访问权限" sub="您没有访问该页面的权限。如需开通，请联系组织管理员。">
      <template #icon>
        <svg width="46" height="46" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
          <rect x="4" y="11" width="16" height="10" rx="2" />
          <path d="M8 11V7a4 4 0 0 1 8 0v4" />
          <circle cx="12" cy="16" r="1" />
        </svg>
      </template>
      <template #actions>
        <RouterLink to="/" class="btn btn-primary">返回工作台</RouterLink>
      </template>
    </EmptyState>
    <div class="code">403</div>
  </div>
</template>

<script setup lang="ts">
import { RouterLink } from "vue-router";
import { EmptyState } from "@/shell/components";
</script>

<style scoped>
.status-page {
  min-height: 70vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}
.status-page :deep(.btn) { text-decoration: none; cursor: pointer; }
.code {
  font-family: var(--ff-mono);
  font-size: 12px;
  letter-spacing: 2px;
  color: var(--text-4);
  margin-top: 8px;
}
</style>
