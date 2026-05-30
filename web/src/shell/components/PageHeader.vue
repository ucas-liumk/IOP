<template>
  <header class="page-header">
    <div class="title-block">
      <h1 class="title">
        <slot name="title">{{ title }}</slot>
      </h1>
      <p v-if="$slots.sub || sub" class="sub">
        <slot name="sub">{{ sub }}</slot>
      </p>
    </div>
    <div v-if="$slots.actions" class="actions">
      <slot name="actions" />
    </div>
  </header>
</template>

<script setup lang="ts">
defineProps<{ title?: string; sub?: string }>();
</script>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 20px;
  margin-bottom: 20px;
}
.title-block { min-width: 0; }
.title {
  font-size: 22px;
  font-weight: 700;
  letter-spacing: -0.01em;
  display: flex;
  align-items: center;
  gap: 10px;
}
.sub {
  font-size: 13px;
  color: var(--text-3);
  margin-top: 4px;
}
.actions {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-shrink: 0;
}
</style>
