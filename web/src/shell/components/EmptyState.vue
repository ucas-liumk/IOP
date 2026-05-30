<template>
  <div class="empty-state">
    <div class="icon">
      <slot name="icon">{{ icon ?? "◌" }}</slot>
    </div>
    <div class="title">{{ title }}</div>
    <div v-if="sub" class="sub">{{ sub }}</div>
    <div v-if="$slots.actions" class="actions">
      <slot name="actions" />
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{ title: string; sub?: string; icon?: string }>();
</script>

<style scoped>
.empty-state {
  text-align: center;
  padding: 60px 20px;
  color: var(--text-3);
}
.icon {
  font-size: 42px;
  color: var(--text-4);
  margin-bottom: 12px;
}
.title {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-2);
}
.sub {
  font-size: 12.5px;
  color: var(--text-3);
  margin-top: 4px;
}
.actions {
  margin-top: 16px;
  display: flex;
  justify-content: center;
  gap: 8px;
}
</style>
