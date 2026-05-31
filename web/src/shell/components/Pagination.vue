<template>
  <div class="pager" v-if="total > 0 || page > 1">
    <span class="pager-total">共 {{ total }} 条</span>

    <div class="pager-controls">
      <button
        class="pager-btn"
        type="button"
        :disabled="page <= 1"
        @click="go(page - 1)"
        aria-label="上一页"
      >
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round">
          <polyline points="15 18 9 12 15 6" />
        </svg>
      </button>

      <span class="pager-pages">{{ page }} / {{ totalPages }}</span>

      <button
        class="pager-btn"
        type="button"
        :disabled="page >= totalPages"
        @click="go(page + 1)"
        aria-label="下一页"
      >
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round">
          <polyline points="9 6 15 12 9 18" />
        </svg>
      </button>
    </div>

    <select class="pager-size" :value="pageSize" @change="onSizeChange">
      <option v-for="opt in sizeOptions" :key="opt" :value="opt">{{ opt }} 条/页</option>
    </select>

    <div class="pager-jump">
      跳至
      <input
        class="pager-jump-input"
        type="number"
        min="1"
        :max="totalPages"
        v-model.number="jumpTo"
        @keyup.enter="onJump"
      />
      页
      <button class="pager-btn pager-jump-go" type="button" @click="onJump">确定</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";

const props = withDefaults(
  defineProps<{
    total: number;
    page: number;
    pageSize: number;
    /** Page-size choices shown in the select. */
    sizeOptions?: number[];
  }>(),
  { sizeOptions: () => [10, 20, 50, 100] },
);

const emit = defineEmits<{
  (e: "update:page", value: number): void;
  (e: "update:pageSize", value: number): void;
}>();

const totalPages = computed(() =>
  Math.max(1, Math.ceil(props.total / Math.max(1, props.pageSize))),
);

const jumpTo = ref<number>(props.page);
watch(
  () => props.page,
  (p) => {
    jumpTo.value = p;
  },
);

function clamp(p: number): number {
  if (!Number.isFinite(p)) return props.page;
  return Math.min(totalPages.value, Math.max(1, Math.trunc(p)));
}

function go(p: number) {
  const next = clamp(p);
  if (next !== props.page) emit("update:page", next);
}

function onJump() {
  go(jumpTo.value);
}

function onSizeChange(e: Event) {
  const size = Number((e.target as HTMLSelectElement).value);
  if (size === props.pageSize) return;
  emit("update:pageSize", size);
  // Reset to page 1 on size change so the offset stays valid.
  if (props.page !== 1) emit("update:page", 1);
}
</script>

<style scoped>
.pager {
  display: flex;
  align-items: center;
  gap: 14px;
  flex-wrap: wrap;
  padding: 12px 4px 2px;
  font-size: 12.5px;
  color: var(--text-3);
}
.pager-total { white-space: nowrap; }
.pager-controls {
  display: flex;
  align-items: center;
  gap: 8px;
}
.pager-pages {
  min-width: 56px;
  text-align: center;
  color: var(--text-2);
  font-variant-numeric: tabular-nums;
}
.pager-btn {
  display: grid;
  place-items: center;
  height: 28px;
  min-width: 28px;
  padding: 0 6px;
  border: 1px solid var(--border);
  border-radius: var(--r-sm, 6px);
  background: var(--surface);
  color: var(--text-2);
  cursor: pointer;
  transition: background .12s, border-color .12s, color .12s;
}
.pager-btn:hover:not(:disabled) {
  background: var(--surface-2);
  border-color: var(--primary);
  color: var(--primary);
}
.pager-btn:disabled {
  opacity: .45;
  cursor: not-allowed;
}
.pager-size {
  height: 28px;
  border: 1px solid var(--border);
  border-radius: var(--r-sm, 6px);
  background: var(--surface);
  color: var(--text-2);
  font-size: 12.5px;
  padding: 0 6px;
  cursor: pointer;
}
.pager-jump {
  display: flex;
  align-items: center;
  gap: 4px;
  white-space: nowrap;
}
.pager-jump-input {
  width: 52px;
  height: 28px;
  border: 1px solid var(--border);
  border-radius: var(--r-sm, 6px);
  background: var(--surface);
  color: var(--text);
  text-align: center;
  font-size: 12.5px;
  padding: 0 4px;
}
.pager-jump-input::-webkit-outer-spin-button,
.pager-jump-input::-webkit-inner-spin-button { -webkit-appearance: none; margin: 0; }
.pager-jump-input { -moz-appearance: textfield; }
.pager-jump-go { font-size: 12px; }
@media (prefers-reduced-motion: reduce) {
  .pager-btn { transition: none; }
}
</style>
