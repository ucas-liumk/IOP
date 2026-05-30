<template>
  <div class="skeleton" role="status" aria-label="加载中">
    <span
      v-for="i in lines"
      :key="i"
      class="sk-line"
      :style="{ height: h, width: lineWidth(i) }"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";

const props = withDefaults(
  defineProps<{
    /** Number of skeleton bars. */
    lines?: number;
    /** Height of each bar (number = px). */
    height?: number | string;
    /** Shrink the last line to look like a paragraph tail. */
    lastShort?: boolean;
  }>(),
  { lines: 3, height: 14, lastShort: true },
);

const h = computed(() =>
  typeof props.height === "number" ? `${props.height}px` : props.height,
);

function lineWidth(i: number): string {
  if (props.lastShort && i === props.lines && props.lines > 1) return "60%";
  return "100%";
}
</script>

<style scoped>
.skeleton {
  display: flex;
  flex-direction: column;
  gap: 10px;
  width: 100%;
}
.sk-line {
  display: block;
  border-radius: var(--r-sm);
  background: linear-gradient(
    90deg,
    var(--border-soft) 25%,
    var(--border) 37%,
    var(--border-soft) 63%
  );
  background-size: 400% 100%;
  animation: sk-shimmer 1.3s ease-in-out infinite;
}
@keyframes sk-shimmer {
  0% { background-position: 100% 50%; }
  100% { background-position: 0 50%; }
}
</style>
