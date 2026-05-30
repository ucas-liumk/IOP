<template>
  <div class="status-page">
    <EmptyState title="页面不存在" sub="您访问的页面已被移动或从未存在。请检查链接是否正确。">
      <template #icon>
        <svg width="46" height="46" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="11" cy="11" r="7" />
          <path d="m21 21-4.3-4.3" />
          <line x1="8" y1="11" x2="14" y2="11" />
        </svg>
      </template>
      <template #actions>
        <RouterLink to="/" class="btn btn-primary">返回工作台</RouterLink>
      </template>
    </EmptyState>
    <div class="code">404</div>
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
