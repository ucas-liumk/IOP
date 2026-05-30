<template>
  <span
    class="spinner"
    :style="{ width: dim, height: dim, borderWidth: bw }"
    role="status"
    aria-label="加载中"
  />
</template>

<script setup lang="ts">
import { computed } from "vue";

const props = withDefaults(defineProps<{ size?: number | string }>(), {
  size: 18,
});

const dim = computed(() =>
  typeof props.size === "number" ? `${props.size}px` : props.size,
);
const bw = computed(() => {
  const n = typeof props.size === "number" ? props.size : parseInt(props.size, 10) || 18;
  return `${Math.max(2, Math.round(n / 9))}px`;
});
</script>

<style scoped>
.spinner {
  display: inline-block;
  box-sizing: border-box;
  border-style: solid;
  border-color: var(--border-strong);
  border-top-color: var(--primary);
  border-radius: 50%;
  animation: spin .7s linear infinite;
  vertical-align: middle;
}
@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
